package newpair

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestLPPrincipalJSON verifies old responses, explicit nulls and exact decimal strings.
//
// Version:
//   - 2026-09-22: Added.
func TestLPPrincipalJSON(t *testing.T) {
	for _, raw := range []string{`{}`, `{"lpPrincipal":null}`, `{"lpPrincipal":{"token0":{"amount":"0","amountPercentage":null},"token1":{"amount":"0","amountPercentage":null},"evaluatedAt":1790000000000000,"position":{"kind":"block","number":"1","id":"hash"}}}`, `{"lpPrincipal":{"token0":{"amount":"1","amountPercentage":"0.001"},"token1":{"amount":"99999","amountPercentage":"99.999"},"evaluatedAt":1790000000000000,"position":{"kind":"block","number":"1","id":"hash"}}}`} {
		var p, restored Pair
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(encoded, &restored); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(p, restored) {
			t.Fatal("principal changed during JSON round trip")
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			t.Fatal(err)
		}
		if _, ok := fields["lpPrincipal"]; !ok {
			t.Fatal("missing explicit principal field")
		}
	}
}

// TestCloneLPPrincipal verifies mutable observation fields are copied independently.
//
// Version:
//   - 2026-09-22: Added.
func TestCloneLPPrincipal(t *testing.T) {
	a, b := "25", "75"
	p := &LPPrincipal{Token0: LPPrincipalToken{Amount: "1", AmountPercentage: &a}, Token1: LPPrincipalToken{Amount: "3", AmountPercentage: &b}, Position: Position{Details: json.RawMessage(`{"height":"1"}`)}}
	c := CloneLPPrincipal(p)
	*c.Token0.AmountPercentage = "0"
	*c.Token1.AmountPercentage = "100"
	c.Position.Details[2] = 'x'
	if a != "25" || b != "75" || string(p.Position.Details) != `{"height":"1"}` || CloneLPPrincipal(nil) != nil {
		t.Fatal("clone shared mutable fields")
	}
}
