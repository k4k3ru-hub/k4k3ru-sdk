package newpair

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestTokenControlsNullableClone verifies false, skipped checks and legacy JSON.
//
// Version:
//   - 2026-09-23: Added.
func TestTokenControlsNullableClone(t *testing.T) {
	v := NewTokenControls("unknown", "unsupported_model")
	value := false
	v.Minting.CanMint = ControlBoolFinding{Status: "observed", Value: &value}
	at := int64(1790000000000000)
	v.ObservedAt = &at
	v.Position = &Position{Kind: "block", Number: "100", ID: "hash", Details: json.RawMessage(`{"a":1}`)}
	c := CloneTokenControls(v)
	*c.Minting.CanMint.Value = true
	*c.ObservedAt++
	c.Position.Details[5] = '2'
	if *v.Minting.CanMint.Value || *v.ObservedAt != at || string(v.Position.Details) != `{"a":1}` {
		t.Fatal("clone shared pointers")
	}
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"value":false`) || !strings.Contains(string(raw), `"value":null`) {
		t.Fatal("nullable contract lost")
	}
	var token Token
	if err := json.Unmarshal([]byte(`{"id":"legacy"}`), &token); err != nil || token.Controls != nil {
		t.Fatal("legacy response rejected")
	}
	trusted := NewTokenControls("trusted", "sdk_definition")
	if trusted.ObservedAt != nil || trusted.Position != nil || trusted.Minting.CanMint.Value != nil {
		t.Fatal("trust fabricated observations")
	}
}
