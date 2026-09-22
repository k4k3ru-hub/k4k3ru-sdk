package markethub

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestListSourceFilterJSON(t *testing.T) {
	filter := &ListSourceFilter{LiquidityModels: []string{"order-book"}}
	for _, params := range []any{ListSymbolsParams{SourceFilter: filter}, ListVenuesParams{SourceFilter: filter}} {
		data, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != `{"sourceFilter":{"liquidityModels":["order-book"]}}` {
			t.Fatalf("unexpected JSON: %s", data)
		}
		var symbols ListSymbolsParams
		var venues ListVenuesParams
		if err := json.Unmarshal(data, &symbols); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &venues); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(symbols.SourceFilter, filter) || !reflect.DeepEqual(venues.SourceFilter, filter) {
			t.Fatal("filter lost during decoding")
		}
	}
}

func TestListSourceFilterValidationAndMatching(t *testing.T) {
	for _, filter := range []*ListSourceFilter{nil, {}, {LiquidityModels: []string{}}} {
		if err := filter.Validate(); err != nil {
			t.Fatal(err)
		}
		if !filter.MatchesLiquidityModel("amm") || !filter.MatchesLiquidityModel("order-book") {
			t.Fatal("empty filter excluded venues")
		}
	}
	for _, models := range [][]string{{"cex"}, {"amm", "amm"}, {""}} {
		if err := (&ListSourceFilter{LiquidityModels: models}).Validate(); err == nil {
			t.Fatalf("accepted %v", models)
		}
	}
	filter := &ListSourceFilter{LiquidityModels: []string{"order-book"}}
	if filter.MatchesLiquidityModel("amm") || filter.MatchesLiquidityModel("") || !filter.MatchesLiquidityModel("order-book") {
		t.Fatal("incorrect matching")
	}
}
