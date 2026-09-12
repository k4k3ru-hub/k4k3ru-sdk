package websocket

import (
	"context"
	"encoding/json"
	"testing"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKSpread "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/spread"
)

func TestSpreadClientSubscribeAndUnsubscribe(t *testing.T) {
	t.Parallel()
	params := k4k3ruSDKSpread.Params{Symbol: "BTC/USDC", BaseAsset: "BTC", Quantity: "0.1"}.Normalize()
	sender := &fakeSpreadJSONRPCSender{params: params}
	lifecycle, err := newSubscriptionLifecycle(&fakeSubscriptionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	registry := newSpreadEventRegistry()
	client, err := newSpreadClient(sender, lifecycle, registry)
	if err != nil {
		t.Fatal(err)
	}
	subscription, err := client.Subscribe(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	result := k4k3ruSDKSpread.Result{AssetClass: params.AssetClass, Symbol: params.Symbol, BaseAsset: params.BaseAsset, Quantity: params.Quantity, MinimumGrossSpreadBps: params.MinimumGrossSpreadBps, RouteFamilies: params.RouteFamilies}
	if routed, routeErr := registry.route(result); routeErr != nil || !routed {
		t.Fatalf("route() = %v, %v", routed, routeErr)
	}
	if received := <-subscription.Events(); received.Symbol != params.Symbol {
		t.Fatalf("event = %#v", received)
	}
	if err := client.Unsubscribe(context.Background(), subscription); err != nil {
		t.Fatal(err)
	}
	if len(sender.methods) != 2 || sender.methods[0] != k4k3ruSDKJSONRPC.MethodMarketHubSpreadSubscribe || sender.methods[1] != k4k3ruSDKJSONRPC.MethodMarketHubSpreadUnsubscribe {
		t.Fatalf("methods = %#v", sender.methods)
	}
}

type fakeSpreadJSONRPCSender struct {
	params  k4k3ruSDKSpread.Params
	methods []k4k3ruSDKJSONRPC.Method
}

func (s *fakeSpreadJSONRPCSender) send(_ context.Context, method k4k3ruSDKJSONRPC.Method, _ json.RawMessage) (*k4k3ruSDKJSONRPC.Response, error) {
	s.methods = append(s.methods, method)
	result, err := json.Marshal(s.params)
	if err != nil {
		return nil, err
	}
	return &k4k3ruSDKJSONRPC.Response{Result: result}, nil
}

// TestSpreadEventAgeRouting isolates subscriptions with different AMM age limits.
// Version:
//   - 2026-09-12: Added.
func TestSpreadEventAgeRouting(t *testing.T) {
	r := newSpreadEventRegistry()
	p := k4k3ruSDKSpread.Params{Symbol: "BTC/USDC", BaseAsset: "BTC", Quantity: "1"}.Normalize()
	short, err := r.register(p)
	if err != nil {
		t.Fatal(err)
	}
	defer r.unregister(short)
	p.MaxAgeSeconds = 60
	long, err := r.register(p)
	if err != nil {
		t.Fatal(err)
	}
	defer r.unregister(long)
	result := k4k3ruSDKSpread.Result{MaxAgeSeconds: 60, AssetClass: p.AssetClass, Symbol: p.Symbol, BaseAsset: p.BaseAsset, Quantity: p.Quantity, MinimumGrossSpreadBps: p.MinimumGrossSpreadBps, RouteFamilies: p.RouteFamilies}
	if routed, err := r.route(result); err != nil || !routed {
		t.Fatalf("route: %v %v", routed, err)
	}
	select {
	case <-long.events:
	default:
		t.Fatal("missing long-age event")
	}
	select {
	case <-short.events:
		t.Fatal("delivered to wrong age subscription")
	default:
	}
}
