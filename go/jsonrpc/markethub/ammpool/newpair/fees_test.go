package newpair

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestFeesCompatibilityAndClone verifies Pool swap fee behavior.
//
// Version:
//   - 2026-09-22: Added.
func TestFeesCompatibilityAndClone(t *testing.T) {
	var old Pair
	if err := json.Unmarshal([]byte(`{"poolId":"old"}`), &old); err != nil || old.Fees != nil {
		t.Fatalf("old JSON: %v", err)
	}
	sender := "0x0000000000000000000000000000000000000000"
	p := Pair{Fees: &Fees{Model: "variable", Token0ToToken1: FeeRate{Rate: "0"}, Token1ToToken0: FeeRate{Rate: "0.003"}, ReferenceSender: &sender, Position: Position{Details: json.RawMessage(`{"x":1}`)}}}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var round Pair
	if err := json.Unmarshal(raw, &round); err != nil {
		t.Fatal(err)
	}
	if round.Fees == nil || round.Fees.Token0ToToken1.Rate != "0" {
		t.Fatal("lost confirmed zero")
	}
	feeRaw, err := json.Marshal(round.Fees)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(feeRaw), `"status"`) {
		t.Fatal("exposed fee status")
	}
	clone := CloneFees(p.Fees)
	*clone.ReferenceSender = "changed"
	clone.Position.Details[5] = '2'
	if *p.Fees.ReferenceSender != sender || string(p.Fees.Position.Details) != `{"x":1}` {
		t.Fatal("clone aliases original")
	}
}
