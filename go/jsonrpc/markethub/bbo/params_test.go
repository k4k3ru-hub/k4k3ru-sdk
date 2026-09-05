package bbo

import (
	"encoding/json"
	"testing"
)

func TestParamsNormalizeAndSubscriptionKey(t *testing.T) {
	t.Parallel()

	params := Params{MarketType: " spot ", Symbol: " btc/usdc "}
	normalized := params.Normalize()
	if normalized.AssetClass != AssetClassCrypto || normalized.MarketType != MarketTypeSpot || normalized.Symbol != "BTC/USDC" {
		t.Fatalf("Normalize() = %#v", normalized)
	}
	key, err := params.SubscriptionKey()
	if err != nil {
		t.Fatalf("SubscriptionKey() error = %v", err)
	}
	if want := "MarketHub.BBO:ac=CRYPTO:mt=SPOT:s=BTC/USDC:src=*"; key != want {
		t.Fatalf("SubscriptionKey() = %q, want %q", key, want)
	}
}

func TestParamsJSONRejectsAggregationMode(t *testing.T) {
	t.Parallel()

	var params Params
	if err := json.Unmarshal([]byte(`{"marketType":"spot","symbol":"BTC/USDC","aggregationMode":"consolidated-bbo"}`), &params); err == nil {
		t.Fatal("Unmarshal() error = nil, want unknown-field error")
	}
}

func TestParamsJSONRejectsDepth(t *testing.T) {
	t.Parallel()

	var params Params
	if err := json.Unmarshal([]byte(`{"marketType":"spot","symbol":"BTC/USDC","depth":1}`), &params); err == nil {
		t.Fatal("Unmarshal() error = nil, want unknown-field error")
	}
}

func TestParamsValidateSourceFilter(t *testing.T) {
	t.Parallel()

	params := Params{
		MarketType: MarketTypeSpot,
		Symbol:     "BTC/USDC",
		SourceFilter: &SourceFilter{
			VenueCategories: []VenueCategory{VenueCategoryDEX},
			LiquidityModels: []LiquidityModel{LiquidityModelAMM},
			AMMPoolChains:   []Chain{ChainBase},
		},
	}
	if err := params.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
