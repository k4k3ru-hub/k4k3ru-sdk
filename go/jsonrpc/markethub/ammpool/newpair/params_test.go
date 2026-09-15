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
	yes := true
	p := Params{MaxPoolAgeSeconds: 86400, HasSwap: &yes, MinLiquidityUSD: "1000.00"}
	normalized := p.Normalize()
	yes = false
	if !*normalized.HasSwap {
		t.Fatal("normalized filter retained mutable pointer")
	}
	q := normalized
	q.MinLiquidityUSD = "1000"
	a, e := q.SubscriptionKey()
	if e != nil {
		t.Fatal(e)
	}
	b, e := normalized.SubscriptionKey()
	if e != nil || a != b {
		t.Fatal(a, b, e)
	}
	for _, raw := range []string{`{"maxPoolAgeSeconds":86400,"maxTokenAgeSeconds":10}`, `{"maxPoolAgeSeconds":0}`, `{"maxPoolAgeSeconds":86400,"minLiquidityUsd":"NaN"}`, `{"maxPoolAgeSeconds":86400,"minLiquidityUsd":"-1"}`} {
		var p Params
		if json.Unmarshal([]byte(raw), &p) == nil {
			t.Fatal("invalid filter accepted", raw)
		}
	}
}
