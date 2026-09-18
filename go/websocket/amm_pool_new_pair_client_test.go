package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool/newpair"
)

type ammPoolNewPairSender struct {
	methods []rpc.Method
	ack     *dto.Params
	err     error
}

func (s *ammPoolNewPairSender) send(_ context.Context, method rpc.Method, raw json.RawMessage) (*rpc.Response, error) {
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

// TestAMMPoolNewPairClientRoutingAndCleanup verifies selector isolation, latest results, and unsubscribe cleanup.
//
// Version:
//   - 2026-09-19: Verify excluded pairs in replacement snapshots.
//   - 2026-09-16: Support AMM pool new pair monitoring.
//   - 2026-09-11: Added.
func TestAMMPoolNewPairClientRoutingAndCleanup(t *testing.T) {
	registry := newAMMPoolNewPairEventRegistry()
	sender := &ammPoolNewPairSender{}
	lifecycle, err := newSubscriptionLifecycle(&fakeSubscriptionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := newAMMPoolNewPairClient(sender, lifecycle, registry)
	if err != nil {
		t.Fatal(err)
	}
	router, err := newMessageRouter(newRequestTracker(), newBBOEventRegistry(), newOrderBookEventRegistry(), newSpreadEventRegistry(), newCarryEventRegistry(), newAMMPoolEventRegistry(), newAMMPoolNewPairEventRegistry())
	if err != nil {
		t.Fatal(err)
	}
	router.ammPoolNewPairEvents = registry
	first, err := client.Subscribe(context.Background(), dto.Params{Chain: " base ", Network: "sepolia"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := client.Subscribe(context.Background(), dto.Params{Chain: "base", Network: "mainnet"})
	if err != nil {
		t.Fatal(err)
	}
	if first.Params().Chain != "base" {
		t.Fatal(first.Params())
	}
	// Route the actual wire envelope, including exclusion data in the latest snapshot.
	for _, message := range []string{
		`{"e":"apnp","data":{"filter":{"chain":"base","network":"sepolia"},"epoch":"epoch","version":1,"pairs":[{"poolId":"pool","isListed":true}],"excludedPairs":[]}}`,
		`{"e":"apnp","data":{"filter":{"chain":"base","network":"sepolia"},"epoch":"epoch","version":2,"pairs":[],"excludedPairs":[{"poolId":"pool","isListed":false,"exclusionReason":"liquidity_stale","liquidityUsd":{"status":"known","value":"2000"},"liquidityEvaluatedAt":1789711411000000}]}}`,
	} {
		if err := router.route([]byte(message)); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case result := <-first.Events():
		if result.Version != 2 || len(result.Pairs) != 0 || len(result.ExcludedPairs) != 1 {
			t.Fatal(result)
		}
		excluded := result.ExcludedPairs[0]
		if excluded.IsListed || excluded.ExclusionReason != "liquidity_stale" || excluded.LiquidityUSD.Value != "2000" || excluded.LiquidityEvaluatedAt == nil {
			t.Fatal(excluded)
		}
	default:
		t.Fatal("missing latest replacement")
	}
	select {
	case <-second.Events():
		t.Fatal("network selectors crossed")
	default:
	}
	if err := client.Unsubscribe(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if _, open := <-first.Events(); open {
		t.Fatal("unsubscribed channel open")
	}
	if ok, err := registry.route(dto.Result{Filter: dto.Params{Chain: "base", Network: "mainnet"}}); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if err := client.Unsubscribe(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if len(sender.methods) != 4 || sender.methods[0] != rpc.MethodMarketHubAMMPoolNewPairSubscribe || sender.methods[2] != rpc.MethodMarketHubAMMPoolNewPairUnsubscribe {
		t.Fatal(sender.methods)
	}
}

// TestAMMPoolNewPairClientFailedACKCanRetry verifies failed acknowledgements permit a clean retry.
//
// Version:
//   - 2026-09-16: Support AMM pool new pair monitoring.
//   - 2026-09-11: Added.
func TestAMMPoolNewPairClientFailedACKCanRetry(t *testing.T) {
	registry := newAMMPoolNewPairEventRegistry()
	sentinel := errors.New("transport failed")
	sender := &ammPoolNewPairSender{err: sentinel}
	lifecycle, err := newSubscriptionLifecycle(&fakeSubscriptionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := newAMMPoolNewPairClient(sender, lifecycle, registry)
	if err != nil {
		t.Fatal(err)
	}
	p := dto.Params{Chain: "base", Network: "mainnet"}
	if _, err := client.Subscribe(context.Background(), p); !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	sender.err = nil
	sender.ack = &dto.Params{Chain: "base", Network: "sepolia"}
	if _, err := client.Subscribe(context.Background(), p); err == nil {
		t.Fatal("wrong network acknowledged")
	}
	sender.ack = nil
	sub, err := client.Subscribe(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Unsubscribe(context.Background(), sub); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Subscribe(context.Background(), dto.Params{Chain: "!invalid"}); err == nil {
		t.Fatal("invalid scope accepted")
	}
}
