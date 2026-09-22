package newpair

import (
	"encoding/json"
	"testing"
)

// TestTokenTaxesJSON preserves old JSON, nulls, zero and partial observations.
//
// Version:
//   - 2026-09-22: Added.
func TestTokenTaxesJSON(t *testing.T) {
	for _, raw := range []string{`{}`, `{"tokenTaxes":null}`, `{"tokenTaxes":{"token0":null,"token1":null}}`, `{"tokenTaxes":{"token0":{"buyRate":"0","sellRate":null,"canChange":false,"hasExemptions":null,"source":"contract_analysis","observedAt":1,"position":{"kind":"block","number":"1","id":"hash"}},"token1":null}}`} {
		var p Pair
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		var round Pair
		if err := json.Unmarshal(encoded, &round); err != nil {
			t.Fatal(err)
		}
		if p.TokenTaxes != nil && p.TokenTaxes.Token0 != nil {
			v := round.TokenTaxes.Token0
			if v.BuyRate == nil || *v.BuyRate != "0" || v.SellRate != nil || v.CanChange == nil || *v.CanChange || v.HasExemptions != nil {
				t.Fatalf("lost partial observation: %+v", v)
			}
		}
	}
}

// TestCloneTokenTaxes protects callers from changes to shared observation fields.
//
// Version:
//   - 2026-09-22: Added.
func TestCloneTokenTaxes(t *testing.T) {
	z, f := "0", false
	v := &TokenTaxes{Token0: &TokenTax{BuyRate: &z, SellRate: &z, CanChange: &f, HasExemptions: &f, Position: Position{Details: json.RawMessage(`{"a":1}`)}}}
	c := CloneTokenTaxes(v)
	*c.Token0.BuyRate = "0.2"
	*c.Token0.CanChange = true
	c.Token0.Position.Details[0] = ' '
	if *v.Token0.BuyRate != "0" || *v.Token0.CanChange || v.Token0.Position.Details[0] != '{' || c.Token1 != nil || CloneTokenTaxes(nil) != nil {
		t.Fatal("clone shares fields or changes null")
	}
}
