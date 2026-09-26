package scalping

import (
	"fmt"
	"math/big"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observations "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"

	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// Normalize returns independent canonical parameters without adding defaults.
//
// Version:
//   - 2026-09-26: Normalize shared observation targets and explicit scaled quantities.
//   - 2026-09-25: Use SDK finance market types and canonical perpetual values.
//   - 2026-09-23: Added.
func (p Params) Normalize() Params {
	p.MarketType = p.MarketType.Normalize()
	p.BaseAsset = p.BaseAsset.Normalize()
	p.QuoteAsset = p.QuoteAsset.Normalize()
	observation := p.ObservationParams().Normalize()
	p.Symbol, p.Markets, p.Buy, p.Sell = observation.Symbol, observation.Markets, observation.Buy, observation.Sell
	p.Conditions = p.Conditions.Normalize()
	p.ExecutionRule = p.ExecutionRule.Normalize()
	return p
}

// Validate validates the request structure and exact numeric representations.
// Asset equivalence, market metadata, and executable inventory need server checks.
//
// Version:
//   - 2026-09-26: Validate single-symbol observations and shared market targets.
//   - 2026-09-25: Use SDK finance market types and canonical perpetual values.
//   - 2026-09-24: Enforce the maximum observation window through Conditions validation.
//   - 2026-09-23: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if err := validateMarketType(p.MarketType); err != nil {
		return fmt.Errorf("failed to validate scalping parameters: %w", err)
	}
	if err := p.BaseAsset.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping parameters: %w: asset=%q", err, "base")
	}
	if err := p.QuoteAsset.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping parameters: %w: asset=%q", err, "quote")
	}
	if p.BaseAsset == p.QuoteAsset {
		return v.Invalid("validate scalping parameters", "asset_pair", "invalid")
	}
	if err := p.ObservationParams().Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping parameters: %w", err)
	}
	if err := p.Conditions.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping parameters: %w", err)
	}
	if err := p.ExecutionRule.Validate(rule.MarketType(p.MarketType)); err != nil {
		return fmt.Errorf("failed to validate scalping parameters: %w", err)
	}
	return nil
}

// UnmarshalJSON decodes and validates parameters, rejecting unknown fields.
//
// Version:
//   - 2026-09-26: Require an explicit observation symbol.
//   - 2026-09-24: Resolve omitted observation windows while rejecting explicit invalid values.
//   - 2026-09-23: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return v.Invalid("decode scalping parameters", "destination", "null")
	}
	type wire Params
	var decoded wire
	if err := v.Decode(data, &decoded, "marketType", "symbol", "baseAsset", "quoteAsset", "markets", "conditions", "executionRule"); err != nil {
		return fmt.Errorf("failed to decode scalping parameters: %w", err)
	}
	value := Params(decoded).Normalize()
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping parameters: %w", err)
	}
	*p = value
	return nil
}

// UnmarshalJSON defaults an omitted window and validates explicit condition values.
// Null and zero windows are rejected. Decode failure leaves the receiver unchanged.
//
// Version:
//   - 2026-09-26: Accept optional snapshot age and scaled volume conditions.
//   - 2026-09-24: Added.
func (c *Conditions) UnmarshalJSON(data []byte) error {
	const op = "decode scalping conditions"
	if c == nil {
		return v.Invalid(op, "destination", "null")
	}
	// Shadow only the wire field to distinguish omission from explicit null,
	// preserving the existing uint64 field used by Go callers.
	type fields Conditions
	window := DefaultWindowMS
	decoded := struct {
		fields
		WindowMS *uint64 `json:"windowMs"`
	}{WindowMS: &window}
	if err := v.Decode(data, &decoded, "maximumDataAgeMs"); err != nil {
		return fmt.Errorf("failed to decode scalping conditions: %w", err)
	}
	if decoded.WindowMS == nil {
		return v.Invalid(op, "window_ms", "null")
	}
	value := Conditions(decoded.fields).Normalize()
	value.WindowMS = *decoded.WindowMS
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping conditions: %w", err)
	}
	*c = value
	return nil
}

// Normalize copies all optional condition bounds without supplying thresholds.
//
// Version:
//   - 2026-09-26: Preserve explicit quantity scales and optional snapshot age.
//   - 2026-09-23: Added.
func (c Conditions) Normalize() Conditions {
	c.PriceChangeBPS = normalizeDecimalRange(c.PriceChangeBPS)
	c.BuyVolumeRatioBPS = normalizeDecimalRange(c.BuyVolumeRatioBPS)
	c.MaximumSnapshotAgeMS = v.Pointer(c.MaximumSnapshotAgeMS)
	if c.QuoteVolume != nil {
		c.QuoteVolume = &QuantityRange{Minimum: v.Pointer(c.QuoteVolume.Minimum), Maximum: v.Pointer(c.QuoteVolume.Maximum)}
	}
	if c.TradeCount != nil {
		c.TradeCount = &CountRange{Minimum: v.Pointer(c.TradeCount.Minimum), Maximum: v.Pointer(c.TradeCount.Maximum)}
	}
	return c
}

func normalizeDecimalRange(r *DecimalRange) *DecimalRange {
	if r == nil {
		return nil
	}
	return &DecimalRange{Minimum: v.StringPointer(r.Minimum), Maximum: v.StringPointer(r.Maximum)}
}

// Validate validates the observation window and explicit AND condition bounds.
//
// Version:
//   - 2026-09-26: Compare quantity bounds across scales and default to no snapshot age limit.
//   - 2026-09-24: Limit explicit windows to 1 through MaximumWindowMS milliseconds.
//   - 2026-09-23: Added.
func (c Conditions) Validate() error {
	const op = "validate scalping conditions"
	c = c.Normalize()
	if c.WindowMS == 0 {
		return v.Invalid(op, "window_ms", "empty")
	}
	if c.WindowMS > MaximumWindowMS {
		return v.Invalid(op, "window_ms", "out_of_range")
	}
	if c.MaximumDataAgeMS == 0 {
		return v.Invalid(op, "maximum_data_age_ms", "empty")
	}
	if c.MaximumSnapshotAgeMS != nil && *c.MaximumSnapshotAgeMS == 0 {
		return v.Invalid(op, "maximum_snapshot_age_ms", "empty")
	}
	if c.PriceChangeBPS == nil && c.QuoteVolume == nil && c.TradeCount == nil && c.BuyVolumeRatioBPS == nil {
		return v.Invalid(op, "conditions", "empty")
	}
	if r := c.PriceChangeBPS; r != nil {
		if err := validateRange(op, "price_change_bps", r.Minimum, r.Maximum, false, true, nil); err != nil {
			return err
		}
	}
	if r := c.QuoteVolume; r != nil {
		if err := r.Validate(); err != nil {
			return err
		}
	}
	if r := c.BuyVolumeRatioBPS; r != nil {
		if err := validateRange(op, "buy_volume_ratio_bps", r.Minimum, r.Maximum, false, false, big.NewRat(10000, 1)); err != nil {
			return err
		}
	}
	if r := c.TradeCount; r != nil {
		if r.Minimum == nil && r.Maximum == nil {
			return v.Invalid(op, "trade_count", "empty")
		}
		if r.Minimum != nil && r.Maximum != nil && *r.Minimum > *r.Maximum {
			return v.Invalid(op, "trade_count", "out_of_range")
		}
	}
	return nil
}

// ObservationParams returns market-data inputs without converting an execution's input amount.
//
// Version:
//   - 2026-09-26: Forward Buy and Sell inputs without inferring execution amounts.
func (p Params) ObservationParams() observations.Params {
	return observations.Params{MarketType: p.MarketType, Symbol: p.Symbol, Markets: p.Markets, WindowMS: p.Conditions.WindowMS, Buy: p.Buy, Sell: p.Sell}
}

// Validate compares nonnegative volume bounds exactly across their decimal scales.
//
// Version:
//   - 2026-09-26: Added.
func (r QuantityRange) Validate() error {
	const op = "validate scalping quantity range"
	if r.Minimum == nil && r.Maximum == nil {
		return v.Invalid(op, "quantity_range", "empty")
	}
	for _, bound := range []*market.Quantity{r.Minimum, r.Maximum} {
		if bound != nil {
			if err := bound.Validate(); err != nil {
				return fmt.Errorf("failed to validate scalping quantity range: %w", err)
			}
		}
	}
	if r.Minimum != nil && r.Maximum != nil && scaledQuantity(*r.Minimum).Cmp(scaledQuantity(*r.Maximum)) > 0 {
		return v.Invalid(op, "quantity_range", "out_of_range")
	}
	return nil
}

func scaledQuantity(q market.Quantity) *big.Rat {
	n, _ := new(big.Int).SetString(q.Amount, 10)
	return new(big.Rat).SetFrac(n, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(q.Decimals)), nil))
}

func validateRange(op, field string, minimum, maximum *string, integer, signed bool, upper *big.Rat) error {
	if minimum == nil && maximum == nil {
		return v.Invalid(op, field, "empty")
	}
	var low, high *big.Rat
	for i, bound := range []*string{minimum, maximum} {
		if bound == nil {
			continue
		}
		number, err := v.Number(op, field, *bound, integer, signed)
		if err != nil {
			return err
		}
		if upper != nil && number.Cmp(upper) > 0 {
			return v.Invalid(op, field, "out_of_range")
		}
		if i == 0 {
			low = number
		} else {
			high = number
		}
	}
	if low != nil && high != nil && low.Cmp(high) > 0 {
		return v.Invalid(op, field, "out_of_range")
	}
	return nil
}
