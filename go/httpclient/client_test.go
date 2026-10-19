package httpclient

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/candles"
)

type doerFunc func(*http.Request) (*http.Response, error)

// Do delegates to an injected fake without opening a network connection.
//
// Version:
//   - 2026-10-11: Added.
func (f doerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

type credentialFunc func(context.Context) (authentication.Credential, error)

// Credential supplies the current test credential.
//
// Version:
//   - 2026-10-11: Added.
func (f credentialFunc) Credential(ctx context.Context) (authentication.Credential, error) {
	return f(ctx)
}

func testParams() candles.Params {
	return candles.Params{AssetType: "crypto", Asset: "btc", MarketType: "spot", From: 1791417600, To: 1791504000, Interval: "1m", Limit: 1}
}

func decodeTestRequest(t *testing.T, r *http.Request) jsonrpc.Request {
	t.Helper()
	if r.Method != http.MethodPost || r.URL.String() != "https://api.example.test/" || r.Header.Get("Content-Type") != "application/json" {
		t.Fatal("incorrect HTTP request metadata")
	}
	var request jsonrpc.Request
	err := json.NewDecoder(r.Body).Decode(&request)
	closeErr := r.Body.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("failed to decode test request: %v %v", err, closeErr)
	}
	if request.Method != jsonrpc.MethodMarketHubCandlesGet {
		t.Fatalf("unexpected operation: %q", request.Method)
	}
	return request
}

func testResponse(t *testing.T, request jsonrpc.Request, value any, rpcErr *apperror.AppError, status int) *http.Response {
	t.Helper()
	var result json.RawMessage
	if value != nil {
		var err error
		result, err = json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
	}
	body, err := json.Marshal(jsonrpc.Response{ID: request.ID, Result: result, Error: rpcErr})
	if err != nil {
		t.Fatal(err)
	}
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}
}

func emptyTestResult(t *testing.T, request jsonrpc.Request) candles.Result {
	t.Helper()
	var p candles.Params
	if err := json.Unmarshal(request.Params, &p); err != nil {
		t.Fatal(err)
	}
	return candles.Result{Definition: "composite", Asset: p.Asset, MarketType: p.MarketType, Interval: p.Interval, From: p.From, To: p.To, Candles: []candles.Candle{}, MissingMinutes: int((p.To - p.From) / 60)}
}

// TestNewClient composes every HTTP API group and forwards one anonymous call to the injected transport.
//
// Version:
//   - 2026-10-11: Added.
func TestNewClient(t *testing.T) {
	calls := 0
	doer := doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		request := decodeTestRequest(t, r)
		if request.Auth != nil || string(request.ID) != "1" {
			t.Fatal("anonymous request has authentication or incorrect ID")
		}
		return testResponse(t, request, emptyTestResult(t, request), nil, 200), nil
	})
	client, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doer})
	if err != nil || client.MarketHub() == nil || client.MarketHub().Candles() == nil || calls != 0 {
		t.Fatalf("constructor did not compose the client without I/O: %v", err)
	}
	result, err := client.MarketHub().Candles().Get(t.Context(), testParams())
	if err != nil || calls != 1 || result.To != testParams().To || result.MissingMinutes != 1440 {
		t.Fatalf("composed request failed: %+v %v", result, err)
	}
	for _, endpoint := range []string{"", "file:///tmp/test", "https://", "https://key:secret@api.example.test/", "https://api.example.test/?secret=secret", "https://api.example.test/#secret", "https://%secret"} {
		_, err := NewClient(Config{EndpointURL: endpoint, HTTPClient: doer})
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("invalid endpoint accepted or leaked: %v", err)
		}
	}
	if _, err := NewClient(Config{EndpointURL: "https://api.example.test/"}); err == nil {
		t.Fatal("missing transport accepted")
	}
}

// TestAuthenticatedCandles signs exact parameters with fresh credentials and a nonce per call.
//
// Version:
//   - 2026-10-11: Added.
func TestAuthenticatedCandles(t *testing.T) {
	for _, algorithm := range []authentication.SignatureAlgorithm{authentication.SignatureAlgorithmHMACSHA256, authentication.SignatureAlgorithmEd25519} {
		t.Run(string(algorithm), func(t *testing.T) {
			secret := "test-secret-only"
			if algorithm == authentication.SignatureAlgorithmEd25519 {
				secret = base64.RawURLEncoding.EncodeToString(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)))
			}
			now := time.Unix(1791610000, 0)
			credentialCalls, nonceCalls, httpCalls := 0, 0, 0
			config := Config{
				EndpointURL: "https://api.example.test/", Now: func() time.Time { return now },
				CredentialProvider: credentialFunc(func(context.Context) (authentication.Credential, error) {
					credentialCalls++
					return authentication.Credential{APIKey: fmt.Sprintf("test-key-%d", credentialCalls), SecretKey: secret, SignatureAlgorithm: algorithm}, nil
				}),
				Nonce: func() (string, error) { nonceCalls++; return fmt.Sprintf("test-nonce-%d", nonceCalls), nil },
				HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
					httpCalls++
					request := decodeTestRequest(t, r)
					credential := authentication.Credential{APIKey: fmt.Sprintf("test-key-%d", httpCalls), SecretKey: secret, SignatureAlgorithm: algorithm}
					want, err := authentication.SignRequest(jsonrpc.MethodMarketHubCandlesGet, request.Params, credential, now.Unix(), fmt.Sprintf("test-nonce-%d", httpCalls))
					if err != nil || !reflect.DeepEqual(request.Auth, want) || string(request.ID) != fmt.Sprint(httpCalls) {
						t.Fatalf("authentication composition mismatch: %v", err)
					}
					return testResponse(t, request, emptyTestResult(t, request), nil, 200), nil
				}),
			}
			client, err := NewClient(config)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				if _, err := client.MarketHub().Candles().Get(t.Context(), testParams()); err != nil {
					t.Fatal(err)
				}
			}
			if credentialCalls != 2 || nonceCalls != 2 || httpCalls != 2 {
				t.Fatal("requests reused authentication or retried implicitly")
			}
		})
	}
}

// TestHTTPFailures preserves application and transport errors without retrying billable calls.
//
// Version:
//   - 2026-10-11: Added.
func TestHTTPFailures(t *testing.T) {
	transportErr := errors.New("failed to reach test transport")
	for _, status := range []int{200, 429, 503, 0} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			appErr := apperror.OutOfTicks()
			if status == 429 {
				appErr = apperror.New("rate_limit_error", "")
			}
			client, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				request := decodeTestRequest(t, r)
				if status == 0 {
					return nil, transportErr
				}
				response := testResponse(t, request, nil, appErr, status)
				response.Header.Set("Retry-After", "60")
				return response, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.MarketHub().Candles().Get(t.Context(), testParams())
			if calls != 1 || (status == 0 && !errors.Is(err, transportErr)) || (status != 0 && !errors.Is(err, appErr)) {
				t.Fatalf("error chain or call count changed: calls=%d error=%v", calls, err)
			}
			if status >= 400 {
				var httpErr *HTTPError
				if !errors.As(err, &httpErr) || httpErr.StatusCode != status || httpErr.RetryAfter != "60" {
					t.Fatalf("HTTP metadata lost: %v", err)
				}
			}
		})
	}
}

// TestAuthenticationFailure rejects a failed credential provider without anonymous fallback or network I/O.
//
// Version:
//   - 2026-10-11: Added.
func TestAuthenticationFailure(t *testing.T) {
	want := errors.New("failed to obtain test credential")
	client, err := NewClient(Config{
		EndpointURL:        "https://api.example.test/",
		HTTPClient:         doerFunc(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected anonymous fallback"); return nil, nil }),
		CredentialProvider: credentialFunc(func(context.Context) (authentication.Credential, error) { return authentication.Credential{}, want }),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.MarketHub().Candles().Get(t.Context(), testParams()); !errors.Is(err, want) {
		t.Fatalf("credential error lost: %v", err)
	}
}
