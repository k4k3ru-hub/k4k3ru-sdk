package swap

import (
	"encoding/json"
	"errors"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestPrepareParamsNormalizeAndValidate(t *testing.T) {
	t.Parallel()
	params := validPrepareParams()
	params.Chain, params.Network, params.Venue = " BASE ", " SEPOLIA ", " UNISWAP-V3 "
	params.Signer, params.Recipient, params.IdempotencyKey = " 0xowner ", " 0xrecipient ", " request-1 "
	if err := params.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	normalized := params.Normalize()
	if normalized.Chain != "base" || normalized.Network != "sepolia" || normalized.Venue != "uniswap-v3" || normalized.Signer != "0xowner" || normalized.IdempotencyKey != "request-1" {
		t.Fatalf("Normalize() = %+v", normalized)
	}
}

func TestPrepareParamsValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*PrepareParams)
	}{
		{name: "quote fields", mutate: func(p *PrepareParams) { p.Amount = "0" }},
		{name: "signer", mutate: func(p *PrepareParams) { p.Signer = "" }},
		{name: "recipient", mutate: func(p *PrepareParams) { p.Recipient = "" }},
		{name: "approval amount", mutate: func(p *PrepareParams) { p.ApprovalAmount = "0" }},
		{name: "execution ttl null", mutate: func(p *PrepareParams) { p.ExecutionTTLMS = nil }},
		{name: "execution ttl zero", mutate: func(p *PrepareParams) { zero := uint64(0); p.ExecutionTTLMS = &zero }},
		{name: "idempotency key", mutate: func(p *PrepareParams) { p.IdempotencyKey = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			params := validPrepareParams()
			test.mutate(&params)
			if err := params.Validate(); !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestPrepareParamsUnmarshalJSONRejectsUnknownField(t *testing.T) {
	t.Parallel()
	var params PrepareParams
	if err := json.Unmarshal([]byte(`{"unknown":true}`), &params); err == nil {
		t.Fatal("UnmarshalJSON() error = nil")
	}
}

func validPrepareParams() PrepareParams {
	slippage, ttl := uint64(50), uint64(60_000)
	return PrepareParams{
		Chain: "base", Network: "sepolia", Venue: "uniswap-v3", PoolID: "0xpool",
		TokenInAssetID: "0xtoken-in", TokenOutAssetID: "0xtoken-out", Amount: "1000000",
		Kind: KindExactInput, MaximumSlippageBPS: &slippage, Signer: "0xowner", Recipient: "0xrecipient",
		ApprovalAmount: "100000000", ExecutionTTLMS: &ttl, IdempotencyKey: "request-1",
	}
}
