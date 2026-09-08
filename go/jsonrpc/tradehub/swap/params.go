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

type Params struct {
	Chain              k4k3ruOnchainCore.Chain           `json:"chain"`
	Network            k4k3ruOnchainCore.Network         `json:"network"`
	Venue              k4k3ruSDKMarketHubArbitrage.Venue `json:"venue"`
	PoolID             string                            `json:"poolId"`
	TokenInAssetID     string                            `json:"tokenInAssetId"`
	TokenOutAssetID    string                            `json:"tokenOutAssetId"`
	Amount             string                            `json:"amount"`
	Kind               Kind                              `json:"kind"`
	MaximumSlippageBPS *uint64                           `json:"maximumSlippageBps"`
}

// Normalize applies canonical formatting to swap quote parameters.
//
// Returns:
//   - Normalized parameters.
//
// Version:
//   - 2026-09-08: Added.
func (p Params) Normalize() Params {
	p.Chain = k4k3ruOnchainCore.Chain(strings.ToLower(strings.TrimSpace(string(p.Chain))))
	p.Network = k4k3ruOnchainCore.Network(strings.ToLower(strings.TrimSpace(string(p.Network))))
	p.Venue = k4k3ruSDKMarketHubArbitrage.Venue(strings.ToLower(strings.TrimSpace(string(p.Venue))))
	p.PoolID = strings.TrimSpace(p.PoolID)
	p.TokenInAssetID = strings.TrimSpace(p.TokenInAssetID)
	p.TokenOutAssetID = strings.TrimSpace(p.TokenOutAssetID)
	p.Amount = strings.TrimSpace(p.Amount)
	p.Kind = Kind(strings.ToLower(strings.TrimSpace(string(p.Kind))))
	return p
}

// Validate validates swap quote parameters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-08: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if err := p.Chain.Validate(); err != nil {
		return invalid("chain=invalid", err)
	}
	if err := p.Network.Validate(); err != nil {
		return invalid("network=invalid", err)
	}
	if p.Venue == "" {
		return invalid("venue=empty", nil)
	}
	for field, value := range map[string]string{"pool_id": p.PoolID, "token_in_asset_id": p.TokenInAssetID, "token_out_asset_id": p.TokenOutAssetID} {
		if value == "" {
			return invalid(field+"=empty", nil)
		}
	}
	if p.TokenInAssetID == p.TokenOutAssetID {
		return invalid("assets=invalid", nil)
	}
	amount, ok := new(big.Int).SetString(p.Amount, 10)
	if !ok {
		return invalid("amount=invalid", nil)
	}
	if amount.Sign() <= 0 || amount.BitLen() > 256 {
		return invalid("amount=out_of_range", nil)
	}
	if p.Kind != KindExactInput && p.Kind != KindExactOutput {
		return invalid("kind=invalid", nil)
	}
	if p.MaximumSlippageBPS == nil {
		return invalid("maximum_slippage_bps=null", nil)
	}
	if *p.MaximumSlippageBPS > 10_000 {
		return invalid("maximum_slippage_bps=out_of_range", nil)
	}
	return nil
}

// UnmarshalJSON decodes swap quote parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded parameters.
//
// Version:
//   - 2026-09-08: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination=null", nil)
	}
	type wire Params
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded wire
	if err := decoder.Decode(&decoded); err != nil {
		return invalid("json=invalid", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing json value")
		}
		return invalid("json=invalid", err)
	}
	*p = Params(decoded)
	return nil
}

func invalid(state string, err error) error {
	if err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade hub swap quote parameters: %w: %w: %s", k4k3ruSDKAppError.InvalidParameter(), err, state)
	}
	return k4k3ruSDKAppError.Tracef("failed to validate trade hub swap quote parameters: %w: %s", k4k3ruSDKAppError.InvalidParameter(), state)
}
