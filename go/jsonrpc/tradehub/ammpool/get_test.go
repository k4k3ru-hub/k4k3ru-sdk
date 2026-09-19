package ammpool

import (
	"encoding/json"
	"testing"
)

// TestGetParamsContract rejects incomplete, malformed and unexpected lookup inputs.
//
// Version:
//   - 2026-09-19: Added.
func TestGetParamsContract(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `{}`, `{"chain":"base","network":"sepolia","venue":"uniswap-v3","poolId":"bad"}`, `{"chain":"base","network":"sepolia","venue":"uniswap-v3","poolId":"0x0000000000000000000000000000000000000000"}`, `{"chain":"base","network":"sepolia","venue":"uniswap-v3","poolId":"0x94bfc0574ff48e92ce43d495376c477b1d0eeec0","rpcUrl":"http://example.com"}`} {
		var p GetParams
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	var p GetParams
	if err := json.Unmarshal([]byte(`{"chain":" BASE ","network":" Sepolia ","venue":"UNISWAP-V3","poolId":"0x94bfc0574ff48e92ce43d495376c477b1d0eeec0"}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.Chain != "base" || p.Network != "sepolia" || p.Venue != "uniswap-v3" {
		t.Fatalf("not normalized: %+v", p)
	}
	unprefixed := p
	unprefixed.PoolID = p.PoolID[2:]
	if unprefixed.Normalize() != p {
		t.Fatal("equivalent pool address was not canonicalized")
	}

}
