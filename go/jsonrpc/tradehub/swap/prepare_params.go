package swap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	k4k3ruSDKMarketHubArbitrage "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/arbitrage"
	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

type PrepareParams struct {
	Chain              k4k3ruOnchainCore.Chain                     `json:"chain"`
	Network            k4k3ruOnchainCore.Network                   `json:"network"`
	Venue              k4k3ruSDKMarketHubArbitrage.Venue           `json:"venue"`
	PoolID             string                                      `json:"poolId"`
	TokenInAssetID     string                                      `json:"tokenInAssetId"`
	TokenOutAssetID    string                                      `json:"tokenOutAssetId"`
	Amount             string                                      `json:"amount"`
	Kind               Kind                                        `json:"kind"`
	MaximumSlippageBPS *uint64                                     `json:"maximumSlippageBps"`
	Signer             string                                      `json:"signer"`
	Recipient          string                                      `json:"recipient"`
	ApprovalAmount     string                                      `json:"approvalAmount"`
	StateReference     *k4k3ruSDKMarketHubArbitrage.StateReference `json:"stateReference,omitempty"`
	ExecutionTTLMS     *uint64                                     `json:"executionTtlMs"`
	IdempotencyKey     string                                      `json:"idempotencyKey"`
}

// Normalize applies canonical formatting to swap preparation parameters.
//
// Returns:
//   - Normalized parameters.
//
// Version:
//   - 2026-09-10: Added.
func (p PrepareParams) Normalize() PrepareParams {
	quote := Params{
		Chain: p.Chain, Network: p.Network, Venue: p.Venue, PoolID: p.PoolID,
		TokenInAssetID: p.TokenInAssetID, TokenOutAssetID: p.TokenOutAssetID,
		Amount: p.Amount, Kind: p.Kind, MaximumSlippageBPS: p.MaximumSlippageBPS,
	}.Normalize()
	p.Chain, p.Network, p.Venue, p.PoolID = quote.Chain, quote.Network, quote.Venue, quote.PoolID
	p.TokenInAssetID, p.TokenOutAssetID, p.Amount, p.Kind = quote.TokenInAssetID, quote.TokenOutAssetID, quote.Amount, quote.Kind
	p.Signer = strings.TrimSpace(p.Signer)
	p.Recipient = strings.TrimSpace(p.Recipient)
	p.ApprovalAmount = strings.TrimSpace(p.ApprovalAmount)
	p.IdempotencyKey = strings.TrimSpace(p.IdempotencyKey)
	return p
}

// Validate validates swap preparation parameters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-10: Added.
func (p PrepareParams) Validate() error {
	p = p.Normalize()
	quote := Params{
		Chain: p.Chain, Network: p.Network, Venue: p.Venue, PoolID: p.PoolID,
		TokenInAssetID: p.TokenInAssetID, TokenOutAssetID: p.TokenOutAssetID,
		Amount: p.Amount, Kind: p.Kind, MaximumSlippageBPS: p.MaximumSlippageBPS,
	}
	if err := quote.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade hub swap preparation parameters: %w", err)
	}
	if p.Signer == "" {
		return invalidPrepareParameter("signer=empty")
	}
	if p.Recipient == "" {
		return invalidPrepareParameter("recipient=empty")
	}
	approvalAmount, ok := new(big.Int).SetString(p.ApprovalAmount, 10)
	if !ok {
		return invalidPrepareParameter("approval_amount=invalid")
	}
	if approvalAmount.Sign() <= 0 || approvalAmount.BitLen() > 256 {
		return invalidPrepareParameter("approval_amount=out_of_range")
	}
	if p.ExecutionTTLMS == nil {
		return invalidPrepareParameter("execution_ttl_ms=null")
	}
	if *p.ExecutionTTLMS == 0 {
		return invalidPrepareParameter("execution_ttl_ms=empty")
	}
	if p.IdempotencyKey == "" {
		return invalidPrepareParameter("idempotency_key=empty")
	}
	if len(p.IdempotencyKey) > 128 {
		return invalidPrepareParameter("idempotency_key=too_long")
	}
	return nil
}

// UnmarshalJSON decodes swap preparation parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded parameters.
//
// Version:
//   - 2026-09-10: Added.
func (p *PrepareParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalidPrepareParameter("destination=null")
	}
	type wire PrepareParams
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded wire
	if err := decoder.Decode(&decoded); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub swap preparation parameters: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing json value")
		}
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub swap preparation parameters: %w", err)
	}
	*p = PrepareParams(decoded)
	return nil
}

func invalidPrepareParameter(state string) error {
	return k4k3ruSDKAppError.Tracef("failed to validate trade hub swap preparation parameters: %w: %s", k4k3ruSDKAppError.InvalidParameter(), state)
}
