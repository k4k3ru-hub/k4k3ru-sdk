package market

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
	onchain "github.com/k4k3ru-hub/onchain/go/core"
)

type MarketRef struct {
	Venue       Venue           `json:"venue"`
	Network     onchain.Network `json:"network"`
	Chain       onchain.Chain   `json:"chain,omitempty"`
	PoolID      string          `json:"poolId,omitempty"`
	VenueSymbol string          `json:"venueSymbol,omitempty"`
}

// Normalize normalizes scopes without changing case-sensitive instrument identifiers.
//
// Version:
//   - 2026-09-25: Normalize scopes using onchain-owned Chain and Network types.
func (r MarketRef) Normalize() MarketRef {
	r.Venue = Venue(strings.ToLower(strings.TrimSpace(string(r.Venue))))
	r.Chain = onchain.Chain(strings.ToLower(strings.TrimSpace(string(r.Chain))))
	r.Network = onchain.Network(strings.ToLower(strings.TrimSpace(string(r.Network))))
	r.PoolID = strings.TrimSpace(r.PoolID)
	r.VenueSymbol = strings.TrimSpace(r.VenueSymbol)
	return r
}

// Validate validates a pool or venue instrument identity without resolving metadata.
//
// Version:
//   - 2026-09-25: Delegate chain and network validation to onchain.
func (r MarketRef) Validate() error {
	r = r.Normalize()
	if err := r.Venue.Validate(); err != nil {
		return fmt.Errorf("failed to validate market reference: %w: %w", apperror.InvalidParameter(), err)
	}
	if err := r.Network.Validate(); err != nil {
		return fmt.Errorf("failed to validate market reference: %w: %w", apperror.InvalidParameter(), err)
	}
	if (r.PoolID == "") == (r.VenueSymbol == "") {
		return fmt.Errorf("failed to validate market reference: %w: instrument=invalid", apperror.InvalidParameter())
	}
	if r.Chain != "" {
		if err := r.Chain.Validate(); err != nil {
			return fmt.Errorf("failed to validate market reference: %w: %w", apperror.InvalidParameter(), err)
		}
	}
	if r.PoolID != "" {
		if r.Chain == "" {
			return fmt.Errorf("failed to validate market reference: %w: chain=empty", apperror.InvalidParameter())
		}
		return referenceText("pool_id", r.PoolID, 256)
	}
	return referenceText("venue_symbol", r.VenueSymbol, 256)
}

// UnmarshalJSON decodes a strict, normalized market reference atomically.
//
// Version:
//   - 2026-09-25: Added.
func (r *MarketRef) UnmarshalJSON(data []byte) error {
	if r == nil {
		return fmt.Errorf("failed to decode market reference: %w: destination=null", apperror.InvalidParameter())
	}
	type wire MarketRef
	var decoded wire
	if err := jsonobject.Decode(data, &decoded, "venue", "network"); err != nil {
		return fmt.Errorf("failed to decode market reference: %w", err)
	}
	value := MarketRef(decoded).Normalize()
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode market reference: %w", err)
	}
	*r = value
	return nil
}

func referenceText(name, value string, maximum int) error {
	state := ""
	switch {
	case value == "":
		state = "empty"
	case len(value) > maximum:
		state = "too_long"
	case value != strings.TrimSpace(value) || strings.IndexFunc(value, unicode.IsControl) >= 0:
		state = "invalid"
	}
	if state != "" {
		return fmt.Errorf("failed to validate market identifier: %w: %s=%s", apperror.InvalidParameter(), name, state)
	}
	return nil
}
