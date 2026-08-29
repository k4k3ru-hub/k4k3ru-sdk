package aggregation

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestParamsJSON(t *testing.T) {
	t.Parallel()

	want := Params{
		AssetClass:      AssetClassCrypto,
		MarketType:      MarketTypePerp,
		Symbol:          Symbol("BTC/USDC"),
		AggregationMode: AggregationModeCompositeMid,
		SourceFilter: &AggregationSourceFilter{
			VenueCategories: []VenueCategory{VenueCategoryCEX, VenueCategoryDEX},
			LiquidityModels: []LiquidityModel{LiquidityModelOrderBook, LiquidityModelAMM},
			AMMPoolChains:   []Chain{ChainBase, ChainEthereum},
		},
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"assetClass":"crypto","marketType":"perp","symbol":"BTC/USDC","aggregationMode":"composite-mid","sourceFilter":{"venueCategories":["cex","dex"],"liquidityModels":["order-book","amm"],"ammPoolChains":["base","ethereum"]}}`
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

func TestParamsJSONRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	var params Params
	err := json.Unmarshal([]byte(`{"marketType":"spot","symbol":"BTC/USDC","unexpected":true}`), &params)
	if err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
		t.Fatalf("Unmarshal() error = %v, want invalid parameter", err)
	}
}

func TestParamsJSONOmitsOptionalFields(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(Params{MarketType: MarketTypeSpot, Symbol: Symbol("BTC/USDC")})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"marketType":"spot","symbol":"BTC/USDC"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}
}

func TestParamsNormalizeAppliesDefaultsWithoutMutatingSourceFilter(t *testing.T) {
	t.Parallel()

	venueCategories := []VenueCategory{" DEX ", " CEX "}
	params := Params{
		MarketType: " PERP ",
		Symbol:     " btc/usdc ",
		SourceFilter: &AggregationSourceFilter{
			VenueCategories: venueCategories,
		},
	}
	normalized := params.Normalize()
	if normalized.AssetClass != AssetClassCrypto || normalized.MarketType != MarketTypePerp || normalized.Symbol != "BTC/USDC" || normalized.AggregationMode != AggregationModeCompositeMid {
		t.Fatalf("Normalize() = %#v", normalized)
	}
	wantCategories := []VenueCategory{VenueCategoryCEX, VenueCategoryDEX}
	if normalized.SourceFilter == nil || !reflect.DeepEqual(normalized.SourceFilter.VenueCategories, wantCategories) {
		t.Fatalf("Normalize() source filter = %#v, want %#v", normalized.SourceFilter, wantCategories)
	}
	if !reflect.DeepEqual(venueCategories, []VenueCategory{" DEX ", " CEX "}) {
		t.Fatalf("Normalize() mutated input = %#v", venueCategories)
	}
}

func TestParamsNormalizeOmitsEmptySourceFilter(t *testing.T) {
	t.Parallel()

	normalized := (Params{SourceFilter: &AggregationSourceFilter{}}).Normalize()
	if normalized.SourceFilter != nil {
		t.Fatalf("Normalize() source filter = %#v, want nil", normalized.SourceFilter)
	}
}

func TestParamsValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []Params{
		{AssetClass: "stock", MarketType: MarketTypePerp, Symbol: "BTC/USDC"},
		{MarketType: "future", Symbol: "BTC/USDC"},
		{MarketType: MarketTypePerp},
		{MarketType: MarketTypePerp, Symbol: "BTC/USDC-TOO-LONG"},
		{MarketType: MarketTypePerp, Symbol: "BTC/USDC", AggregationMode: "vwap"},
		{MarketType: MarketTypeSpot, Symbol: "BTC/USDC", SourceFilter: &AggregationSourceFilter{VenueCategories: []VenueCategory{}}},
	}
	for i, params := range tests {
		if err := params.Validate(); err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
			t.Fatalf("Validate() error = %v, want invalid parameter for test_index=%d", err, i)
		}
	}
}

func TestParamsSubscriptionKeyNormalizesEquivalentSelections(t *testing.T) {
	t.Parallel()

	left := Params{
		MarketType: MarketTypeSpot,
		Symbol:     "BTC/USDT",
		SourceFilter: &AggregationSourceFilter{
			VenueCategories: []VenueCategory{VenueCategoryDEX, VenueCategoryCEX},
			LiquidityModels: []LiquidityModel{LiquidityModelOrderBook, LiquidityModelAMM},
			AMMPoolChains:   []Chain{ChainEthereum, ChainBase},
		},
	}
	right := Params{
		MarketType: " SPOT ",
		Symbol:     " btc/usdt ",
		SourceFilter: &AggregationSourceFilter{
			VenueCategories: []VenueCategory{" CEX ", " DEX "},
			LiquidityModels: []LiquidityModel{" AMM ", " ORDER-BOOK "},
			AMMPoolChains:   []Chain{" BASE ", " ETHEREUM "},
		},
	}
	leftKey, err := left.SubscriptionKey()
	if err != nil {
		t.Fatalf("SubscriptionKey() error = %v", err)
	}
	rightKey, err := right.SubscriptionKey()
	if err != nil {
		t.Fatalf("SubscriptionKey() error = %v", err)
	}
	if leftKey != rightKey {
		t.Fatalf("SubscriptionKey() differs: left=%q right=%q", leftKey, rightKey)
	}
	want := "MarketHub.Aggregation:ac=CRYPTO:mt=SPOT:s=BTC/USDT:am=COMPOSITE-MID:src=VC=CEX,DEX|LM=AMM,ORDER-BOOK|APC=BASE,ETHEREUM"
	if leftKey != want {
		t.Fatalf("SubscriptionKey() = %q, want %q", leftKey, want)
	}
}

func TestParamsSubscriptionKeyDefaultsToAllSources(t *testing.T) {
	t.Parallel()

	key, err := (Params{MarketType: MarketTypePerp, Symbol: "BTC/USDC"}).SubscriptionKey()
	if err != nil {
		t.Fatalf("SubscriptionKey() error = %v", err)
	}
	want := "MarketHub.Aggregation:ac=CRYPTO:mt=PERP:s=BTC/USDC:am=COMPOSITE-MID:src=*"
	if key != want {
		t.Fatalf("SubscriptionKey() = %q, want %q", key, want)
	}
}
