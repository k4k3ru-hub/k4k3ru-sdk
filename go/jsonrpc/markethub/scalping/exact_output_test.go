package scalping_test

import (
	"encoding/json"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
)

// TestExactOutputContract separates fixed input/output subscriptions and preserves default input semantics.
//
// Version:
//   - 2026-10-02: Added.
func TestExactOutputContract(t *testing.T) {
	var p scalping.Params
	if err := json.Unmarshal([]byte(validRequest), &p); err != nil {
		t.Fatal(err)
	}
	p.Buy = &scalping.SideParams{Quantity: &market.Quantity{Amount: "1000000000", Decimals: 9}}
	inputKey, err := p.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	p.Buy.Kind = scalping.KindExactInput
	explicitKey, err := p.SubscriptionKey()
	if err != nil || inputKey != explicitKey {
		t.Fatal("default input identity changed", err)
	}
	p.Buy.Kind = scalping.KindExactOutput
	outputKey, err := p.SubscriptionKey()
	if err != nil || outputKey == inputKey {
		t.Fatal("output target reused an input subscription", err)
	}
	normal := p.Normalize()
	normal.Buy.Quantity.Amount = "2"
	if p.Buy.Quantity.Amount != "1000000000" || normal.Buy.Kind != scalping.KindExactOutput {
		t.Fatal("normalization lost mode or aliased target")
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded scalping.Params
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded.Buy.Kind != scalping.KindExactOutput {
		t.Fatal("wire mode lost", err)
	}
	for _, raw := range []string{`{"kind":"exact-output"}`, `{"kind":"unknown","quantity":{"amount":"1","decimals":0}}`, `{"kind":"exact-output","quantity":{"amount":"0","decimals":0}}`} {
		var side scalping.SideParams
		if json.Unmarshal([]byte(raw), &side) == nil {
			t.Fatal("invalid output target accepted", raw)
		}
	}
	p.MarketType = market.MarketTypePerpetual
	if p.Validate() == nil {
		t.Fatal("Spot receipt semantics accepted for perpetual")
	}
}
