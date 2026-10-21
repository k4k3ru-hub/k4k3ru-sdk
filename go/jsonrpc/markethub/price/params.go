package price

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
)

const MaximumMarkets = 64

var assetPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type Params struct {
	AssetType  market.AssetClass  `json:"assetType"` // Required; only crypto is supported.
	Asset      string             `json:"asset"`
	MarketType market.MarketType  `json:"marketType"`
	Markets    []market.MarketRef `json:"markets,omitempty"` // Optional exact Mainnet DEX references; does not filter Composite inputs.
}

// Normalize copies and normalizes market references without changing asset selectors.
//
// Version:
//   - 2026-10-21: Added.
func (p Params) Normalize() Params {
	if p.Markets != nil {
		p.Markets = append([]market.MarketRef{}, p.Markets...)
		for i := range p.Markets {
			p.Markets[i] = p.Markets[i].Normalize()
		}
	}
	return p
}

// Validate validates asset selectors and optional bounded Mainnet DEX references.
//
// Version:
//   - 2026-10-21: Validate individual market requests independently of Composite selection.
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
	if p.Markets != nil && len(p.Markets) == 0 {
		return fmt.Errorf("failed to validate price parameters: %w: markets=empty", apperror.InvalidParameter())
	}
	if len(p.Markets) > MaximumMarkets {
		return fmt.Errorf("failed to validate price parameters: %w: markets=too_long max_length=%d", apperror.InvalidParameter(), MaximumMarkets)
	}
	seen := make(map[market.MarketRef]bool, len(p.Markets))
	for i, ref := range p.Markets {
		ref = ref.Normalize()
		if err := validateMarketRef(ref); err != nil {
			return fmt.Errorf("failed to validate price parameters: %w: market_index=%d", err, i)
		}
		if seen[ref] {
			return fmt.Errorf("failed to validate price parameters: %w: duplicate_market=invalid", apperror.InvalidParameter())
		}
		seen[ref] = true
	}
	return nil
}

func validateMarketRef(ref market.MarketRef) error {
	if err := ref.Validate(); err != nil {
		return fmt.Errorf("failed to validate price market: %w", err)
	}
	filter := market.AggregationSourceFilter{VenueCategories: []market.VenueCategory{market.VenueCategoryDEX}}
	if ref.Network != "mainnet" || !filter.Matches(market.MarketSourceKey{Venue: ref.Venue, Chain: market.Chain(ref.Chain)}) {
		return fmt.Errorf("failed to validate price market: %w: scope=invalid", apperror.InvalidParameter())
	}
	return nil
}

// UnmarshalJSON decodes strict price selectors and optional nonempty market references.
//
// Version:
//   - 2026-10-21: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode price parameters: %w: destination=null", apperror.InvalidParameter())
	}
	type wire Params
	var value wire
	if err := jsonobject.Decode(data, &value, "assetType", "asset", "marketType"); err != nil {
		return fmt.Errorf("failed to decode price parameters: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to decode price parameters: %w", err)
	}
	if raw, exists := fields["markets"]; exists && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("failed to decode price parameters: %w: markets=null", apperror.InvalidParameter())
	}
	normalized := Params(value).Normalize()
	if err := normalized.Validate(); err != nil {
		return fmt.Errorf("failed to decode price parameters: %w", err)
	}
	*p = normalized
	return nil
}
