package change

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// TestParameterDefaults verifies omitted periods and independent per-request slices.
//
// Version:
//   - 2026-10-19: Update required crypto asset selectors.
//   - 2026-10-18: Added.
func TestParameterDefaults(t *testing.T) {
	want := []Period{Period5m, Period15m, Period1h, Period4h, Period24h, Period7d}
	for _, raw := range []string{
		`{"assetType":"crypto","asset":"sui","marketType":"spot"}`,
		`{"assetType":"crypto","asset":"sui","marketType":"spot","periods":[]}`,
		`{"assetType":"crypto","asset":"sui","marketType":"spot","periods":null}`,
	} {
		var original Params
		if err := json.Unmarshal([]byte(raw), &original); err != nil {
			t.Fatal(err)
		}
		if err := original.Validate(); err != nil {
			t.Fatal(err)
		}
		value := original.Normalize()
		if len(original.Periods) != 0 || !slices.Equal(value.Periods, want) {
			t.Fatalf("incorrect defaults or mutated request: %+v", value)
		}
		value.Periods[0] = Period7d
		if !slices.Equal(original.Normalize().Periods, want) {
			t.Fatal("defaults share mutable storage")
		}
	}
	original := Params{AssetType: "crypto", Asset: "btc", MarketType: "perpetual", Periods: []Period{Period24h, Period5m}}
	value := original.Normalize()
	if !slices.Equal(value.Periods, original.Periods) {
		t.Fatal("normalization reordered selected periods")
	}
	value.Periods[0] = Period7d
	if original.Periods[0] != Period24h {
		t.Fatal("normalization shares the caller's period slice")
	}
}

// TestParameterValidation preserves invalid-parameter errors and rejects ambiguous selections.
//
// Version:
//   - 2026-10-19: Update required crypto asset selectors.
//   - 2026-10-18: Added.
func TestParameterValidation(t *testing.T) {
	for _, p := range []Params{
		{AssetType: "crypto", Asset: "sui", MarketType: "spot"},
		{AssetType: "crypto", Asset: "eth", MarketType: "perpetual", Periods: []Period{Period5m, Period7d}},
	} {
		if err := p.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []Params{
		{AssetType: "crypto", Asset: "", MarketType: "spot"},
		{AssetType: "crypto", Asset: "SUI", MarketType: "spot"},
		{AssetType: "crypto", Asset: "sui/usdc", MarketType: "spot"},
		{AssetType: "crypto", Asset: "sui", MarketType: "perp"},
		{AssetType: "crypto", Asset: "sui", MarketType: "future"},
		{AssetType: "crypto", Asset: "sui", MarketType: "spot", Periods: []Period{"1d"}},
		{AssetType: "crypto", Asset: "sui", MarketType: "spot", Periods: []Period{"365d"}},
		{AssetType: "crypto", Asset: "sui", MarketType: "spot", Periods: []Period{""}},
		{AssetType: "crypto", Asset: "sui", MarketType: "spot", Periods: []Period{Period5m, Period5m}},
		{AssetType: "crypto", Asset: "sui", MarketType: "spot", Periods: make([]Period, 7)},
	} {
		if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid parameters accepted or error chain lost: %+v %v", p, err)
		}
	}
	oversized := Params{AssetType: "crypto", Asset: "sui", MarketType: "spot", Periods: []Period{Period(strings.Repeat("sensitive", 1024))}}
	err := oversized.Validate()
	if err == nil || strings.Contains(err.Error(), "sensitive") {
		t.Fatal("invalid input echoed into an error")
	}
}

// TestPeriodDurations verifies the accepted periods are fixed elapsed durations.
//
// Version:
//   - 2026-10-18: Added.
func TestPeriodDurations(t *testing.T) {
	want := map[Period]time.Duration{
		Period5m: 5 * time.Minute, Period15m: 15 * time.Minute,
		Period1h: time.Hour, Period4h: 4 * time.Hour,
		Period24h: 24 * time.Hour, Period7d: 168 * time.Hour,
		"": 0, "1d": 0,
	}
	for period, duration := range want {
		if period.Duration() != duration {
			t.Fatalf("wrong duration: period=%q", period)
		}
	}
}
