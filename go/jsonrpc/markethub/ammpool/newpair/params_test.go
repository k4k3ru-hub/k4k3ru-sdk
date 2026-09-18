package newpair

import (
	"encoding/json"
	"testing"
)

func TestChainNeutralIdentifiers(t *testing.T) {
	for _, p := range []GetParams{{"solana", "mainnet", "raydium", "AbCd123"}, {"sui", "mainnet", "cetus", "0x123"}, {"base", "mainnet", "uniswap-v4", "0x456"}} {
		if err := p.Validate(); err != nil {
			t.Fatal(err)
		}
		if p.Normalize().PoolID != p.PoolID {
			t.Fatal("identifier case changed", p)
		}
	}
}
func TestFiltersAndIsolation(t *testing.T) {
	p := Params{Chain: " BASE ", Network: "MAINNET", Venue: "Uniswap-V4"}
	a, err := p.SubscriptionKey()
	b, other := (Params{Chain: "base", Network: "mainnet", Venue: "uniswap-v4"}).SubscriptionKey()
	if err != nil || other != nil || a != b {
		t.Fatal(a, b, err, other)
	}
	for _, raw := range []string{`{"maxPoolAgeSeconds":86400}`, `{"hasSwap":false}`, `{"minLiquidityUsd":"0"}`, `{"maxTokenAgeSeconds":10}`} {
		var p Params
		if json.Unmarshal([]byte(raw), &p) == nil {
			t.Fatal("removed filter accepted", raw)
		}
	}
	var all Params
	if err := json.Unmarshal([]byte(`{}`), &all); err != nil {
		t.Fatal(err)
	}
}
