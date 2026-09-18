package newpair

import (
	"encoding/json"
	"testing"
)

// TestBackfillAbandonmentWire preserves abandonment evidence and source status.
//
// Version:
//   - 2026-09-18: Added.
func TestBackfillAbandonmentWire(t *testing.T) {
	var result Result
	if err := json.Unmarshal([]byte(`{"pairs":[{"poolId":"pool","backfillAbandonedAt":1789711411000000}],"coverage":[{"status":"backfill-abandoned","reason":"backfill_retry_exhausted"}]}`), &result); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var copy Result
	if err := json.Unmarshal(raw, &copy); err != nil {
		t.Fatal(err)
	}
	if copy.Pairs[0].BackfillAbandonedAt == nil || *copy.Pairs[0].BackfillAbandonedAt != 1789711411000000 || copy.Coverage[0].Status != "backfill-abandoned" {
		t.Fatal(copy)
	}
	var old Pair
	if err := json.Unmarshal([]byte(`{"poolId":"old"}`), &old); err != nil || old.BackfillAbandonedAt != nil {
		t.Fatal(old, err)
	}
}
