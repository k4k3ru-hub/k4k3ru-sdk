package perpetual

import (
	"fmt"

	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// TradingCapacity retains the venue's direction-specific capacity observation.
// Values are bounds, not reservations or guarantees of a subsequent fill.
type TradingCapacity struct {
	Buy        TradingSideCapacity `json:"buy"`
	Sell       TradingSideCapacity `json:"sell"`
	Leverage   uint32              `json:"leverage"`
	MarginMode string              `json:"marginMode"`
	MarkPrice  string              `json:"markPrice"`
	Contract   Contract            `json:"contract"`
	// ObservedAt is TradeHub's read-completion time; activeAssetData has no venue timestamp.
	ObservedAt uint64 `json:"observedAt"`
}

type TradingSideCapacity struct {
	MaximumQuantity     string `json:"maximumQuantity"`     // Base contracts, before local unreflected reservations
	AvailableCollateral string `json:"availableCollateral"` // Quote collateral, can be negative
}

// Validate checks complete finite capacity without treating a balance as executable buying power.
//
// Version:
//   - 2026-10-04: Added.
func (c TradingCapacity) Validate() error {
	const op = "validate perpetual trading capacity"
	if c.ObservedAt == 0 || c.Leverage == 0 || c.Leverage > c.Contract.MaxLeverage || c.Contract.Coin != "SUI" || c.Contract.SizeDecimals < 0 || c.Contract.SizeDecimals > 6 || (c.MarginMode != "cross" && c.MarginMode != "isolated") || c.Contract.OnlyIsolated && c.MarginMode != "isolated" {
		return v.Invalid(op, "capacity", "invalid")
	}
	price, err := v.Number(op, "mark_price", c.MarkPrice, false, false)
	if err != nil {
		return fmt.Errorf("failed to validate perpetual capacity: %w", err)
	}
	if price.Sign() <= 0 {
		return v.Invalid(op, "mark_price", "out_of_range")
	}
	for _, side := range []TradingSideCapacity{c.Buy, c.Sell} {
		if _, err := v.Number(op, "maximum_quantity", side.MaximumQuantity, false, false); err != nil {
			return fmt.Errorf("failed to validate perpetual capacity: %w", err)
		}
		if _, err := v.Number(op, "available_collateral", side.AvailableCollateral, false, true); err != nil {
			return fmt.Errorf("failed to validate perpetual capacity: %w", err)
		}
	}
	return nil
}
