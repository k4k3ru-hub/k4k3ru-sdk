package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
)

const maxResponseBytes = 2 << 20

// HTTPError retains the HTTP status and Retry-After header, and unwraps an
// application error when the server supplied one. The client never retries it.
type HTTPError struct {
	StatusCode int
	RetryAfter string
	cause      error
}

// Error returns the HTTP failure without the response body or request credentials.
//
// Version:
//   - 2026-10-11: Added.
func (e *HTTPError) Error() string {
	if e.cause != nil {
		return fmt.Errorf("failed to call http json rpc: %w: status_code=%d", e.cause, e.StatusCode).Error()
	}
	return fmt.Sprintf("failed to call http json rpc: status_code=%d", e.StatusCode)
}

// Unwrap returns the underlying application error for errors.Is and errors.As.
//
// Version:
//   - 2026-10-11: Added.
func (e *HTTPError) Unwrap() error { return e.cause }

type requestSender struct {
	endpoint    string
	http        HTTPDoer
	credentials authentication.CredentialProvider
	now         func() time.Time
	nonce       func() (string, error)
	id          atomic.Uint64
}

func (s *requestSender) call(ctx context.Context, method jsonrpc.Method, params json.RawMessage) (result json.RawMessage, err error) {
	if ctx == nil {
		return nil, fmt.Errorf("failed to call http json rpc: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to call http json rpc: %w", err)
	}
	request := jsonrpc.Request{ID: json.RawMessage(strconv.FormatUint(s.id.Add(1), 10)), Method: method, Params: params}
	if s.credentials != nil {
		credential, err := s.credentials.Credential(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get http request credential: %w", err)
		}
		nonce, err := s.nonce()
		if err != nil {
			return nil, fmt.Errorf("failed to generate http request nonce: %w", err)
		}
		request.Auth, err = authentication.SignRequest(method, params, credential, s.now().Unix(), nonce)
		if err != nil {
			return nil, fmt.Errorf("failed to sign http request: %w", err)
		}
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to encode http request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	response, err := s.http.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to send http request: %w", err)
	}
	if response == nil || response.Body == nil {
		return nil, fmt.Errorf("failed to read http response: response_body=null")
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close http response: %w", closeErr))
			result = nil
		}
	}()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read http response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("failed to read http response: body=too_long max_length=%d", maxResponseBytes)
	}
	var rpcResponse jsonrpc.Response
	decodeErr := json.Unmarshal(body, &rpcResponse)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		httpErr := &HTTPError{StatusCode: response.StatusCode, RetryAfter: response.Header.Get("Retry-After")}
		if decodeErr != nil {
			httpErr.cause = fmt.Errorf("failed to decode http response: %w", decodeErr)
		} else if rpcResponse.Error != nil {
			httpErr.cause = rpcResponse.Error
		}
		return nil, httpErr
	}
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode http response: %w", decodeErr)
	}
	if !bytes.Equal(bytes.TrimSpace(rpcResponse.ID), request.ID) {
		return nil, fmt.Errorf("failed to validate http response: id=invalid")
	}
	if rpcResponse.Error != nil {
		if len(rpcResponse.Result) != 0 {
			return nil, fmt.Errorf("failed to validate http response: envelope=invalid")
		}
		return nil, fmt.Errorf("failed to call http json rpc: %w", rpcResponse.Error)
	}
	if len(rpcResponse.Result) == 0 || bytes.Equal(bytes.TrimSpace(rpcResponse.Result), []byte("null")) {
		return nil, fmt.Errorf("failed to validate http response: result=null")
	}
	return rpcResponse.Result, nil
}
