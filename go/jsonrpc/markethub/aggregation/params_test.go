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
