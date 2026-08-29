package websocket

import (
	"strings"
	"testing"

	k4k3ruSDKJSONRPCAggregation "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/aggregation"
)

func TestAggregationEventRegistryRoutesLatestEvent(t *testing.T) {
	t.Parallel()

	registry := newAggregationEventRegistry()
	params := validAggregationParams()
	subscription, err := registry.register(params)
	if err != nil {
		t.Fatalf("register() error = %v", err)
	}
	first := aggregationResult(params, "100")
	second := aggregationResult(params, "101")
	for _, result := range []k4k3ruSDKJSONRPCAggregation.Result{first, second} {
		routed, err := registry.route(result)
		if err != nil || !routed {
			t.Fatalf("route() = %v, %v", routed, err)
		}
	}
	if result := <-subscription.events; result.Price != second.Price {
		t.Fatalf("event price = %q, want %q", result.Price, second.Price)
	}
}

func TestAggregationEventRegistryIgnoresUnknownSubscription(t *testing.T) {
	t.Parallel()

	params := validAggregationParams()
	routed, err := newAggregationEventRegistry().route(aggregationResult(params, "100"))
	if err != nil {
		t.Fatalf("route() error = %v", err)
	}
	if routed {
		t.Fatal("route() = true")
	}
}

func TestAggregationEventRegistryRejectsDuplicateAndInvalidSubscription(t *testing.T) {
	t.Parallel()

	registry := newAggregationEventRegistry()
	params := validAggregationParams()
	if _, err := registry.register(params); err != nil {
		t.Fatalf("register() error = %v", err)
	}
	if _, err := registry.register(params); err == nil || !strings.Contains(err.Error(), "subscription_key=duplicate") {
		t.Fatalf("duplicate error = %v", err)
	}
	if _, err := registry.register(k4k3ruSDKJSONRPCAggregation.Params{}); err == nil {
		t.Fatal("invalid registration error = nil")
	}
}

func TestAggregationEventRegistryUnregisterDoesNotRemoveReplacement(t *testing.T) {
	t.Parallel()

	registry := newAggregationEventRegistry()
	params := validAggregationParams()
	first, _ := registry.register(params)
	registry.unregister(first)
	replacement, err := registry.register(params)
	if err != nil {
		t.Fatalf("register replacement error = %v", err)
	}
	registry.unregister(first)
	routed, err := registry.route(aggregationResult(params, "100"))
	if err != nil || !routed {
		t.Fatalf("route() = %v, %v", routed, err)
	}
	if _, ok := <-replacement.events; !ok {
		t.Fatal("replacement events channel closed")
	}
}

func validAggregationParams() k4k3ruSDKJSONRPCAggregation.Params {
	return k4k3ruSDKJSONRPCAggregation.Params{
		AssetClass:      k4k3ruSDKJSONRPCAggregation.AssetClassCrypto,
		MarketType:      k4k3ruSDKJSONRPCAggregation.MarketTypePerp,
		Symbol:          k4k3ruSDKJSONRPCAggregation.Symbol("BTC/USDC"),
		AggregationMode: k4k3ruSDKJSONRPCAggregation.AggregationModeCompositeMid,
	}
}

func aggregationResult(params k4k3ruSDKJSONRPCAggregation.Params, price string) k4k3ruSDKJSONRPCAggregation.Result {
	return k4k3ruSDKJSONRPCAggregation.Result{
		AssetClass:       params.AssetClass,
		MarketType:       params.MarketType,
		Symbol:           params.Symbol,
		AggregationMode:  params.AggregationMode,
		Price:            price,
		SourceVenueCount: 2,
		Timestamp:        1,
		SourceFilter:     params.SourceFilter,
	}
}
