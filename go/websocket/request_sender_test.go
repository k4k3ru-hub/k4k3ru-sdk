package websocket

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	k4k3ruSDKAuthentication "github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
)

func TestRequestSenderSignsSendsAndWaitsForResponse(t *testing.T) {
	t.Parallel()

	tracker := newRequestTracker()
	transport := &fakeRequestTransport{tracker: tracker}
	sender, err := newRequestSender(
		transport,
		tracker,
		staticCredentialProvider{credential: validSigningCredential()},
		fixedClock{value: time.Unix(1788019200, 0)},
		fixedNonceGenerator{value: "nonce"},
		&fixedRequestIDGenerator{value: json.RawMessage(`7`)},
	)
	if err != nil {
		t.Fatalf("newRequestSender() error = %v", err)
	}
	response, err := sender.send(context.Background(), k4k3ruSDKJSONRPC.MethodMarketHubAggregationSubscribe, json.RawMessage(`{"symbol":"BTC/USDC"}`))
	if err != nil {
		t.Fatalf("send() error = %v", err)
	}
	if response == nil || string(response.Result) != `{"ok":true}` {
		t.Fatalf("send() response = %#v", response)
	}
	if transport.request.Auth == nil || transport.request.Auth.APIKey != "api-key" || transport.request.Auth.Timestamp != 1788019200 || transport.request.Auth.Nonce != "nonce" || transport.request.Auth.Signature == "" {
		t.Fatalf("request auth = %#v", transport.request.Auth)
	}
}

func TestRequestSenderCancelsPendingRequestOnContextCancellation(t *testing.T) {
	t.Parallel()

	tracker := newRequestTracker()
	transport := &fakeRequestTransport{tracker: tracker, skipResponse: true}
	sender, _ := newRequestSender(transport, tracker, staticCredentialProvider{credential: validSigningCredential()}, fixedClock{value: time.Now()}, fixedNonceGenerator{value: "nonce"}, &fixedRequestIDGenerator{value: json.RawMessage(`8`)})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := sender.send(ctx, k4k3ruSDKJSONRPC.MethodMarketHubAggregationSubscribe, json.RawMessage(`{}`))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("send() error = %v", err)
	}
	resolved, resolveErr := tracker.resolve(k4k3ruSDKJSONRPC.Response{ID: json.RawMessage(`8`), Result: json.RawMessage(`{}`)})
	if resolveErr != nil || resolved {
		t.Fatalf("resolve after cancellation = %v, %v", resolved, resolveErr)
	}
}

type fakeRequestTransport struct {
	tracker      *requestTracker
	request      k4k3ruSDKJSONRPC.Request
	skipResponse bool
}

func (f *fakeRequestTransport) sendRaw(_ context.Context, payload []byte) error {
	if err := json.Unmarshal(payload, &f.request); err != nil {
		return err
	}
	if f.skipResponse {
		return nil
	}
	_, err := f.tracker.resolve(k4k3ruSDKJSONRPC.Response{ID: f.request.ID, Result: json.RawMessage(`{"ok":true}`)})
	return err
}

type staticCredentialProvider struct {
	credential k4k3ruSDKAuthentication.Credential
}

func (p staticCredentialProvider) Credential(context.Context) (k4k3ruSDKAuthentication.Credential, error) {
	return p.credential, nil
}

type fixedClock struct{ value time.Time }

func (c fixedClock) now() time.Time { return c.value }

type fixedNonceGenerator struct{ value string }

func (g fixedNonceGenerator) generate() (string, error) { return g.value, nil }

type fixedRequestIDGenerator struct{ value json.RawMessage }

func (g *fixedRequestIDGenerator) next() json.RawMessage {
	return append(json.RawMessage(nil), g.value...)
}

func validSigningCredential() k4k3ruSDKAuthentication.Credential {
	return k4k3ruSDKAuthentication.Credential{
		APIKey:             "api-key",
		SecretKey:          base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")),
		SignatureAlgorithm: k4k3ruSDKAuthentication.SignatureAlgorithmHMACSHA256,
	}
}
