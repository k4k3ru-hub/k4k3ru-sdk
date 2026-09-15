package launch

import (
	"encoding/json"
	"testing"
)

// TestLaunchParams validates wire types, canonical keys and independent age selectors.
//
// Version:
//   - 2026-09-15: Added.
func TestLaunchParams(t *testing.T) {
	for _, raw := range []string{`null`, `{}`, `{"maxAgeSeconds":0}`, `{"maxAgeSeconds":1.5}`, `{"maxAgeSeconds":-1}`, `{"maxAgeSeconds":604801}`, `{"maxAgeSeconds":1,"maxTokenAgeSeconds":604801}`, `{"maxAgeSeconds":1,"unknown":true}`, `{"maxAgeSeconds":1} {}`} {
		var p Params
		if json.Unmarshal([]byte(raw), &p) == nil {
			t.Fatal("accepted", raw)
		}
	}
	p := Params{Chain: " BASE ", Venue: "UNISWAP-V4", MaxAgeSeconds: 86400, MaxTokenAgeSeconds: 3600}
	a, e := p.SubscriptionKey()
	if e != nil {
		t.Fatal(e)
	}
	b, e := p.Normalize().SubscriptionKey()
	if e != nil || a != b {
		t.Fatal(a, b, e)
	}
	p.MaxTokenAgeSeconds = 7200
	b, e = p.SubscriptionKey()
	if e != nil || a == b {
		t.Fatal("token ages collide")
	}
	var list ListParams
	if e := json.Unmarshal([]byte(`{"filter":{"maxAgeSeconds":86400},"limit":200}`), &list); e != nil {
		t.Fatal(e)
	}
}

// TestGetIdentity distinguishes v4 IDs from per-pool contract addresses.
//
// Version:
//   - 2026-09-15: Added.
func TestGetIdentity(t *testing.T) {
	p := GetParams{Chain: "base", Network: "mainnet", Venue: "uniswap-v3", PoolID: "0x0000000000000000000000000000000000000001"}
	if e := p.Validate(); e != nil {
		t.Fatal(e)
	}
	p.Venue = "uniswap-v4"
	if p.Validate() == nil {
		t.Fatal("v4 address accepted as pool ID")
	}
}
