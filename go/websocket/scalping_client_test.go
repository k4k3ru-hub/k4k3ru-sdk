package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping"
	event "github.com/k4k3ru-hub/k4k3ru-sdk/go/subscription"
)

func scalpingEvent(id, key string, sequence uint64) dto.SubscriptionEvent {
	return dto.SubscriptionEvent{ExecutionID: id, SubscriptionKey: key, Sequence: sequence, Kind: dto.EventKindError, Error: &dto.StreamError{Code: "evaluation_unavailable", Retryable: true}}
}

// TestScalpingACKRaceAndReconnect verifies early events, stale keys and explicit recovery.
//
// Version:
//   - 2026-09-24: Added.
func TestScalpingACKRaceAndReconnect(t *testing.T) {
	r := newScalpingEventRegistry()
	l, err := newSubscriptionLifecycle(&executionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	sender := executionSender(func(_ context.Context, method rpc.Method, raw json.RawMessage) (*rpc.Response, error) {
		if method == rpc.MethodTradeHubScalpingUnsubscribe {
			return &rpc.Response{Result: raw}, nil
		}
		r.route(scalpingEvent("scalp_one", "old", 99))
		r.route(scalpingEvent("scalp_other", "new", 10))
		r.route(scalpingEvent("scalp_one", "new", 1))
		return &rpc.Response{Result: json.RawMessage(`{"executionId":"scalp_one","subscriptionKey":"new"}`)}, nil
	})
	c, err := newScalpingClient(sender, r, l)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Subscribe(t.Context(), dto.SubscribeParams{ExecutionID: "scalp_one"})
	if err != nil {
		t.Fatal(err)
	}
	if e := <-s.Events(); e.Sequence != 1 || e.SubscriptionKey != "new" {
		t.Fatal(e)
	}
	r.route(scalpingEvent("scalp_one", "old", 100))
	r.route(scalpingEvent("scalp_one", "new", 1))
	r.route(scalpingEvent("scalp_one", "new", 2))
	if e := <-s.Events(); e.Sequence != 2 {
		t.Fatal(e)
	}
	if s.Reference().ExecutionID != "scalp_one" {
		t.Fatal("execution reference missing")
	}
	r.interrupt()
	if err := <-s.Errors(); !errors.Is(err, errWebSocketConnectionClosed) {
		t.Fatal(err)
	}
	if _, open := <-s.Events(); open {
		t.Fatal("disconnected handle retained")
	}
	resumed, err := c.Subscribe(t.Context(), dto.SubscribeParams{ExecutionID: s.Reference().ExecutionID})
	if err != nil || resumed == s {
		t.Fatalf("resume failed: %v", err)
	}
	if err := c.Unsubscribe(t.Context(), resumed); err != nil {
		t.Fatal(err)
	}
	if len(r.active) != 0 {
		t.Fatal("unsubscribed handle retained")
	}
}

// TestScalpingCompositionAndRouting verifies the owning constructor and typed event routing.
//
// Version:
//   - 2026-09-24: Added.
func TestScalpingCompositionAndRouting(t *testing.T) {
	m, err := newModule(t.Context(), validModuleConfig("wss://api.k4k3ru.com/"), validModuleDeps())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	if m.Scalping() == nil || m.Scalping().events != m.router.scalpingEvents || m.Scalping().lifecycle != m.subscriptions {
		t.Fatal("incomplete composition")
	}
	data, err := json.Marshal(scalpingEvent("scalp_one", "key", 1))
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(event.Event{Type: event.EventTypeScalping, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.router.route(wire); err != nil {
		t.Fatal(err)
	}
	if err := m.router.route([]byte(`{"e":"sc","data":{"subscriptionKey":"key"}}`)); err == nil {
		t.Fatal("invalid event accepted")
	}
}

// TestScalpingRejectsACKMismatchAndOverflow verifies delivery failures are visible.
//
// Version:
//   - 2026-09-24: Added.
func TestScalpingRejectsACKMismatchAndOverflow(t *testing.T) {
	for _, overflow := range []bool{false, true} {
		r := newScalpingEventRegistry()
		l, err := newSubscriptionLifecycle(&executionTransport{})
		if err != nil {
			t.Fatal(err)
		}
		c, err := newScalpingClient(executionSender(func(context.Context, rpc.Method, json.RawMessage) (*rpc.Response, error) {
			if overflow {
				for i := 1; i <= 17; i++ {
					r.route(scalpingEvent("scalp_one", "key", uint64(i)))
				}
			}
			return &rpc.Response{Result: json.RawMessage(`{"executionId":"wrong","subscriptionKey":"key"}`)}, nil
		}), r, l)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Subscribe(t.Context(), dto.SubscribeParams{ExecutionID: "scalp_one"}); err == nil {
			t.Fatal("bad acknowledgement accepted")
		}
		if r.pending != nil || len(r.active) != 0 {
			t.Fatal("failed request retained")
		}
	}
}
