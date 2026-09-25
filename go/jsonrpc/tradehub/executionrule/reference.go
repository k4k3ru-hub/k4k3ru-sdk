package executionrule

import (
	"fmt"
	"strings"

	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// Validate validates a spot or perpetual market type.
//
// Version:
//   - 2026-09-25: Require perpetual and reject the former perp value.
//   - 2026-09-23: Added.
func (m MarketType) Validate() error {
	switch MarketType(strings.ToLower(strings.TrimSpace(string(m)))) {
	case MarketTypeSpot, MarketTypePerpetual:
		return nil
	default:
		return v.Invalid("validate market type", "market_type", "invalid")
	}
}

// Normalize normalizes the asset scope while preserving identifier case.
//
// Version:
//   - 2026-09-23: Added.
func (r AssetRef) Normalize() AssetRef {
	r.Chain = strings.ToLower(strings.TrimSpace(r.Chain))
	r.Venue = strings.ToLower(strings.TrimSpace(r.Venue))
	r.Network = strings.ToLower(strings.TrimSpace(r.Network))
	r.AssetID = strings.TrimSpace(r.AssetID)
	return r
}

// Validate validates a chain asset or a venue asset, without resolving metadata.
//
// Version:
//   - 2026-09-23: Added.
func (r AssetRef) Validate() error {
	const op = "validate asset reference"
	r = r.Normalize()
	if (r.Chain == "") == (r.Venue == "") {
		return v.Invalid(op, "scope", "invalid")
	}
	if r.Chain != "" {
		if err := v.Text(op, "chain", r.Chain, 64); err != nil {
			return err
		}
	} else if err := v.Text(op, "venue", r.Venue, 64); err != nil {
		return err
	}
	if err := v.Text(op, "network", r.Network, 64); err != nil {
		return err
	}
	return v.Text(op, "asset_id", r.AssetID, 256)
}

// UnmarshalJSON decodes a validated asset reference with no unknown fields.
//
// Version:
//   - 2026-09-23: Added.
func (r *AssetRef) UnmarshalJSON(data []byte) error {
	if r == nil {
		return v.Invalid("decode asset reference", "destination", "null")
	}
	type wire AssetRef
	var decoded wire
	if err := v.Decode(data, &decoded, "network", "assetId"); err != nil {
		return fmt.Errorf("failed to decode asset reference: %w", err)
	}
	value := AssetRef(decoded).Normalize()
	if err := value.Validate(); err != nil {
		return err
	}
	*r = value
	return nil
}

// Normalize normalizes the market scope while preserving pool and symbol case.
//
// Version:
//   - 2026-09-23: Added.
func (r MarketRef) Normalize() MarketRef {
	r.Venue = strings.ToLower(strings.TrimSpace(r.Venue))
	r.Network = strings.ToLower(strings.TrimSpace(r.Network))
	r.Chain = strings.ToLower(strings.TrimSpace(r.Chain))
	r.PoolID = strings.TrimSpace(r.PoolID)
	r.VenueSymbol = strings.TrimSpace(r.VenueSymbol)
	return r
}

// Validate validates a pool or native instrument reference without resolving it.
//
// Version:
//   - 2026-09-23: Added.
func (r MarketRef) Validate() error {
	const op = "validate market reference"
	r = r.Normalize()
	if err := v.Text(op, "venue", r.Venue, 64); err != nil {
		return err
	}
	if err := v.Text(op, "network", r.Network, 64); err != nil {
		return err
	}
	if (r.PoolID == "") == (r.VenueSymbol == "") {
		return v.Invalid(op, "instrument", "invalid")
	}
	if r.Chain != "" {
		if err := v.Text(op, "chain", r.Chain, 64); err != nil {
			return err
		}
	}
	if r.PoolID != "" {
		if r.Chain == "" {
			return v.Invalid(op, "chain", "empty")
		}
		return v.Text(op, "pool_id", r.PoolID, 256)
	}
	return v.Text(op, "venue_symbol", r.VenueSymbol, 256)
}

// UnmarshalJSON decodes a validated market reference with no unknown fields.
//
// Version:
//   - 2026-09-23: Added.
func (r *MarketRef) UnmarshalJSON(data []byte) error {
	if r == nil {
		return v.Invalid("decode market reference", "destination", "null")
	}
	type wire MarketRef
	var decoded wire
	if err := v.Decode(data, &decoded, "venue", "network"); err != nil {
		return fmt.Errorf("failed to decode market reference: %w", err)
	}
	value := MarketRef(decoded).Normalize()
	if err := value.Validate(); err != nil {
		return err
	}
	*r = value
	return nil
}

// NormalizeMarkets copies and normalizes a market list without changing its order.
//
// Version:
//   - 2026-09-23: Added.
func NormalizeMarkets(markets []MarketRef) []MarketRef {
	if markets == nil {
		return nil
	}
	result := make([]MarketRef, len(markets))
	for i, market := range markets {
		result[i] = market.Normalize()
	}
	return result
}

// ValidateMarkets validates a nonempty list of distinct market references.
//
// Version:
//   - 2026-09-23: Added.
func ValidateMarkets(markets []MarketRef) error {
	if len(markets) == 0 {
		return v.Invalid("validate market list", "markets", "empty")
	}
	seen := make(map[MarketRef]struct{}, len(markets))
	for i, market := range markets {
		market = market.Normalize()
		if err := market.Validate(); err != nil {
			return fmt.Errorf("failed to validate market list: %w: market_index=%d", err, i)
		}
		if _, found := seen[market]; found {
			return v.Invalid("validate market list", "duplicate_market", "invalid")
		}
		seen[market] = struct{}{}
	}
	return nil
}
