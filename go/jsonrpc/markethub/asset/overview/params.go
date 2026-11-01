// Package overview defines retained asset views independently of request-time market calculations.
package overview

import (
	"fmt"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/asset"
)

const MaximumResponseBytes = 4 << 20
const MaximumMarkets = 512
const MaximumAssets = 512

type Params struct {
	AssetType market.AssetClass `json:"assetType"`
	Asset     string            `json:"asset"`
}

// Validate validates the crypto classification and canonical catalogue slug.
//
// Version:
//   - 2026-11-01: Added.
func (p Params) Validate() error {
	return (asset.Params{AssetType: p.AssetType, Asset: p.Asset}).Validate()
}

// UnmarshalJSON decodes required selectors atomically and rejects ambiguous fields.
//
// Version:
//   - 2026-11-01: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode overview parameters: destination=null")
	}
	type wire Params
	var v wire
	if err := jsonobject.Decode(data, &v, "assetType", "asset"); err != nil {
		return fmt.Errorf("failed to decode overview parameters: %w", err)
	}
	if err := Params(v).Validate(); err != nil {
		return fmt.Errorf("failed to decode overview parameters: %w", err)
	}
	*p = Params(v)
	return nil
}
