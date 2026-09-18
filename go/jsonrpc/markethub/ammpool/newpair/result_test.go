package newpair

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestObservationAndExclusionWire preserves the last evaluation and chain-neutral swap evidence.
//
// Version:
//   - 2026-09-19: Replace abandonment fields with observed swap and exclusion fields.
func TestObservationAndExclusionWire(t *testing.T) {
	raw := []byte(`{"epoch":"epoch","version":3,"pairs":[{"poolId":"listed","isListed":true,"swapObservedAt":1789711411000000,"swapObservedPosition":{"kind":"slot","number":"18446744073709551615","id":"position","transactionId":"tx","eventIndex":"3"},"confirmedAt":1789711412000000,"liquidityEvaluatedAt":1789711413000000,"liquidityUsd":{"status":"known","value":"2000"},"liquidityMethod":"lp_principal"}],"excludedPairs":[{"poolId":"expired","isListed":false,"exclusionReason":"liquidity_stale","liquidityUsd":{"status":"known","value":"2000"},"liquidityEvaluatedAt":1789711400000000}],"coverage":[{"status":"live"}]}`)
	var result Result
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	listed := result.Pairs[0]
	excluded := result.ExcludedPairs[0]
	if !listed.IsListed || listed.SwapObservedAt == nil || *listed.SwapObservedAt != 1789711411000000 || listed.SwapObservedPosition == nil || listed.SwapObservedPosition.Number != "18446744073709551615" || listed.ConfirmedAt == nil {
		t.Fatal(listed)
	}
	if excluded.IsListed || excluded.ExclusionReason != "liquidity_stale" || excluded.LiquidityUSD.Value != "2000" || excluded.LiquidityEvaluatedAt == nil {
		t.Fatal(excluded)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip Result
	if err := json.Unmarshal(encoded, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, roundtrip) {
		t.Fatalf("round trip changed evidence: %s", encoded)
	}
	var fields map[string]json.RawMessage
	pairRaw, err := json.Marshal(listed)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(pairRaw, &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"firstSwapAt", "firstLiquidityAt", "backfillAbandonedAt", "exclusionReason"} {
		if _, ok := fields[key]; ok {
			t.Fatal("unexpected field", key)
		}
	}
}

// TestUnknownObservationWire distinguishes missing evidence from zero values.
//
// Version:
//   - 2026-09-19: Added.
func TestUnknownObservationWire(t *testing.T) {
	raw, err := json.Marshal(Pair{PoolID: "unknown"})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"swapObservedAt", "swapObservedPosition", "confirmedAt", "liquidityEvaluatedAt"} {
		if string(fields[key]) != "null" {
			t.Fatal(key, string(fields[key]))
		}
	}
	var result Result
	if err := json.Unmarshal([]byte(`{"pairs":[],"excludedPairs":[]}`), &result); err != nil {
		t.Fatal(err)
	}
	if result.Pairs == nil || result.ExcludedPairs == nil {
		t.Fatal("empty lists lost")
	}
}
