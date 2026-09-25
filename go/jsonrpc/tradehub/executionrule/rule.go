package executionrule

import (
	"fmt"
	"strings"

	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// Normalize returns an independent rule with normalized scopes and scalar values.
// It does not supply trading thresholds or execution defaults.
//
// Version:
//   - 2026-09-23: Added.
func (r Rule) Normalize() Rule {
	r.Open.Spot = v.Pointer(r.Open.Spot)
	if r.Open.Spot != nil {
		r.Open.Spot.Amount = strings.TrimSpace(r.Open.Spot.Amount)
	}
	r.Open.Perp = v.Pointer(r.Open.Perp)
	if p := r.Open.Perp; p != nil {
		p.Side = PositionSide(strings.ToLower(strings.TrimSpace(string(p.Side))))
		p.Quantity = strings.TrimSpace(p.Quantity)
		p.MarginMode = MarginMode(strings.ToLower(strings.TrimSpace(string(p.MarginMode))))
	}
	r.Open.LimitPrice = v.StringPointer(r.Open.LimitPrice)
	r.Open.MaximumSlippageBPS = v.Pointer(r.Open.MaximumSlippageBPS)
	r.Close.TakeProfit = normalizeTrigger(r.Close.TakeProfit)
	r.Close.StopLoss = normalizeTrigger(r.Close.StopLoss)
	r.Close.MaximumHoldingMS = v.Pointer(r.Close.MaximumHoldingMS)
	r.Close.MaximumSlippageBPS = v.Pointer(r.Close.MaximumSlippageBPS)
	r.Close.Spot = v.Pointer(r.Close.Spot)
	if r.Close.Spot != nil {
		r.Close.Spot.Markets = NormalizeMarkets(r.Close.Spot.Markets)
	}
	return r
}

func normalizeTrigger(trigger *Trigger) *Trigger {
	if trigger == nil {
		return nil
	}
	copy := *trigger
	copy.Type = TriggerType(strings.ToLower(strings.TrimSpace(string(copy.Type))))
	copy.Value = strings.TrimSpace(copy.Value)
	return &copy
}

// Validate validates both legs against the enclosing request's market type.
// Metadata, inventory, margin availability, and actual prices require server checks.
//
// Version:
//   - 2026-09-23: Added.
func (r Rule) Validate(marketType MarketType) error {
	if err := marketType.Validate(); err != nil {
		return fmt.Errorf("failed to validate execution rule: %w", err)
	}
	marketType = MarketType(strings.ToLower(strings.TrimSpace(string(marketType))))
	r = r.Normalize()
	if err := validateOpen(r.Open, marketType); err != nil {
		return fmt.Errorf("failed to validate execution rule: %w", err)
	}
	if err := validateClose(r.Close, marketType); err != nil {
		return fmt.Errorf("failed to validate execution rule: %w", err)
	}
	return nil
}

func validateOpen(open OpenRule, marketType MarketType) error {
	const op = "validate open rule"
	if marketType == MarketTypeSpot {
		if open.Spot == nil || open.Perp != nil {
			return v.Invalid(op, "spot", "invalid")
		}
		if err := positive(op, "amount", open.Spot.Amount, true); err != nil {
			return err
		}
	} else {
		if open.Perp == nil || open.Spot != nil {
			return v.Invalid(op, "perp", "invalid")
		}
		perp := open.Perp
		if perp.Side != PositionSideLong && perp.Side != PositionSideShort {
			return v.Invalid(op, "side", "invalid")
		}
		if err := positive(op, "quantity", perp.Quantity, true); err != nil {
			return err
		}
		if perp.Leverage == 0 {
			return v.Invalid(op, "leverage", "empty")
		}
		if perp.MarginMode != MarginModeCross && perp.MarginMode != MarginModeIsolated {
			return v.Invalid(op, "margin_mode", "invalid")
		}
	}
	if open.LimitPrice != nil {
		if err := positive(op, "limit_price", *open.LimitPrice, false); err != nil {
			return err
		}
	}
	return validateExecutionBounds(op, open.MaximumSlippageBPS, open.ExecutionTTLMS)
}

func validateClose(close CloseRule, marketType MarketType) error {
	const op = "validate close rule"
	if close.TakeProfit == nil && close.StopLoss == nil && close.MaximumHoldingMS == nil {
		return v.Invalid(op, "conditions", "empty")
	}
	if close.TakeProfit != nil {
		if err := close.TakeProfit.Validate(); err != nil {
			return fmt.Errorf("failed to validate close rule: %w: trigger=%q", err, "take_profit")
		}
	}
	if close.StopLoss != nil {
		if err := close.StopLoss.Validate(); err != nil {
			return fmt.Errorf("failed to validate close rule: %w: trigger=%q", err, "stop_loss")
		}
	}
	if close.MaximumHoldingMS != nil && *close.MaximumHoldingMS == 0 {
		return v.Invalid(op, "maximum_holding_ms", "empty")
	}
	if marketType == MarketTypeSpot {
		if close.Spot == nil {
			return v.Invalid(op, "spot", "null")
		}
		if err := ValidateMarkets(close.Spot.Markets); err != nil {
			return fmt.Errorf("failed to validate close rule: %w", err)
		}
	} else if close.Spot != nil {
		return v.Invalid(op, "spot", "invalid")
	}
	return validateExecutionBounds(op, close.MaximumSlippageBPS, close.ExecutionTTLMS)
}

func validateExecutionBounds(operation string, slippage *uint64, ttl uint64) error {
	if slippage == nil {
		return v.Invalid(operation, "maximum_slippage_bps", "null")
	}
	if *slippage > 10000 {
		return v.Invalid(operation, "maximum_slippage_bps", "out_of_range")
	}
	if ttl == 0 {
		return v.Invalid(operation, "execution_ttl_ms", "empty")
	}
	return nil
}

func positive(operation, field, value string, integer bool) error {
	number, err := v.Number(operation, field, value, integer, false)
	if err != nil {
		return err
	}
	if number.Sign() <= 0 {
		return v.Invalid(operation, field, "out_of_range")
	}
	return nil
}

// Validate validates a positive price or a signed return expressed in basis points.
// Trigger direction and return accounting depend on the enclosing execution policy.
//
// Version:
//   - 2026-09-23: Added.
func (t Trigger) Validate() error {
	const op = "validate close trigger"
	t = *normalizeTrigger(&t)
	switch t.Type {
	case TriggerTypePrice:
		return positive(op, "value", t.Value, false)
	case TriggerTypeReturnBPS:
		_, err := v.Number(op, "value", t.Value, false, true)
		return err
	default:
		return v.Invalid(op, "type", "invalid")
	}
}

// UnmarshalJSON decodes a complete round trip, rejecting absent or null legs.
// Validate with the enclosing market type also rejects a mismatched open variant.
//
// Version:
//   - 2026-09-25: Validate perpetual variants with the canonical market type.
//   - 2026-09-23: Added.
func (r *Rule) UnmarshalJSON(data []byte) error {
	if r == nil {
		return v.Invalid("decode execution rule", "destination", "null")
	}
	type wire Rule
	var decoded wire
	if err := v.Decode(data, &decoded, "open", "close"); err != nil {
		return fmt.Errorf("failed to decode execution rule: %w", err)
	}
	value := Rule(decoded).Normalize()
	marketType := MarketTypeSpot
	if value.Open.Perp != nil {
		marketType = MarketTypePerpetual
	}
	if err := value.Validate(marketType); err != nil {
		return fmt.Errorf("failed to decode execution rule: %w", err)
	}
	*r = value
	return nil
}
