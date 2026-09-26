package scalping_test

import (
	"encoding/json"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
)

// TestSubscriptionIdentity verifies condition isolation and order-independent keys without input mutation.
//
// Version:
//   - 2026-09-26: Added.
func TestSubscriptionIdentity(t *testing.T) {
	p := dto.Params{MarketType: "spot", Symbol: "SUI/USDC", WindowMS: 60000, Markets: []market.MarketTarget{{Venue: "cetus", Network: "testnet"}, {Venue: "hyperliquid", Network: "mainnet"}}}
	key, err := p.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	reordered := p.Normalize()
	reordered.Markets[0], reordered.Markets[1] = reordered.Markets[1], reordered.Markets[0]
	got, err := reordered.SubscriptionKey()
	if err != nil || got != key || reordered.Markets[0].Venue != "hyperliquid" {
		t.Fatalf("unstable key or mutated targets: %s %v", got, err)
	}
	for _, modify := range []func(*dto.Params){
		func(p *dto.Params) { p.WindowMS = 5000 }, func(p *dto.Params) { p.Symbol = "BTC/USDC" }, func(p *dto.Params) { p.MarketType = "perpetual" }, func(p *dto.Params) { p.Markets[0].Network = "mainnet" }, func(p *dto.Params) { p.Buy = &dto.SideParams{Quantity: &market.Quantity{Amount: "1", Decimals: 9}} },
	} {
		other := p.Normalize()
		modify(&other)
		k, e := other.SubscriptionKey()
		if e != nil || k == key {
			t.Fatalf("conditions conflated: %s %v", k, e)
		}
	}
	p.Buy = &dto.SideParams{Quantity: &market.Quantity{Amount: "001", Decimals: 9}}
	a, err := p.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	p.Buy.Quantity.Amount = "1"
	b, err := p.SubscriptionKey()
	if err != nil || a != b {
		t.Fatal("leading zeros changed identity", err)
	}
	for _, raw := range []string{`null`, `{}`, `{"subscriptionKey":null}`, `{"subscriptionKey":"bad"}`, `{"subscriptionKey":"` + key + `","executionId":"x"}`, `{"subscriptionKey":"` + key + `","subscriptionKey":"` + key + `"}`} {
		var value dto.UnsubscribeParams
		if json.Unmarshal([]byte(raw), &value) == nil {
			t.Fatalf("accepted invalid unsubscribe: %s", raw)
		}
	}
	var value dto.UnsubscribeParams
	if err := json.Unmarshal([]byte(`{"subscriptionKey":"`+key+`"}`), &value); err != nil {
		t.Fatal(err)
	}
}
