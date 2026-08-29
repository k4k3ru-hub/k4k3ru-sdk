package markethub

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestMarketHubListVenuesParamsJSON(t *testing.T) {
	t.Parallel()

	want := ListVenuesParams{}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `{}` {
		t.Fatalf("Marshal() = %s, want {}", data)
	}

	var got ListVenuesParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestMarketHubListVenuesResultJSON(t *testing.T) {
	t.Parallel()

	want := ListVenuesResult{
		Venues: []ListVenuesVenue{
			{Name: "binance", Status: "active", UpdatedAt: "2026-08-29T12:34:56Z"},
			{Name: "btse", Status: "failed", UpdatedAt: "2026-08-29T12:35:56Z"},
		},
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"venues":[{"name":"binance","status":"active","updatedAt":"2026-08-29T12:34:56Z"},{"name":"btse","status":"failed","updatedAt":"2026-08-29T12:35:56Z"}]}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got ListVenuesResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}
