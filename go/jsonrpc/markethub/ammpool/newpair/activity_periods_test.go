package newpair

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestActivityWindowsCompatibility preserves legacy payloads, exact numbers and null observations.
//
// Version:
//   - 2026-09-27: Added.
func TestActivityWindowsCompatibility(t *testing.T) {
	var legacy Activity
	if err := json.Unmarshal([]byte(`{"swapCount":"42"}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Windows != nil {
		t.Fatal("legacy payload invented period history")
	}
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"windows"`) {
		t.Fatal("legacy response acquired an empty period object")
	}
	raw = []byte(`{"swapCount":"9007199254740994","windows":{"1h":{"from":1790470800000000,"to":1790474400000000,"observedFrom":1790474100000000,"swapCount":"9007199254740993","token0ToToken1":{"count":"9007199254740993","token0Amount":"12345678901234567890.123456789012345678","token1Amount":null,"volumeUsd":null},"netToken0Liquidity":"-12345678901234567890.123456789012345678"},"24h":null}}`)
	var a Activity
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	w := a.Windows.OneHour
	if a.SwapCount != "9007199254740994" || w.SwapCount != "9007199254740993" || *w.Token0ToToken1.Token0Amount != "12345678901234567890.123456789012345678" || w.Token0ToToken1.VolumeUSD != nil || a.Windows.TwentyFourHours != nil {
		t.Fatal("period values changed lifetime totals, precision or null")
	}
	out, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"24h":null`) || !strings.Contains(string(out), `"netToken0Liquidity":"-12345678901234567890.123456789012345678"`) {
		t.Fatal("period round trip lost unknown or signed quantity")
	}
	for _, key := range []string{"completeness", "historyIncomplete", "missingRanges", "_activityState"} {
		if strings.Contains(string(out), key) {
			t.Fatalf("internal state exposed: %s", key)
		}
	}
}

// TestActivityWindowsCloneIsolation verifies published observations cannot mutate other windows or source state.
//
// Version:
//   - 2026-09-27: Added.
func TestActivityWindowsCloneIsolation(t *testing.T) {
	amount := "1.000000000000000001"
	w := &ActivityWindow{SwapCount: "7", Token0ToToken1: ActivitySwaps{Token0Amount: &amount, Token1Amount: &amount, VolumeUSD: &amount}, Token1ToToken0: ActivitySwaps{Token0Amount: &amount, Token1Amount: &amount, VolumeUSD: &amount}, LiquidityAdded: ActivityLiquidity{Token0Amount: &amount, Token1Amount: &amount}, LiquidityRemoved: ActivityLiquidity{Token0Amount: &amount, Token1Amount: &amount}, NetToken0Liquidity: &amount, NetToken1Liquidity: &amount}
	a := &Activity{Windows: &ActivityWindows{OneHour: w, TwentyFourHours: w}}
	c := CloneActivity(a)
	for _, p := range []*string{c.Windows.OneHour.Token0ToToken1.Token0Amount, c.Windows.OneHour.Token0ToToken1.Token1Amount, c.Windows.OneHour.Token0ToToken1.VolumeUSD, c.Windows.OneHour.Token1ToToken0.Token0Amount, c.Windows.OneHour.Token1ToToken0.Token1Amount, c.Windows.OneHour.Token1ToToken0.VolumeUSD, c.Windows.OneHour.LiquidityAdded.Token0Amount, c.Windows.OneHour.LiquidityAdded.Token1Amount, c.Windows.OneHour.LiquidityRemoved.Token0Amount, c.Windows.OneHour.LiquidityRemoved.Token1Amount, c.Windows.OneHour.NetToken0Liquidity, c.Windows.OneHour.NetToken1Liquidity} {
		*p = "0"
	}
	c.Windows.OneHour.SwapCount = "0"
	if *w.Token0ToToken1.Token0Amount != "1.000000000000000001" || *c.Windows.TwentyFourHours.NetToken1Liquidity != "1.000000000000000001" || w.SwapCount != "7" {
		t.Fatal("clone shared a period or quantity")
	}
	if CloneActivityWindows(nil) != nil || CloneActivityWindows(&ActivityWindows{}).OneHour != nil {
		t.Fatal("clone replaced unavailable periods")
	}
}
