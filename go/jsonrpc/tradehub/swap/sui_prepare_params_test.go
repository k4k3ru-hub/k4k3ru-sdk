package swap

import (
	"encoding/json"
	"testing"

	execution "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
	sui "github.com/k4k3ru-hub/onchain/go/sui"
)

func suiPrepareParams() PrepareParams {
	bps, ttl := uint64(100), uint64(30000)
	return PrepareParams{Chain: "sui", Network: "testnet", Venue: "cetus", PoolID: "0x99", TokenInAssetID: "0x2::sui::SUI", TokenOutAssetID: "0x3::usdc::USDC", Amount: "1000", Kind: KindExactInput, Signer: "0x11", Recipient: "0x12", MaximumSlippageBPS: &bps, ExecutionTTLMS: &ttl, IdempotencyKey: "sui-test", Sui: &SuiPrepareParams{GasBudget: "50000000", GasPayment: []execution.SuiObjectRef{{ObjectID: "0x81", Version: "9007199254740993", Digest: (sui.ObjectDigest{1}).String()}}}}
}

// TestSuiPrepareParams verifies lossless JSON, validation and immutable normalization.
//
// Version:
//   - 2026-09-24: Added.
func TestSuiPrepareParams(t *testing.T) {
	p := suiPrepareParams()
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	n := p.Normalize()
	if p.Sui.GasPayment[0].ObjectID != "0x81" || n.Sui.GasPayment[0].ObjectID == "0x81" {
		t.Fatal("normalization mutated caller or failed to canonicalize")
	}
	n.Sui.GasPayment[0].Version = "1"
	if p.Sui.GasPayment[0].Version != "9007199254740993" {
		t.Fatal("shared mutable slice")
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded PrepareParams
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Sui.GasPayment[0].Version != "9007199254740993" {
		t.Fatal("lost version precision")
	}
	cases := map[string]func(*PrepareParams){
		"missing gas":   func(p *PrepareParams) { p.Sui.GasPayment = nil },
		"duplicate gas": func(p *PrepareParams) { p.Sui.GasPayment = append(p.Sui.GasPayment, p.Sui.GasPayment[0]) },
		"overlap": func(p *PrepareParams) {
			p.TokenInAssetID, p.TokenOutAssetID = p.TokenOutAssetID, p.TokenInAssetID
			p.Sui.InputCoins = p.Sui.GasPayment
		},
		"no token coins": func(p *PrepareParams) { p.TokenInAssetID, p.TokenOutAssetID = p.TokenOutAssetID, p.TokenInAssetID },
		"native extra coin": func(p *PrepareParams) {
			p.Sui.InputCoins = []execution.SuiObjectRef{{ObjectID: "0x82", Version: "1", Digest: (sui.ObjectDigest{2}).String()}}
		},
		"overflow amount": func(p *PrepareParams) { p.Amount = "18446744073709551616" },
		"zero gas":        func(p *PrepareParams) { p.Sui.GasBudget = "0" },
		"approval":        func(p *PrepareParams) { p.ApprovalAmount = "1" },
		"exact output":    func(p *PrepareParams) { p.Kind = KindExactOutput },
		"missing sui":     func(p *PrepareParams) { p.Sui = nil },
		"evm with sui":    func(p *PrepareParams) { p.Chain = "base"; p.Network = "sepolia"; p.ApprovalAmount = "1" },
		"ttl overflow":    func(p *PrepareParams) { v := ^uint64(0); p.ExecutionTTLMS = &v },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			p := suiPrepareParams()
			change(&p)
			if err := p.Validate(); err == nil {
				t.Fatal("accepted invalid request")
			}
		})
	}
	if err := json.Unmarshal([]byte(`{"sui":{"unknown":true}}`), &decoded); err == nil {
		t.Fatal("accepted unknown nested field")
	}
}
