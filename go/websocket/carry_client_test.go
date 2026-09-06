package websocket

import (
	"context"
	"encoding/json"
	"testing"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKCarry "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/carry"
)

func TestCarryClientSubscribeAndUnsubscribe(t *testing.T) {
	t.Parallel()
	params := k4k3ruSDKCarry.Params{Symbol: "BTC/USDC", BaseAsset: "BTC", Quantity: "0.1", HoldingPeriodMinutes: 1440}.Normalize()
	sender := &fakeCarryJSONRPCSender{params: params}
	lifecycle, err := newSubscriptionLifecycle(&fakeSubscriptionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	registry := newCarryEventRegistry()
	client, err := newCarryClient(sender, lifecycle, registry)
	if err != nil {
		t.Fatal(err)
	}
	subscription, err := client.Subscribe(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	result := k4k3ruSDKCarry.Result{AssetClass: params.AssetClass, Symbol: params.Symbol, BaseAsset: params.BaseAsset, Quantity: params.Quantity, HoldingPeriodMinutes: params.HoldingPeriodMinutes, MinimumEstimatedFundingBps: params.MinimumEstimatedFundingBps, RouteFamilies: params.RouteFamilies}
	if routed, routeErr := registry.route(result); routeErr != nil || !routed {
		t.Fatalf("route() = %v, %v", routed, routeErr)
	}
	if received := <-subscription.Events(); received.Symbol != params.Symbol {
		t.Fatalf("event = %#v", received)
	}
	if err := client.Unsubscribe(context.Background(), subscription); err != nil {
		t.Fatal(err)
	}
	if len(sender.methods) != 2 || sender.methods[0] != k4k3ruSDKJSONRPC.MethodMarketHubCarrySubscribe || sender.methods[1] != k4k3ruSDKJSONRPC.MethodMarketHubCarryUnsubscribe {
		t.Fatalf("methods = %#v", sender.methods)
	}
}

type fakeCarryJSONRPCSender struct {
	params  k4k3ruSDKCarry.Params
	methods []k4k3ruSDKJSONRPC.Method
}

func (s *fakeCarryJSONRPCSender) send(_ context.Context, method k4k3ruSDKJSONRPC.Method, _ json.RawMessage) (*k4k3ruSDKJSONRPC.Response, error) {
	s.methods = append(s.methods, method)
	result, err := json.Marshal(s.params)
	if err != nil {
		return nil, err
	}
	return &k4k3ruSDKJSONRPC.Response{Result: result}, nil
}
