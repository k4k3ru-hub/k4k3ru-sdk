package market

import (
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
	onchain "github.com/k4k3ru-hub/onchain/go/core"
)

type MarketTarget struct {
	Venue       Venue           `json:"venue"`
	Network     onchain.Network `json:"network"`
	Chain       onchain.Chain   `json:"chain,omitempty"`
	PoolID      string          `json:"poolId,omitempty"`
	VenueSymbol string          `json:"venueSymbol,omitempty"`
}

// Normalize normalizes a search scope while preserving instrument case.
//
// Version:
//   - 2026-09-26: Added.
func (t MarketTarget) Normalize() MarketTarget {
	return MarketTarget(MarketRef(t).Normalize())
}

// Validate validates a catalog search scope with optional chain and instrument filters.
// A concrete MarketRef must be resolved and validated by the service.
//
// Version:
//   - 2026-09-26: Added.
func (t MarketTarget) Validate() error {
	t = t.Normalize()
	if err := t.Venue.Validate(); err != nil {
		return fmt.Errorf("failed to validate market target: %w: %w", apperror.InvalidParameter(), err)
	}
	if err := t.Network.Validate(); err != nil {
		return fmt.Errorf("failed to validate market target: %w: %w", apperror.InvalidParameter(), err)
	}
	if t.Chain != "" {
		if err := t.Chain.Validate(); err != nil {
			return fmt.Errorf("failed to validate market target: %w: %w", apperror.InvalidParameter(), err)
		}
	}
	if t.PoolID != "" && t.VenueSymbol != "" {
		return fmt.Errorf("failed to validate market target: %w: instrument=invalid", apperror.InvalidParameter())
	}
	if t.PoolID != "" {
		return referenceText("pool_id", t.PoolID, 256)
	}
	if t.VenueSymbol != "" {
		return referenceText("venue_symbol", t.VenueSymbol, 256)
	}
	return nil
}

// UnmarshalJSON decodes a normalized search scope atomically.
//
// Version:
//   - 2026-09-26: Added.
func (t *MarketTarget) UnmarshalJSON(data []byte) error {
	if t == nil {
		return fmt.Errorf("failed to decode market target: %w: destination=null", apperror.InvalidParameter())
	}
	type wire MarketTarget
	var value wire
	if err := jsonobject.Decode(data, &value, "venue", "network"); err != nil {
		return fmt.Errorf("failed to decode market target: %w", err)
	}
	normalized := MarketTarget(value).Normalize()
	if err := normalized.Validate(); err != nil {
		return fmt.Errorf("failed to decode market target: %w", err)
	}
	*t = normalized
	return nil
}
