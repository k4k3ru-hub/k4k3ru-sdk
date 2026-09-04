package orderbook

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestParamsNormalizeAppliesDefaults(t *testing.T) {
	t.Parallel()

	params := Params{MarketType: " SPOT ", Symbol: " btc/usdc "}
	got := params.Normalize()
	if got.AssetClass != AssetClassCrypto || got.MarketType != MarketTypeSpot || got.Symbol != "BTC/USDC" || got.Depth != 3 || got.AggregationMode != AggregationModeConsolidatedDepth {
		t.Fatalf("Normalize() = %#v", got)
	}
}

func TestParamsJSON(t *testing.T) {
	t.Parallel()

	want := Params{
		AssetClass: AssetClassCrypto, MarketType: MarketTypePerp, Symbol: "BTC/USDC", Depth: 10,
		AggregationMode: AggregationModeConsolidatedDepth,
		SourceFilter:    &SourceFilter{VenueCategories: []VenueCategory{VenueCategoryCEX}, LiquidityModels: []LiquidityModel{LiquidityModelOrderBook}},
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"assetClass":"crypto","marketType":"perp","symbol":"BTC/USDC","depth":10,"aggregationMode":"consolidated-depth","sourceFilter":{"venueCategories":["cex"],"liquidityModels":["order-book"]}}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}
	var got Params
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestParamsRejectsUnknownJSONField(t *testing.T) {
	t.Parallel()

	var params Params
	err := json.Unmarshal([]byte(`{"marketType":"spot","symbol":"BTC/USDC","venue":"binance"}`), &params)
	if err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
		t.Fatalf("Unmarshal() error = %v, want invalid parameter", err)
	}
}

func TestParamsValidateDepth(t *testing.T) {
	t.Parallel()

	for _, depth := range []uint16{1, 3, 20} {
		if err := (Params{MarketType: MarketTypeSpot, Symbol: "BTC/USDC", Depth: depth}).Validate(); err != nil {
			t.Fatalf("Validate() depth=%d error = %v", depth, err)
		}
	}
	err := (Params{MarketType: MarketTypeSpot, Symbol: "BTC/USDC", Depth: 21}).Validate()
	if err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
		t.Fatalf("Validate() error = %v, want invalid parameter", err)
	}
}

func TestParamsSubscriptionKeyIsVenueIndependent(t *testing.T) {
	t.Parallel()

	key, err := (Params{MarketType: MarketTypeSpot, Symbol: "BTC/USDC"}).SubscriptionKey()
	if err != nil {
		t.Fatalf("SubscriptionKey() error = %v", err)
	}
	want := "MarketHub.OrderBook:ac=CRYPTO:mt=SPOT:s=BTC/USDC:d=3:am=CONSOLIDATED-DEPTH:src=*"
	if key != want {
		t.Fatalf("SubscriptionKey() = %q, want %q", key, want)
	}
}
