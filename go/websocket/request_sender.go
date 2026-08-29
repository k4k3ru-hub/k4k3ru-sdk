package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	k4k3ruSDKAuthentication "github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKSignature "github.com/k4k3ru-hub/k4k3ru-sdk/go/signature"
)

type requestTransport interface {
	sendRaw(context.Context, []byte) error
}

type clock interface {
	now() time.Time
}

type nonceGenerator interface {
	generate() (string, error)
}

type requestIDGenerator interface {
	next() json.RawMessage
}

type systemClock struct{}

func (systemClock) now() time.Time { return time.Now() }

type secureNonceGenerator struct{}

func (secureNonceGenerator) generate() (string, error) { return k4k3ruSDKSignature.GenerateNonce() }

type atomicRequestIDGenerator struct{ value atomic.Uint64 }

func (g *atomicRequestIDGenerator) next() json.RawMessage {
	return json.RawMessage(strconv.FormatUint(g.value.Add(1), 10))
}

type requestSender struct {
	transport          requestTransport
	requests           *requestTracker
	credentialProvider k4k3ruSDKAuthentication.CredentialProvider
	clock              clock
	nonceGenerator     nonceGenerator
	requestIDGenerator requestIDGenerator
}

func newRequestSender(transport requestTransport, requests *requestTracker, provider k4k3ruSDKAuthentication.CredentialProvider, clock clock, nonceGenerator nonceGenerator, requestIDGenerator requestIDGenerator) (*requestSender, error) {
	if transport == nil {
		return nil, fmt.Errorf("failed to create websocket request sender: transport=null")
	}
	if requests == nil {
		return nil, fmt.Errorf("failed to create websocket request sender: request_tracker=null")
	}
	if provider == nil {
		return nil, fmt.Errorf("failed to create websocket request sender: credential_provider=null")
	}
	if clock == nil {
		return nil, fmt.Errorf("failed to create websocket request sender: clock=null")
	}
	if nonceGenerator == nil {
		return nil, fmt.Errorf("failed to create websocket request sender: nonce_generator=null")
	}
	if requestIDGenerator == nil {
		return nil, fmt.Errorf("failed to create websocket request sender: request_id_generator=null")
	}
	return &requestSender{transport: transport, requests: requests, credentialProvider: provider, clock: clock, nonceGenerator: nonceGenerator, requestIDGenerator: requestIDGenerator}, nil
}

func (s *requestSender) send(ctx context.Context, method k4k3ruSDKJSONRPC.Method, params json.RawMessage) (*k4k3ruSDKJSONRPC.Response, error) {
	if s == nil {
		return nil, fmt.Errorf("failed to send websocket json rpc request: request_sender=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to send websocket json rpc request: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to send websocket json rpc request: %w", err)
	}
	credential, err := s.credentialProvider.Credential(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to send websocket json rpc request: failed to get credential: %w", err)
	}
	nonce, err := s.nonceGenerator.generate()
	if err != nil {
		return nil, fmt.Errorf("failed to send websocket json rpc request: failed to generate nonce: %w", err)
	}
	auth, err := k4k3ruSDKAuthentication.SignRequest(method, params, credential, s.clock.now().Unix(), nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to send websocket json rpc request: %w", err)
	}
	request := k4k3ruSDKJSONRPC.Request{ID: s.requestIDGenerator.next(), Method: method, Params: params, Auth: auth}
	if err := request.Validate(); err != nil {
		return nil, fmt.Errorf("failed to send websocket json rpc request: %w", err)
	}
	pending, err := s.requests.register(request.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to send websocket json rpc request: %w", err)
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		s.requests.cancel(pending)
		return nil, fmt.Errorf("failed to send websocket json rpc request: failed to encode request: %w", err)
	}
	if err := s.transport.sendRaw(ctx, encoded); err != nil {
		s.requests.cancel(pending)
		return nil, fmt.Errorf("failed to send websocket json rpc request: %w", err)
	}
	select {
	case <-ctx.Done():
		s.requests.cancel(pending)
		return nil, fmt.Errorf("failed to send websocket json rpc request: %w", ctx.Err())
	case outcome := <-pending.outcome:
		if outcome.err != nil {
			return nil, fmt.Errorf("failed to send websocket json rpc request: %w", outcome.err)
		}
		return &outcome.response, nil
	}
}
