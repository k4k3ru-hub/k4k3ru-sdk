package orderbook

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestResultJSONDoesNotExposeVenues(t *testing.T) {
	t.Parallel()

	want := Result{
		AssetClass: AssetClassCrypto, MarketType: MarketTypeSpot, Symbol: "BTC/USDC", Depth: 3,
		AggregationMode: AggregationModeConsolidatedDepth,
		Bids:            []Level{{Price: "100", Quantity: "2.5"}}, Asks: []Level{{Price: "101", Quantity: "3.5"}},
		SourceVenueCount: 2, Version: 42, Timestamp: 1788512400000000,
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"ac":"crypto","mt":"spot","s":"BTC/USDC","d":3,"am":"consolidated-depth","b":[{"p":"100","q":"2.5"}],"a":[{"p":"101","q":"3.5"}],"svc":2,"v":42,"ts":1788512400000000}`
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
