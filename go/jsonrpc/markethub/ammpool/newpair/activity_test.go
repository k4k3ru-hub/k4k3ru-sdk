package newpair

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestActivityWireCompatibility preserves optional fields, decimal precision and independent copies.
//
// Version:
//   - 2026-09-21: Added.
func TestActivityWireCompatibility(t *testing.T) {
	var legacy Pair
	if err := json.Unmarshal([]byte(`{"poolId":"pool"}`), &legacy); err != nil || legacy.Activity != nil {
		t.Fatal(legacy, err)
	}
	amount := "12345678901234567890.123456789012345678"
	a := &Activity{SwapCount: "9007199254740993", Token0ToToken1: ActivitySwaps{Count: "9007199254740993", Token0Amount: &amount}}
	raw, err := json.Marshal(Pair{Activity: a})
	if err != nil {
		t.Fatal(err)
	}
	var p Pair
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if *p.Activity.Token0ToToken1.Token0Amount != amount || p.Activity.SwapCount != a.SwapCount || p.Activity.Token0ToToken1.VolumeUSD != nil {
		t.Fatal("precision or null lost")
	}
	for _, key := range []string{"completeness", "historyIncomplete", "missingRanges", "_activityState"} {
		if strings.Contains(string(raw), key) {
			t.Fatal("unexpected status", key)
		}
	}
	c := CloneActivity(a)
	*c.Token0ToToken1.Token0Amount = "0"
	if *a.Token0ToToken1.Token0Amount != amount {
		t.Fatal("mutable alias")
	}
}
