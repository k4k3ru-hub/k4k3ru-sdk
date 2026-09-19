package ammpool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"strings"
)

type GetParams struct {
	Chain   string `json:"chain"`
	Network string `json:"network"`
	Venue   string `json:"venue"`
	PoolID  string `json:"poolId"`
}

type GetResult struct {
	Pool PoolMetadata `json:"pool"`
}

// Normalize canonicalizes a pool lookup without changing its identity.
//
// Version:
//   - 2026-09-19: Added.
func (p GetParams) Normalize() GetParams {
	scope := (ListParams{Chain: p.Chain, Network: p.Network, Venue: p.Venue}).Normalize()
	p.Chain, p.Network, p.Venue = scope.Chain, scope.Network, scope.Venue
	p.PoolID = strings.ToLower(strings.TrimSpace(p.PoolID))
	if common.IsHexAddress(p.PoolID) {
		p.PoolID = strings.ToLower(common.HexToAddress(p.PoolID).Hex())
	}
	return p
}

// Validate validates a complete EVM pool lookup.
//
// Version:
//   - 2026-09-19: Added.
func (p GetParams) Validate() error {
	p = p.Normalize()
	if p.Chain == "" || p.Network == "" || p.Venue == "" {
		return fmt.Errorf("failed to validate trade hub amm pool get parameters: %w: scope=empty", app.InvalidParameter())
	}
	if err := (ListParams{Chain: p.Chain, Network: p.Network, Venue: p.Venue}).Validate(); err != nil {
		return fmt.Errorf("failed to validate trade hub amm pool get parameters: %w", err)
	}
	if !common.IsHexAddress(p.PoolID) || common.HexToAddress(p.PoolID) == (common.Address{}) {
		return fmt.Errorf("failed to validate trade hub amm pool get parameters: %w: pool_id=invalid", app.InvalidParameter())
	}
	return nil
}

// UnmarshalJSON decodes and validates a strict pool lookup object.
//
// Version:
//   - 2026-09-19: Added.
func (p *GetParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode trade hub amm pool get parameters: %w: destination=null", app.InvalidParameter())
	}
	if data = bytes.TrimSpace(data); len(data) == 0 || data[0] != '{' || !json.Valid(data) {
		return fmt.Errorf("failed to decode trade hub amm pool get parameters: %w: json=invalid", app.InvalidParameter())
	}
	type wire GetParams
	var value wire
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("failed to decode trade hub amm pool get parameters: %w: %w", app.InvalidParameter(), err)
	}
	normalized := GetParams(value).Normalize()
	if err := normalized.Validate(); err != nil {
		return fmt.Errorf("failed to decode trade hub amm pool get parameters: %w", err)
	}
	*p = normalized
	return nil
}
