package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool"
)

type ammPoolSender struct {
	methods []rpc.Method
	ack     *dto.Params
	err     error
}

func (s *ammPoolSender) send(_ context.Context, method rpc.Method, raw json.RawMessage) (*rpc.Response, error) {
	s.methods = append(s.methods, method)
	if s.err != nil {
		return nil, s.err
	}
	if s.ack != nil {
		var err error
		raw, err = json.Marshal(s.ack)
		if err != nil {
			return nil, err
		}
	}
	return &rpc.Response{Result: raw}, nil
}

// TestAMMPoolClientRoutingAndCleanup verifies selector isolation, latest results, and unsubscribe cleanup.
//
// Version:
//   - 2026-09-11: Added.
func TestAMMPoolClientRoutingAndCleanup(t *testing.T) {
	registry := newAMMPoolEventRegistry()
	sender := &ammPoolSender{}
	lifecycle, err := newSubscriptionLifecycle(&fakeSubscriptionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := newAMMPoolClient(sender, lifecycle, registry)
	if err != nil {
		t.Fatal(err)
	}
	router, err := newMessageRouter(newRequestTracker(), newBBOEventRegistry(), newOrderBookEventRegistry(), newSpreadEventRegistry(), newCarryEventRegistry(), registry)
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.Subscribe(context.Background(), dto.Params{Symbol: " weth/usdc ", MaxAgeSeconds: 5})
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.Subscribe(context.Background(), dto.Params{Symbol: "WETH/USDC", MaxAgeSeconds: 30})
	if err != nil {
		t.Fatal(err)
	}
	if first.Params().Symbol != "WETH/USDC" {
		t.Fatal(first.Params())
	}
	// Route the actual wire envelope, including unavailable replacement snapshots.
	for _, message := range []string{
		`{"e":"ap","data":{"symbol":"WETH/USDC","maxAgeSeconds":5,"epoch":"epoch","version":1,"available":true,"compositeMid":"3000","pools":[]}}`,
		`{"e":"ap","data":{"symbol":"WETH/USDC","maxAgeSeconds":5,"epoch":"epoch","version":2,"available":false,"compositeMid":null,"reason":"no_eligible_pools","pools":[]}}`,
	} {
		if err := router.route([]byte(message)); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case result := <-first.Events():
		if result.Version != 2 || result.Available || result.CompositeMid != nil {
			t.Fatal(result)
		}
	default:
		t.Fatal("missing latest replacement")
	}
	select {
	case <-second.Events():
		t.Fatal("age selectors crossed")
	default:
	}
	if err := client.Unsubscribe(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if _, open := <-first.Events(); open {
		t.Fatal("unsubscribed channel open")
	}
	if ok, err := registry.route(dto.Result{Symbol: "WETH/USDC", MaxAgeSeconds: 30}); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if err := client.Unsubscribe(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if len(sender.methods) != 4 || sender.methods[0] != rpc.MethodMarketHubAMMPoolSubscribe || sender.methods[2] != rpc.MethodMarketHubAMMPoolUnsubscribe {
		t.Fatal(sender.methods)
	}
}

// TestAMMPoolClientFailedACKCanRetry verifies failed acknowledgements permit a clean retry.
//
// Version:
//   - 2026-09-11: Added.
func TestAMMPoolClientFailedACKCanRetry(t *testing.T) {
	registry := newAMMPoolEventRegistry()
	sentinel := errors.New("transport failed")
	sender := &ammPoolSender{err: sentinel}
	lifecycle, err := newSubscriptionLifecycle(&fakeSubscriptionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := newAMMPoolClient(sender, lifecycle, registry)
	if err != nil {
		t.Fatal(err)
	}
	p := dto.Params{Symbol: "WETH/USDC", MaxAgeSeconds: 30}
	if _, err := client.Subscribe(context.Background(), p); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	sender.err = nil
	sender.ack = &dto.Params{Symbol: "WETH/USDC", MaxAgeSeconds: 5}
	if _, err := client.Subscribe(context.Background(), p); err == nil {
		t.Fatal("wrong age acknowledged")
	}
	sender.ack = nil
	sub, err := client.Subscribe(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Unsubscribe(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Subscribe(context.Background(), dto.Params{Symbol: "WETH/USDC"}); err == nil {
		t.Fatal("missing age accepted")
	}
}
