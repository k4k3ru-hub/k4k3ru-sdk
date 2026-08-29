package websocket

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
)

func TestRequestTrackerResolvesMatchingResponse(t *testing.T) {
	t.Parallel()

	tracker := newRequestTracker()
	request, err := tracker.register(json.RawMessage(`"request-1"`))
	if err != nil {
		t.Fatalf("register() error = %v", err)
	}
	response := k4k3ruSDKJSONRPC.Response{ID: json.RawMessage(`"request-1"`), Result: json.RawMessage(`{"ok":true}`)}
	resolved, err := tracker.resolve(response)
	if err != nil {
		t.Fatalf("resolve() error = %v", err)
	}
	if !resolved {
		t.Fatal("resolve() = false")
	}
	outcome := <-request.outcome
	if outcome.err != nil || string(outcome.response.Result) != `{"ok":true}` {
		t.Fatalf("outcome = %#v", outcome)
	}
}

func TestRequestTrackerRejectsDuplicateAndInvalidIDs(t *testing.T) {
	t.Parallel()

	tracker := newRequestTracker()
	if _, err := tracker.register(json.RawMessage(`1`)); err != nil {
		t.Fatalf("register() error = %v", err)
	}
	if _, err := tracker.register(json.RawMessage(`1`)); err == nil || !strings.Contains(err.Error(), "request_id=duplicate") {
		t.Fatalf("duplicate error = %v", err)
	}
	for _, id := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`true`), json.RawMessage(`{}`)} {
		if _, err := tracker.register(id); err == nil {
			t.Fatalf("register(%q) error = nil", id)
		}
	}
}

func TestRequestTrackerIgnoresUnknownResponse(t *testing.T) {
	t.Parallel()

	resolved, err := newRequestTracker().resolve(k4k3ruSDKJSONRPC.Response{ID: json.RawMessage(`99`)})
	if err != nil {
		t.Fatalf("resolve() error = %v", err)
	}
	if resolved {
		t.Fatal("resolve() = true")
	}
}

func TestRequestTrackerFailsAllPendingRequests(t *testing.T) {
	t.Parallel()

	tracker := newRequestTracker()
	first, _ := tracker.register(json.RawMessage(`1`))
	second, _ := tracker.register(json.RawMessage(`"two"`))
	tracker.failAll(errWebSocketConnectionClosed)
	for _, request := range []*pendingRequest{first, second} {
		outcome := <-request.outcome
		if !errors.Is(outcome.err, errWebSocketConnectionClosed) {
			t.Fatalf("outcome error = %v", outcome.err)
		}
	}
	if replacement, err := tracker.register(json.RawMessage(`1`)); err != nil || replacement == nil {
		t.Fatalf("register after failAll = %#v, %v", replacement, err)
	}
}

func TestRequestTrackerCancelDoesNotRemoveReplacement(t *testing.T) {
	t.Parallel()

	tracker := newRequestTracker()
	request, _ := tracker.register(json.RawMessage(`1`))
	tracker.cancel(request)
	replacement, err := tracker.register(json.RawMessage(`1`))
	if err != nil {
		t.Fatalf("register replacement error = %v", err)
	}
	tracker.cancel(request)
	resolved, err := tracker.resolve(k4k3ruSDKJSONRPC.Response{ID: json.RawMessage(`1`)})
	if err != nil || !resolved {
		t.Fatalf("resolve() = %v, %v", resolved, err)
	}
	if _, ok := <-replacement.outcome; !ok {
		t.Fatal("replacement outcome closed without response")
	}
}
