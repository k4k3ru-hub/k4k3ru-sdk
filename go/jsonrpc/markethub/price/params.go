package price

import (
	"fmt"
	"regexp"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

var assetPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type Params struct {
	AssetType  market.AssetClass `json:"assetType"` // Required; only crypto is supported.
	Asset      string            `json:"asset"`
	MarketType market.MarketType `json:"marketType"`
}

// Validate validates the canonical asset and supported Composite market type.
//
// Version:
//   - 2026-10-19: Require the crypto asset type.
//   - 2026-10-18: Added.
func (p Params) Validate() error {
	if p.AssetType == market.AssetClassUnknown {
		return fmt.Errorf("failed to validate price parameters: %w: asset_type=empty", apperror.InvalidParameter())
	}
	if p.AssetType != market.AssetClassCrypto {
		return fmt.Errorf("failed to validate price parameters: %w: asset_type=invalid", apperror.InvalidParameter())
	}
	if !assetPattern.MatchString(p.Asset) {
		return fmt.Errorf("failed to validate price parameters: %w: asset=invalid", apperror.InvalidParameter())
	}
	if p.MarketType != "spot" && p.MarketType != "perpetual" {
		return fmt.Errorf("failed to validate price parameters: %w: market_type=invalid", apperror.InvalidParameter())
	}
	return nil
}
