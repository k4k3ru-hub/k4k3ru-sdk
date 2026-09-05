package websocket

import (
	"context"
	"encoding/json"
	"testing"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKJSONRPCOrderBook "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/orderbook"
)

func TestOrderBookClientSubscribeAndUnsubscribe(t *testing.T) {
	t.Parallel()
	params := validOrderBookParams()
	sender := &fakeOrderBookJSONRPCSender{params: params}
	lifecycle, err := newSubscriptionLifecycle(&fakeSubscriptionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	registry := newOrderBookEventRegistry()
	client, err := newOrderBookClient(sender, lifecycle, registry)
	if err != nil {
		t.Fatal(err)
	}
	subscription, err := client.Subscribe(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if subscription.Params().Depth != params.Depth {
		t.Fatalf("Params() = %#v", subscription.Params())
	}
	if routed, err := registry.route(orderBookResult(params, "100")); err != nil || !routed {
		t.Fatalf("route() = %v, %v", routed, err)
	}
	if result := <-subscription.Events(); result.Bids[0].Price != "100" {
		t.Fatalf("bid price = %q", result.Bids[0].Price)
	}
	if err := client.Unsubscribe(context.Background(), subscription); err != nil {
		t.Fatal(err)
	}
	if len(sender.methods) != 2 || sender.methods[0] != k4k3ruSDKJSONRPC.MethodMarketHubOrderBookSubscribe || sender.methods[1] != k4k3ruSDKJSONRPC.MethodMarketHubOrderBookUnsubscribe {
		t.Fatalf("methods = %#v", sender.methods)
	}
}

type fakeOrderBookJSONRPCSender struct {
	params  k4k3ruSDKJSONRPCOrderBook.Params
	methods []k4k3ruSDKJSONRPC.Method
}

func (s *fakeOrderBookJSONRPCSender) send(_ context.Context, method k4k3ruSDKJSONRPC.Method, _ json.RawMessage) (*k4k3ruSDKJSONRPC.Response, error) {
	s.methods = append(s.methods, method)
	result, err := json.Marshal(s.params)
	if err != nil {
		return nil, err
	}
	return &k4k3ruSDKJSONRPC.Response{Result: result}, nil
}

func validOrderBookParams() k4k3ruSDKJSONRPCOrderBook.Params {
	return k4k3ruSDKJSONRPCOrderBook.Params{
		AssetClass: k4k3ruSDKJSONRPCOrderBook.AssetClassCrypto,
		MarketType: k4k3ruSDKJSONRPCOrderBook.MarketTypeSpot,
		Symbol:     "BTC/USDC", Depth: 3,
	}
}

func orderBookResult(params k4k3ruSDKJSONRPCOrderBook.Params, bidPrice string) k4k3ruSDKJSONRPCOrderBook.Result {
	return k4k3ruSDKJSONRPCOrderBook.Result{
		AssetClass: params.AssetClass, MarketType: params.MarketType, Symbol: params.Symbol, Depth: params.Depth,
		Bids: []k4k3ruSDKJSONRPCOrderBook.Level{{Price: bidPrice, Quantity: "1"}},
		Asks: []k4k3ruSDKJSONRPCOrderBook.Level{{Price: "101", Quantity: "1"}},
	}
}
