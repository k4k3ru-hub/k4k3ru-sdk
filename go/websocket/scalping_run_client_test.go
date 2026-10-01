package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping"
	event "github.com/k4k3ru-hub/k4k3ru-sdk/go/subscription"
)

func scalpingRunEvent(id, key string, sequence uint64) dto.RunEvent {
	return dto.RunEvent{ExecutionID: id, SubscriptionKey: key, Sequence: sequence, Kind: dto.EventKindError, Error: &dto.StreamError{Code: "evaluation_unavailable", Retryable: true}}
}

// TestScalpingRunACKRaceAndReconnect verifies early events, stale keys and explicit recovery.
//
// Version:
//   - 2026-10-01: Added.
func TestScalpingRunACKRaceAndReconnect(t *testing.T) {
	r := newScalpingRunEventRegistry()
	l, err := newSubscriptionLifecycle(&executionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	sender := executionSender(func(_ context.Context, method rpc.Method, raw json.RawMessage) (*rpc.Response, error) {
		if method == rpc.MethodTradeHubScalpingUnsubscribe {
			return &rpc.Response{Result: raw}, nil
		}
		r.route(scalpingRunEvent("scalp_one", "old", 99))
		r.route(scalpingRunEvent("scalp_other", "new", 10))
		r.route(scalpingRunEvent("scalp_one", "new", 1))
		return &rpc.Response{Result: scalpingRunACK(t, "scalp_one", "new")}, nil
	})
	c, err := newScalpingRunClient(sender, r, l)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Run(t.Context(), dto.RunParams{ExecutionID: "scalp_one"})
	if err != nil {
		t.Fatal(err)
	}
	if e := <-s.Events(); e.Sequence != 1 || e.SubscriptionKey != "new" {
		t.Fatal(e)
	}
	r.route(scalpingRunEvent("scalp_one", "old", 100))
	r.route(scalpingRunEvent("scalp_one", "new", 1))
	r.route(scalpingRunEvent("scalp_one", "new", 2))
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
	resumed, err := c.Run(t.Context(), dto.RunParams{ExecutionID: s.Reference().ExecutionID})
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

// TestScalpingRunCompositionAndRouting verifies the owning constructor and typed event routing.
//
// Version:
//   - 2026-10-01: Added.
func TestScalpingRunCompositionAndRouting(t *testing.T) {
	m, err := newModule(t.Context(), validModuleConfig("wss://api.k4k3ru.com/"), validModuleDeps())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	if m.Scalping() == nil || m.Scalping().run == nil || m.Scalping().run.events != m.router.scalpingRunEvents || m.Scalping().run.lifecycle != m.subscriptions {
		t.Fatal("incomplete composition")
	}
	data, err := json.Marshal(scalpingRunEvent("scalp_one", "key", 1))
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(event.Event{Type: event.EventTypeScalpingRun, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.router.route(wire); err != nil {
		t.Fatal(err)
	}
	if err := m.router.route([]byte(`{"e":"scr","data":{"subscriptionKey":"key"}}`)); err == nil {
		t.Fatal("invalid event accepted")
	}
}

// TestScalpingRunRejectsACKMismatchAndOverflow verifies delivery failures are visible.
//
// Version:
//   - 2026-10-01: Added.
func TestScalpingRunRejectsACKMismatchAndOverflow(t *testing.T) {
	for _, overflow := range []bool{false, true} {
		r := newScalpingRunEventRegistry()
		l, err := newSubscriptionLifecycle(&executionTransport{})
		if err != nil {
			t.Fatal(err)
		}
		c, err := newScalpingRunClient(executionSender(func(context.Context, rpc.Method, json.RawMessage) (*rpc.Response, error) {
			if overflow {
				for i := 1; i <= 17; i++ {
					r.route(scalpingRunEvent("scalp_one", "key", uint64(i)))
				}
			}
			return &rpc.Response{Result: scalpingRunACK(t, "wrong", "key")}, nil
		}), r, l)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Run(t.Context(), dto.RunParams{ExecutionID: "scalp_one"}); err == nil {
			t.Fatal("bad acknowledgement accepted")
		}
		if r.pending != nil || len(r.active) != 0 {
			t.Fatal("failed request retained")
		}
	}
}

func scalpingRunACK(t *testing.T, id, key string) json.RawMessage {
	t.Helper()
	data, err := os.ReadFile("../jsonrpc/tradehub/scalping/testdata/run_spot_buy.json")
	if err != nil {
		t.Fatal(err)
	}
	var p dto.RunParams
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(dto.RunResult{ExecutionID: id, SubscriptionKey: key, Params: p.Params})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestScalpingRunRejectsChangedStartACK verifies the server cannot silently change an execution account.
//
// Version:
//   - 2026-10-01: Added.
func TestScalpingRunRejectsChangedStartACK(t *testing.T) {
	data, err := os.ReadFile("../jsonrpc/tradehub/scalping/testdata/run_spot_buy.json")
	if err != nil {
		t.Fatal(err)
	}
	var p dto.RunParams
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	for _, changed := range []bool{false, true} {
		registry := newScalpingRunEventRegistry()
		lifecycle, err := newSubscriptionLifecycle(&executionTransport{})
		if err != nil {
			t.Fatal(err)
		}
		sender := executionSender(func(context.Context, rpc.Method, json.RawMessage) (*rpc.Response, error) {
			settings := p.Params.Normalize()
			if changed {
				settings.ExecutionRule.Markets[0].AccountAddress = "different-wallet"
			}
			raw, err := json.Marshal(dto.RunResult{ExecutionID: "run", SubscriptionKey: "key", Params: &settings})
			return &rpc.Response{Result: raw}, err
		})
		client, err := newScalpingRunClient(sender, registry, lifecycle)
		if err != nil {
			t.Fatal(err)
		}
		subscription, err := client.Run(t.Context(), p)
		if changed {
			if err == nil {
				t.Fatal("account changed without rejection")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		reference := subscription.Reference()
		reference.Params.ExecutionRule.Markets[0].AccountAddress = "mutated"
		if subscription.Reference().Params.ExecutionRule.Markets[0].AccountAddress == "mutated" {
			t.Fatal("reference aliases saved configuration")
		}
		registry.interrupt()
	}
}
