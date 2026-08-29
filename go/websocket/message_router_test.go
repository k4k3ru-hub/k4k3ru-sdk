package websocket

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestMessageRouterRoutesJSONRPCResponse(t *testing.T) {
	t.Parallel()

	router, tracker, _ := newTestMessageRouter(t)
	pending, err := tracker.register(json.RawMessage(`"abc"`))
	if err != nil {
		t.Fatalf("register() error = %v", err)
	}
	router.HandleMessage([]byte(`{"id":"abc","result":{"subscribed":true}}`))
	outcome := <-pending.outcome
	if outcome.err != nil || string(outcome.response.Result) != `{"subscribed":true}` {
		t.Fatalf("outcome = %#v", outcome)
	}
}

func TestMessageRouterRoutesAggregationEvent(t *testing.T) {
	t.Parallel()

	router, _, events := newTestMessageRouter(t)
	params := validAggregationParams()
	subscription, err := events.register(params)
	if err != nil {
		t.Fatalf("register() error = %v", err)
	}
	router.HandleMessage([]byte(`{"ac":"crypto","mt":"perp","s":"BTC/USDC","am":"composite-mid","p":"63004.4","svc":2,"ts":1}`))
	if result := <-subscription.events; result.Price != "63004.4" {
		t.Fatalf("event price = %q", result.Price)
	}
}

func TestMessageRouterReportsMalformedMessages(t *testing.T) {
	t.Parallel()

	router, _, _ := newTestMessageRouter(t)
	for _, message := range [][]byte{
		nil,
		[]byte(`{`),
		[]byte(`{"id":1}`),
		[]byte(`{"id":1,"result":{},"error":{"code":-1,"message":"failure"}}`),
		[]byte(`{}`),
	} {
		router.HandleMessage(message)
		err := <-router.errors
		if err == nil || !strings.Contains(err.Error(), "failed to route") {
			t.Fatalf("router error = %v", err)
		}
	}
}

func TestMessageRouterCloseFailsPendingRequests(t *testing.T) {
	t.Parallel()

	router, tracker, _ := newTestMessageRouter(t)
	pending, _ := tracker.register(json.RawMessage(`1`))
	router.HandleClose()
	if outcome := <-pending.outcome; !errors.Is(outcome.err, errWebSocketConnectionClosed) {
		t.Fatalf("outcome error = %v", outcome.err)
	}
}

func TestNewMessageRouterValidatesDependencies(t *testing.T) {
	t.Parallel()

	if _, err := newMessageRouter(nil, newAggregationEventRegistry()); err == nil || !strings.Contains(err.Error(), "request_tracker=null") {
		t.Fatalf("nil request tracker error = %v", err)
	}
	if _, err := newMessageRouter(newRequestTracker(), nil); err == nil || !strings.Contains(err.Error(), "event_registry=null") {
		t.Fatalf("nil event registry error = %v", err)
	}
}

func newTestMessageRouter(t *testing.T) (*messageRouter, *requestTracker, *aggregationEventRegistry) {
	t.Helper()
	requests := newRequestTracker()
	events := newAggregationEventRegistry()
	router, err := newMessageRouter(requests, events)
	if err != nil {
		t.Fatalf("newMessageRouter() error = %v", err)
	}
	return router, requests, events
}
