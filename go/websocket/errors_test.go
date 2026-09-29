package websocket

import (
	"errors"
	"fmt"
	"testing"

	execution "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
	scalping "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping"
)

// TestPublicSubscriptionErrors verifies recovery classifications survive wrapping.
//
// Version:
//   - 2026-09-30: Added.
func TestPublicSubscriptionErrors(t *testing.T) {
	if !errors.Is(fmt.Errorf("failed to observe test stream: %w", errWebSocketConnectionClosed), ErrConnectionClosed) {
		t.Fatal("lost public disconnect identity")
	}
	for _, pending := range []bool{false, true} {
		r := newScalpingEventRegistry()
		s := &ScalpingSubscription{registry: r, id: "scalp_test", key: "key", events: make(chan scalping.SubscriptionEvent, 1), errors: make(chan error, 1)}
		if pending {
			s.buffer = make([]scalping.SubscriptionEvent, 16)
			r.pending = s
			r.route(scalpingEvent("scalp_test", "key", 1))
		} else {
			r.active[s.id] = s
			s.events <- scalping.SubscriptionEvent{}
			r.deliver(s, scalpingEvent(s.id, s.key, 1))
		}
		if err := <-s.Errors(); !errors.Is(err, ErrSubscriptionOverflow) {
			t.Fatal("lost public overflow identity", err)
		}
	}
	registry := newExecutionEventRegistry()
	sub := &ExecutionSubscription{registry: registry, id: "exec_test", key: "key", events: make(chan execution.SubscriptionEvent, 1), errors: make(chan error, 1)}
	registry.active[sub.id] = sub
	registry.interrupt()
	if err := <-sub.Errors(); !errors.Is(err, ErrConnectionClosed) {
		t.Fatal("lost execution disconnect identity", err)
	}
	for _, pending := range []bool{false, true} {
		r := newExecutionEventRegistry()
		s := &ExecutionSubscription{registry: r, id: "exec_one", key: "key", events: make(chan execution.SubscriptionEvent, 1), errors: make(chan error, 1)}
		if pending {
			s.buffer = make([]execution.SubscriptionEvent, 16)
			s.key = ""
			r.active[s.id] = s
			r.route(executionEvent("key", 1, execution.ObservationStatusPending))
		} else {
			r.active[s.id] = s
			s.events <- execution.SubscriptionEvent{}
			r.deliver(s, executionEvent("key", 1, execution.ObservationStatusPending))
		}
		if err := <-s.Errors(); !errors.Is(err, ErrSubscriptionOverflow) {
			t.Fatal("lost execution overflow identity", err)
		}
	}
}
