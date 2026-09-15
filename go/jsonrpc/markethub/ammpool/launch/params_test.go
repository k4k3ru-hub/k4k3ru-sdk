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
	for _, raw := range []string{`{"maxAgeSeconds":86400}`, `{"maxPoolAgeSeconds":86400,"maxAgeSeconds":86400}`, `null`, `{}`, `{"maxPoolAgeSeconds":0}`, `{"maxPoolAgeSeconds":1.5}`, `{"maxPoolAgeSeconds":-1}`, `{"maxPoolAgeSeconds":604801}`, `{"maxPoolAgeSeconds":1,"maxTokenAgeSeconds":604801}`, `{"maxPoolAgeSeconds":1,"unknown":true}`, `{"maxPoolAgeSeconds":1} {}`} {
		var p Params
		if json.Unmarshal([]byte(raw), &p) == nil {
			t.Fatal("accepted", raw)
		}
	}
	p := Params{Chain: " BASE ", Venue: "UNISWAP-V4", MaxPoolAgeSeconds: 86400, MaxTokenAgeSeconds: 3600}
	a, e := p.SubscriptionKey()
	if e != nil {
		t.Fatal(e)
	}
	b, e := p.Normalize().SubscriptionKey()
	if e != nil || a != b {
		t.Fatal(a, b, e)
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if string(wire["maxPoolAgeSeconds"]) != "86400" || wire["maxAgeSeconds"] != nil {
		t.Fatalf("wrong wire selector: %s", encoded)
	}
	var decoded Params
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded != p {
		t.Fatalf("round trip: %+v %v", decoded, err)
	}
	p.MaxPoolAgeSeconds = 43200
	b, e = p.SubscriptionKey()
	if e != nil || a == b {
		t.Fatal("pool ages collide")
	}
	p.MaxPoolAgeSeconds = 86400
	p.MaxTokenAgeSeconds = 7200
	b, e = p.SubscriptionKey()
	if e != nil || a == b {
		t.Fatal("token ages collide")
	}
	var list ListParams
	if e := json.Unmarshal([]byte(`{"filter":{"maxPoolAgeSeconds":86400},"limit":200}`), &list); e != nil {
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
