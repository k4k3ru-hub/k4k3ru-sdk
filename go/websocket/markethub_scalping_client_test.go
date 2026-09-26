package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
)

func observationParams() dto.Params {
	return dto.Params{MarketType: "spot", Symbol: "SUI/USDC", WindowMS: 60000, Markets: []market.MarketTarget{{Venue: "cetus", Network: "testnet"}}}
}
func observationResult(at int64) dto.Result {
	return dto.Result{EvaluatedAt: at, Buy: []dto.MarketPrice{}, Sell: []dto.MarketPrice{}}
}

// TestMarketHubScalpingLifecycle verifies ACK races, latest-only delivery, duplicate requests and reconnect.
//
// Version:
//   - 2026-09-26: Added.
func TestMarketHubScalpingLifecycle(t *testing.T) {
	r := newMarketHubScalpingEvents()
	l, err := newSubscriptionLifecycle(&executionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	p := observationParams()
	key, err := p.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	c, err := newMarketHubScalpingClient(executionSender(func(_ context.Context, m rpc.Method, raw json.RawMessage) (*rpc.Response, error) {
		calls++
		if m == rpc.MethodMarketHubScalpingUnsubscribe {
			return &rpc.Response{Result: raw}, nil
		}
		for i := int64(1); i <= 20; i++ {
			if err := r.route(dto.SubscriptionEvent{SubscriptionKey: key, Snapshot: observationResult(i)}); err != nil {
				return nil, err
			}
		}
		b, err := json.Marshal(dto.SubscribeResult{SubscriptionKey: key, IntervalMS: 1000})
		return &rpc.Response{Result: b}, err
	}), r, l)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Subscribe(t.Context(), p)
	if err != nil {
		t.Fatal(err)
	}
	if v := <-s.Events(); v.EvaluatedAt != 20 {
		t.Fatal("latest snapshot not retained", v)
	}
	duplicate, err := c.Subscribe(t.Context(), p)
	if err != nil || duplicate != s || calls != 1 {
		t.Fatal("duplicate wire request", calls, err)
	}
	if err := r.route(dto.SubscriptionEvent{SubscriptionKey: key, Snapshot: observationResult(21)}); err != nil {
		t.Fatal(err)
	}
	r.terminate(key, app.OutOfTicks())
	if err := <-s.Errors(); !errors.Is(err, app.OutOfTicks()) {
		t.Fatal(err)
	}
	if _, ok := <-s.Events(); ok {
		t.Fatal("stale buffered snapshot survived termination")
	}
	next, err := c.Subscribe(t.Context(), p)
	if err != nil || next == s {
		t.Fatal("reconnect failed", err)
	}
	if err := c.Unsubscribe(t.Context(), s); err != nil {
		t.Fatal(err)
	}
	if r.active[key] != next {
		t.Fatal("old handle removed replacement")
	}
	if err := c.Unsubscribe(t.Context(), next); err != nil {
		t.Fatal(err)
	}
	if len(r.active) != 0 {
		t.Fatal("unsubscribe retained handle")
	}
}

// TestMarketHubScalpingComposition verifies separate TradeHub routing and keyed terminal errors.
//
// Version:
//   - 2026-09-26: Added.
func TestMarketHubScalpingComposition(t *testing.T) {
	m, err := newModule(t.Context(), validModuleConfig("wss://api.k4k3ru.com/"), validModuleDeps())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	if m.MarketHubScalping() == nil || m.MarketHubScalping().events != m.router.marketHubScalpingEvents || m.Scalping() == nil {
		t.Fatal("incomplete composition")
	}
	key, err := observationParams().SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	r := m.router.marketHubScalpingEvents
	s := &MarketHubScalpingSubscription{registry: r, key: key, events: make(chan dto.Result, 1), errors: make(chan error, 1)}
	r.active[key] = s
	if err := m.router.route([]byte(`{"e":"msc","data":{"subscriptionKey":"` + key + `","snapshot":{"evaluatedAt":1,"buy":[],"sell":[]}}}`)); err != nil {
		t.Fatal(err)
	}
	if len(s.events) != 1 || len(m.router.scalpingEvents.active) != 0 {
		t.Fatal("observation routed to execution client")
	}
	if err := m.router.route([]byte(`{"subscriptionKey":"` + key + `","error":{"code":"out_of_ticks"}}`)); err != nil {
		t.Fatal(err)
	}
	if !s.closed || len(s.events) != 0 || len(s.errors) != 1 {
		t.Fatal("termination did not invalidate stream")
	}
}

// TestMarketHubScalpingRejectsACK verifies a wrong interval cannot activate a subscription.
//
// Version:
//   - 2026-09-26: Added.
func TestMarketHubScalpingRejectsACK(t *testing.T) {
	r := newMarketHubScalpingEvents()
	l, err := newSubscriptionLifecycle(&executionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	key, err := observationParams().SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	c, err := newMarketHubScalpingClient(executionSender(func(context.Context, rpc.Method, json.RawMessage) (*rpc.Response, error) {
		return &rpc.Response{Result: json.RawMessage(`{"subscriptionKey":"` + key + `","intervalMs":0}`)}, nil
	}), r, l)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Subscribe(t.Context(), observationParams()); err == nil {
		t.Fatal("invalid ACK accepted")
	}
	if len(r.active) != 0 {
		t.Fatal("failed subscription retained")
	}
}
