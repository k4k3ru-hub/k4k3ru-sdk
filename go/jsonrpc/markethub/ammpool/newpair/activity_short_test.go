package newpair

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestShortActivityCompatibility verifies nullable comparisons and independent clones.
//
// Version:
//   - 2026-09-28: Added.
func TestShortActivityCompatibility(t *testing.T) {
	value := "-0.000000000000000001"
	short := &ActivityShortWindow{ActivityWindow: ActivityWindow{SwapCount: "9007199254740993"},
		Comparison: &ActivityComparison{From: 1790596800000000, To: 1790597100000000,
			SwapCount: ActivityChange{Previous: &value, Delta: &value, ChangePercentage: &value},
			VolumeUSD: ActivityChange{Previous: &value, Delta: &value, ChangePercentage: &value}}}
	a := &Activity{Windows: &ActivityWindows{FiveMinutes: short, FifteenMinutes: short, OneHour: &ActivityWindow{SwapCount: "1"}}}
	c := CloneActivity(a)
	for _, metric := range []*ActivityChange{&c.Windows.FiveMinutes.Comparison.SwapCount, &c.Windows.FiveMinutes.Comparison.VolumeUSD} {
		for _, p := range []*string{metric.Previous, metric.Delta, metric.ChangePercentage} {
			*p = "0"
		}
	}
	c.Windows.FiveMinutes.SwapCount = "0"
	if *a.Windows.FiveMinutes.Comparison.VolumeUSD.Delta != value || *c.Windows.FifteenMinutes.Comparison.SwapCount.ChangePercentage != value || short.SwapCount != "9007199254740993" {
		t.Fatal("short periods or metrics share mutable storage")
	}
	short.Comparison = nil
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"comparison":null`) || strings.Contains(string(raw), `"ActivityWindow"`) {
		t.Fatal("short window JSON is not flat or lost null comparison")
	}
	var roundTrip Activity
	if err := json.Unmarshal(raw, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if roundTrip.Windows.FiveMinutes.SwapCount != short.SwapCount || roundTrip.Windows.FiveMinutes.Comparison != nil || roundTrip.Windows.TwentyFourHours != nil {
		t.Fatal("short values changed on round trip")
	}
	var legacy ActivityWindows
	if err := json.Unmarshal([]byte(`{"1h":null,"24h":null}`), &legacy); err != nil || legacy.FiveMinutes != nil || legacy.FifteenMinutes != nil {
		t.Fatal("legacy response fabricated short history", err)
	}
}
