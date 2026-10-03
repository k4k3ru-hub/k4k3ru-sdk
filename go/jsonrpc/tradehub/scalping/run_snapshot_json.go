package scalping

import (
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// MarshalJSON encodes Spot orders or Perpetual positions, retaining empty Spot arrays.
//
// Version:
//   - 2026-10-03: Added.
func (r RunSnapshot) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("failed to encode scalping run snapshot: %w", err)
	}
	type wire RunSnapshot
	value := wire(r)
	if r.MarketType == market.MarketTypeSpot {
		return json.Marshal(struct {
			wire
			Orders []RunOrder `json:"orders"`
		}{wire: value, Orders: r.Orders})
	}
	return json.Marshal(value)
}

// UnmarshalJSON decodes and validates a complete product-specific snapshot atomically.
//
// Version:
//   - 2026-10-03: Added.
func (r *RunSnapshot) UnmarshalJSON(data []byte) error {
	if r == nil {
		return v.Invalid("decode scalping run snapshot", "destination", "null")
	}
	type wire RunSnapshot
	var decoded wire
	if err := decodeRunJSON(data, &decoded, "evaluationId", "marketType", "symbol", "evaluatedAt", "entry"); err != nil {
		return fmt.Errorf("failed to decode scalping run snapshot: %w", err)
	}
	value := RunSnapshot(decoded)
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping run snapshot: %w", err)
	}
	*r = value
	return nil
}
