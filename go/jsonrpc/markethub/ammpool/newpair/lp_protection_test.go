package newpair

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestLPProtectionJSON verifies LP protection behavior and boundary conditions.
//
// Version:
//   - 2026-09-23: Added.
func TestLPProtectionJSON(t *testing.T) {
	for _, raw := range []string{`{}`, `{"lpProtection":null}`} {
		var p Pair
		if err := json.Unmarshal([]byte(raw), &p); err != nil || p.LPProtection != nil {
			t.Fatalf("legacy: %v", err)
		}
	}
	raw := `{"status":"stale","reason":"source_disconnected","token0":{"lockedLiquidityPercentage":"0","permanentlyProtectedLiquidityPercentage":"0"},"token1":{"lockedLiquidityPercentage":null,"permanentlyProtectedLiquidityPercentage":null},"allPositionsProtected":false,"earliestUnlockAt":null,"canWeakenProtection":{"status":"not_applicable","value":null},"observedAt":1234,"position":{"kind":"block","number":"100","id":"0xabc","details":{"x":1}}}`
	var v LPProtection
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatal(err)
	}
	cloned := CloneLPProtection(&v)
	if !reflect.DeepEqual(&v, cloned) {
		t.Fatal("clone mismatch")
	}
	*cloned.Token0.LockedLiquidityPercentage = "100"
	*cloned.AllPositionsProtected = true
	*cloned.ObservedAt = 9999
	cloned.Position.Details[0] = ' '
	if *v.Token0.LockedLiquidityPercentage != "0" || *v.AllPositionsProtected || *v.ObservedAt != 1234 || v.Position.Details[0] != '{' {
		t.Fatal("shared mutable data")
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"allPositionsProtected":false`, `"lockedLiquidityPercentage":null`, `"earliestUnlockAt":null`, `"value":null`} {
		if !strings.Contains(string(b), field) {
			t.Fatalf("missing %s: %s", field, b)
		}
	}
	var round LPProtection
	if err := json.Unmarshal(b, &round); err != nil || !reflect.DeepEqual(v, round) {
		t.Fatalf("roundtrip: %v", err)
	}
}
