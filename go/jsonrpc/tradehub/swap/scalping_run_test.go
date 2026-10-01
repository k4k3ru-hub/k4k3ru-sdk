package swap

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestScalpingRunReferences preserves the entry/settlement boundary and strict decoding.
//
// Version:
//   - 2026-10-02: Added.
func TestScalpingRunReferences(t *testing.T) {
	good := ScalpingRunReference{ExecutionID: "scalprun_test"}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	good.PositionOrderID = "18446744073709551615"
	good.OrderRevision = strings.Repeat("a", 64)
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []ScalpingRunReference{{ExecutionID: ""}, {ExecutionID: "run", PositionOrderID: "1"}, {ExecutionID: "run", OrderRevision: good.OrderRevision}, {ExecutionID: "run", PositionOrderID: "01", OrderRevision: good.OrderRevision}, {ExecutionID: "run", PositionOrderID: "0", OrderRevision: good.OrderRevision}, {ExecutionID: "run", PositionOrderID: "1", OrderRevision: strings.Repeat("Z", 64)}} {
		if bad.Validate() == nil {
			t.Fatal("invalid reference accepted", bad)
		}
	}
	p := PrepareParams{ScalpingRun: &good}
	n := p.Normalize()
	n.ScalpingRun.ExecutionID = "changed"
	if p.ScalpingRun.ExecutionID != "scalprun_test" {
		t.Fatal("normalization mutated caller")
	}
	var parsed PrepareParams
	if err := json.Unmarshal([]byte(`{"scalpingRun":{"executionId":"run","unexpected":1}}`), &parsed); err == nil {
		t.Fatal("unknown field accepted")
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(encoded, &parsed); err != nil || parsed.ScalpingRun == nil || *parsed.ScalpingRun != good {
		t.Fatal("reference lost", err)
	}
	p.ScalpingExecutionID = "legacy"
	p.ScalpingRevision = strings.Repeat("a", 64)
	if p.Validate() == nil {
		t.Fatal("mixed legacy and Run accepted")
	}
}
