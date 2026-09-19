package yield

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

// TestRewardAPRCompatibility verifies additive fields and legacy decoding.
//
// Version:
//   - 2026-09-20: Added.
func TestRewardAPRCompatibility(t *testing.T) {
	var legacy Pool
	if err := json.Unmarshal([]byte(`{"periods":[{"period":"24h","apr":{"fee":{"value":"0.3","status":"available"},"reward":{"value":"0","status":"available"},"total":{"value":"0.3","status":"available"}}}]}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.APR != nil || legacy.Periods[0].APR.Rewards != nil || legacy.Periods[0].APR.RewardsStatus != "" {
		t.Fatal("legacy fields invented")
	}
	for _, raw := range []string{
		`{"apr":{"fee":{"value":null,"status":"unavailable"},"reward":{"value":null,"status":"unavailable"},"total":{"value":"1.1","status":"available"},"rewards":[{"tokenId":"token","apr":{"value":"0.2","status":"available"}}],"rewardsStatus":"complete"}}`,
		`{"apr":{"rewards":[],"rewardsStatus":"complete"}}`,
		`{"apr":{"rewards":null,"rewardsStatus":"unknown"}}`,
	} {
		var p, again Pool
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(encoded, []byte(`"attribution"`)) {
			t.Fatal("removed field serialized")
		}
		if err := json.Unmarshal(encoded, &again); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(p, again) {
			t.Fatal("round trip changed reward semantics")
		}
		if len(again.Periods) != 0 {
			t.Fatal("unspecified APR copied into periods")
		}
	}
}
