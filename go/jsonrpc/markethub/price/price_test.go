package price

import (
	"encoding/json"
	"testing"
)

// TestPriceContract checks selectors, decimal values and unavailable serialization.
//
// Version:
//   - 2026-10-18: Added.
func TestPriceContract(t *testing.T) {
	for _, p := range []Params{{"sui", "spot"}, {"btc", "perpetual"}} {
		if err := p.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []Params{{"SUI", "spot"}, {"sui", "perp"}, {"../sui", "spot"}, {"sui", ""}} {
		if p.Validate() == nil {
			t.Fatalf("invalid selectors accepted: %+v", p)
		}
	}
	r := Result{Asset: "sui", MarketType: "spot", Definition: "composite", Currency: "USD", Method: "volume_weighted_latest", Interval: "60s", Status: StatusUnavailable, SentAt: 1792300000123456}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["price"] != nil || decoded["venueCount"] != float64(0) || decoded["sentAt"] != float64(r.SentAt) {
		t.Fatal(string(b))
	}
	for _, text := range []string{"0", "-1", "NaN", "1/2", "1e3", ""} {
		r.Price = &text
		r.Status = StatusLive
		r.VenueCount = 1
		if r.Validate() == nil {
			t.Fatalf("invalid decimal %q", text)
		}
	}
	p := "1.234567890123456789"
	r.Price = &p
	for _, status := range []Status{StatusLive, StatusStale} {
		r.Status = status
		if err := r.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	r.Status = StatusUnavailable
	if r.Validate() == nil {
		t.Fatal("unavailable price accepted")
	}
}
