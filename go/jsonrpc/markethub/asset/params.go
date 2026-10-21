// Package asset defines registered Mainnet asset metadata, independently of prices.
package asset

import (
	"fmt"
	"regexp"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
)

var assetPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type Params struct {
	AssetType market.AssetClass `json:"assetType"`
	Asset     string            `json:"asset"`
}

// Validate validates the required crypto classification and canonical catalog slug.
//
// Version:
//   - 2026-10-21: Added.
func (p Params) Validate() error {
	if p.AssetType == market.AssetClassUnknown {
		return invalid("asset_type", "empty")
	}
	if p.AssetType != market.AssetClassCrypto {
		return invalid("asset_type", "invalid")
	}
	if p.Asset == "" {
		return invalid("asset", "empty")
	}
	if len(p.Asset) > 64 {
		return invalid("asset", "too_long")
	}
	if !assetPattern.MatchString(p.Asset) {
		return invalid("asset", "invalid")
	}
	return nil
}

// UnmarshalJSON decodes required selectors atomically, rejecting unknown or duplicate fields.
//
// Version:
//   - 2026-10-21: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type wire Params
	var value wire
	if err := jsonobject.Decode(data, &value, "assetType", "asset"); err != nil {
		return fmt.Errorf("failed to decode asset parameters: %w", err)
	}
	if err := Params(value).Validate(); err != nil {
		return fmt.Errorf("failed to decode asset parameters: %w", err)
	}
	*p = Params(value)
	return nil
}

func invalid(field, state string) error {
	return fmt.Errorf("failed to validate asset: %w: %s=%s", apperror.InvalidParameter(), field, state)
}
