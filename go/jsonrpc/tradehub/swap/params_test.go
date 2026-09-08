package swap

import (
	"encoding/json"
	"errors"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	k4k3ruSDKMarketHubArbitrage "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/arbitrage"
	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

func TestParamsNormalizeAndValidate(t *testing.T) {
	t.Parallel()
	slippage := uint64(100)
	params := Params{
		Chain: " BASE ", Network: " SEPOLIA ", Venue: " UNISWAP-V3 ", PoolID: " 0x1 ",
		TokenInAssetID: " 0x2 ", TokenOutAssetID: " 0x3 ", Amount: "10000", Kind: " EXACT-INPUT ", MaximumSlippageBPS: &slippage,
	}
	if err := params.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	normalized := params.Normalize()
	if normalized.Chain != k4k3ruOnchainCore.ChainBase || normalized.Network != k4k3ruOnchainCore.NetworkSepolia || normalized.Venue != k4k3ruSDKMarketHubArbitrage.VenueUniswapV3 || normalized.Kind != KindExactInput {
		t.Fatalf("Normalize() = %+v", normalized)
	}
}

func TestParamsValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	slippage := uint64(100)
	valid := Params{Chain: "base", Network: "sepolia", Venue: "uniswap-v3", PoolID: "0x1", TokenInAssetID: "0x2", TokenOutAssetID: "0x3", Amount: "1", Kind: KindExactInput, MaximumSlippageBPS: &slippage}
	tests := []struct {
		name   string
		mutate func(*Params)
	}{
		{name: "amount", mutate: func(p *Params) { p.Amount = "0" }},
		{name: "kind", mutate: func(p *Params) { p.Kind = "market" }},
		{name: "slippage", mutate: func(p *Params) { p.MaximumSlippageBPS = nil }},
		{name: "assets", mutate: func(p *Params) { p.TokenOutAssetID = p.TokenInAssetID }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := valid
			test.mutate(&params)
			if err := params.Validate(); !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestParamsUnmarshalJSONRejectsUnknownField(t *testing.T) {
	t.Parallel()
	var params Params
	if err := json.Unmarshal([]byte(`{"unknown":true}`), &params); !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
}
