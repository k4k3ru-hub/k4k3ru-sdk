package aggregation

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestResultJSON(t *testing.T) {
	t.Parallel()

	want := Result{
		AssetClass:       AssetClassCrypto,
		MarketType:       MarketTypePerp,
		Symbol:           Symbol("BTC/USDC"),
		AggregationMode:  AggregationModeCompositeMid,
		Price:            "100.25",
		SourceVenueCount: 2,
		Timestamp:        1786762800000000,
		SourceFilter: &AggregationSourceFilter{
			VenueCategories: []VenueCategory{VenueCategoryCEX},
			LiquidityModels: []LiquidityModel{LiquidityModelOrderBook},
		},
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"ac":"crypto","mt":"perp","s":"BTC/USDC","am":"composite-mid","p":"100.25","svc":2,"ts":1786762800000000,"sourceFilter":{"venueCategories":["cex"],"liquidityModels":["order-book"]}}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got Result
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}
