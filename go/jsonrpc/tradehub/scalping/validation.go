package scalping

import (
	"fmt"
	"math/big"
	"strings"

	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// Normalize returns independent canonical parameters without adding defaults.
//
// Version:
//   - 2026-09-23: Added.
func (p Params) Normalize() Params {
	p.MarketType = rule.MarketType(strings.ToLower(strings.TrimSpace(string(p.MarketType))))
	p.BaseAsset = p.BaseAsset.Normalize()
	p.QuoteAsset = p.QuoteAsset.Normalize()
	p.Markets = rule.NormalizeMarkets(p.Markets)
	p.Conditions = p.Conditions.Normalize()
	p.ExecutionRule = p.ExecutionRule.Normalize()
	return p
}

// Validate validates the request structure and exact numeric representations.
// Asset equivalence, market metadata, and executable inventory need server checks.
//
// Version:
//   - 2026-09-24: Enforce the maximum observation window through Conditions validation.
//   - 2026-09-23: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if err := p.MarketType.Validate(); err != nil {
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
	if err := rule.ValidateMarkets(p.Markets); err != nil {
		return fmt.Errorf("failed to validate scalping parameters: %w", err)
	}
	if err := p.Conditions.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping parameters: %w", err)
	}
	if err := p.ExecutionRule.Validate(p.MarketType); err != nil {
		return fmt.Errorf("failed to validate scalping parameters: %w", err)
	}
	return nil
}

// UnmarshalJSON decodes and validates parameters, rejecting unknown fields.
//
// Version:
//   - 2026-09-24: Resolve omitted observation windows while rejecting explicit invalid values.
//   - 2026-09-23: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return v.Invalid("decode scalping parameters", "destination", "null")
	}
	type wire Params
	var decoded wire
	if err := v.Decode(data, &decoded, "marketType", "baseAsset", "quoteAsset", "markets", "conditions", "executionRule"); err != nil {
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
//   - 2026-09-23: Added.
func (c Conditions) Normalize() Conditions {
	c.PriceChangeBPS = normalizeDecimalRange(c.PriceChangeBPS)
	c.BuyVolumeRatioBPS = normalizeDecimalRange(c.BuyVolumeRatioBPS)
	if c.QuoteVolume != nil {
		c.QuoteVolume = &IntegerRange{Minimum: v.StringPointer(c.QuoteVolume.Minimum), Maximum: v.StringPointer(c.QuoteVolume.Maximum)}
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
	if c.PriceChangeBPS == nil && c.QuoteVolume == nil && c.TradeCount == nil && c.BuyVolumeRatioBPS == nil {
		return v.Invalid(op, "conditions", "empty")
	}
	if r := c.PriceChangeBPS; r != nil {
		if err := validateRange(op, "price_change_bps", r.Minimum, r.Maximum, false, true, nil); err != nil {
			return err
		}
	}
	if r := c.QuoteVolume; r != nil {
		if err := validateRange(op, "quote_volume", r.Minimum, r.Maximum, true, false, nil); err != nil {
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
