package jsonrpc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMarketHubListSymbolsParamsJSON(t *testing.T) {
	t.Parallel()

	want := MarketHubListSymbolsParams{Venues: []MarketHubListSymbolsVenueParams{{
		Name:        "binance",
		Page:        2,
		Limit:       50,
		MarketTypes: []string{"spot", "perp"},
	}}}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"venues":[{"name":"binance","page":2,"limit":50,"marketTypes":["spot","perp"]}]}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got MarketHubListSymbolsParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestMarketHubListSymbolsParamsJSONOmitsEmptyVenues(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(MarketHubListSymbolsParams{})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `{}` {
		t.Fatalf("Marshal() = %s, want {}", data)
	}
}

func TestMarketHubListSymbolsResultJSON(t *testing.T) {
	t.Parallel()

	want := MarketHubListSymbolsResult{Venues: []MarketHubListSymbolsVenue{{
		Name:  "binance",
		Page:  1,
		Limit: 100,
		Total: 2,
		Symbols: []MarketHubListSymbolsSymbol{
			{Symbol: "BTC/USDT", MarketTypes: []string{"spot", "perp"}},
			{Symbol: "ETH/USDT", MarketTypes: []string{"spot"}},
		},
	}}}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"venues":[{"name":"binance","page":1,"limit":100,"total":2,"symbols":[{"symbol":"BTC/USDT","marketTypes":["spot","perp"]},{"symbol":"ETH/USDT","marketTypes":["spot"]}]}]}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got MarketHubListSymbolsResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}
