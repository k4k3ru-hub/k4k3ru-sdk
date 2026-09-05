package websocket

import (
	"context"
	"encoding/json"
	"testing"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKJSONRPCBBO "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/bbo"
)

func TestBBOClientSubscribeAndUnsubscribe(t *testing.T) {
	t.Parallel()
	params := validBBOParams()
	sender := &fakeBBOJSONRPCSender{params: params}
	lifecycle, err := newSubscriptionLifecycle(&fakeSubscriptionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	registry := newBBOEventRegistry()
	client, err := newBBOClient(sender, lifecycle, registry)
	if err != nil {
		t.Fatal(err)
	}
	subscription, err := client.Subscribe(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if subscription.Params().Symbol != params.Symbol {
		t.Fatalf("Params() = %#v", subscription.Params())
	}
	if routed, err := registry.route(bboResult(params, "100")); err != nil || !routed {
		t.Fatalf("route() = %v, %v", routed, err)
	}
	if result := <-subscription.Events(); result.Bid.Price != "100" {
		t.Fatalf("bid price = %q", result.Bid.Price)
	}
	if err := client.Unsubscribe(context.Background(), subscription); err != nil {
		t.Fatal(err)
	}
	if len(sender.methods) != 2 || sender.methods[0] != k4k3ruSDKJSONRPC.MethodMarketHubBBOSubscribe || sender.methods[1] != k4k3ruSDKJSONRPC.MethodMarketHubBBOUnsubscribe {
		t.Fatalf("methods = %#v", sender.methods)
	}
}

type fakeBBOJSONRPCSender struct {
	params  k4k3ruSDKJSONRPCBBO.Params
	methods []k4k3ruSDKJSONRPC.Method
}

func (s *fakeBBOJSONRPCSender) send(_ context.Context, method k4k3ruSDKJSONRPC.Method, _ json.RawMessage) (*k4k3ruSDKJSONRPC.Response, error) {
	s.methods = append(s.methods, method)
	result, err := json.Marshal(s.params)
	if err != nil {
		return nil, err
	}
	return &k4k3ruSDKJSONRPC.Response{Result: result}, nil
}

func validBBOParams() k4k3ruSDKJSONRPCBBO.Params {
	return k4k3ruSDKJSONRPCBBO.Params{AssetClass: k4k3ruSDKJSONRPCBBO.AssetClassCrypto, MarketType: k4k3ruSDKJSONRPCBBO.MarketTypePerp, Symbol: "BTC/USDC", AggregationMode: k4k3ruSDKJSONRPCBBO.AggregationModeConsolidatedBBO}
}

func bboResult(params k4k3ruSDKJSONRPCBBO.Params, bidPrice string) k4k3ruSDKJSONRPCBBO.Result {
	return k4k3ruSDKJSONRPCBBO.Result{AssetClass: params.AssetClass, MarketType: params.MarketType, Symbol: params.Symbol, AggregationMode: params.AggregationMode, Bid: k4k3ruSDKJSONRPCBBO.Level{Price: bidPrice, Quantity: "1"}, Ask: k4k3ruSDKJSONRPCBBO.Level{Price: "101", Quantity: "1"}}
}
