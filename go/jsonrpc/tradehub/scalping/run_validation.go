package scalping

import (
	"fmt"
	"math/big"
	"strings"
	"unicode"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observations "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// Validate requires a complete idempotent start or an execution-ID-only resume.
//
// Version:
//   - 2026-10-01: Added.
//   - 2026-10-03: Enforce product-specific order and position limits.
func (p RunParams) Validate() error {
	p = p.Normalize()
	const op = "validate scalping run parameters"
	if p.ExecutionID != "" {
		if p.Params != nil || p.IdempotencyKey != "" {
			return v.Invalid(op, "resume_overrides", "invalid")
		}
		return v.Text(op, "execution_id", p.ExecutionID, 128)
	}
	if err := v.Text(op, "idempotency_key", p.IdempotencyKey, 128); err != nil {
		return err
	}
	if p.Params == nil {
		return v.Invalid(op, "parameters", "null")
	}
	if err := p.Params.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping run parameters: %w", err)
	}
	return nil
}

// Validate checks normalized structure and exact quantities without mutating inputs.
// TradeHub must resolve markets, verify the execution subset and account scopes,
// and validate venue-specific precision, leverage, environment and inventory.
//
// Version:
//   - 2026-10-01: Added.
//   - 2026-10-03: Reject Perpetual order counts and Spot position settings.
func (p RunConfiguration) Validate() error {
	const op = "validate scalping run configuration"
	p = p.Normalize()
	if err := validateMarketType(p.MarketType); err != nil {
		return fmt.Errorf("failed to validate scalping run configuration: %w", err)
	}
	if err := p.Symbol.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping run configuration: %w: %w", apperror.InvalidParameter(), err)
	}
	parts := strings.Split(string(p.Symbol), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == parts[1] || strings.ContainsAny(string(p.Symbol), ",;*?[]") || strings.IndexFunc(string(p.Symbol), func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return v.Invalid(op, "symbol", "invalid")
	}
	if err := validateRunObservation(p.Observation); err != nil {
		return fmt.Errorf("failed to validate scalping run configuration: %w", err)
	}
	if err := validateRunExecution(p.ExecutionRule, p.MarketType, p.Observation.Markets); err != nil {
		return fmt.Errorf("failed to validate scalping run configuration: %w", err)
	}
	return nil
}

func validateRunObservation(o Observation) error {
	const op = "validate scalping run observation"
	if err := validateRunMarketCount(op, len(o.Markets)); err != nil {
		return err
	}
	seen := make(map[MarketSelector]bool, len(o.Markets))
	for i, m := range o.Markets {
		if err := validateRunMarket(m); err != nil {
			return fmt.Errorf("failed to validate scalping run observation: %w: market_index=%d", err, i)
		}
		if seen[m] {
			return v.Invalid(op, "duplicate_market", "invalid")
		}
		seen[m] = true
	}
	if *o.WindowMS == 0 || *o.WindowMS > MaximumWindowMS {
		return v.Invalid(op, "window_ms", "out_of_range")
	}
	for _, duration := range []struct {
		name  string
		value *uint64
	}{
		{"maximum_trade_age_ms", o.MaximumTradeAgeMS}, {"maximum_snapshot_age_ms", o.MaximumSnapshotAgeMS},
	} {
		if err := validateRunDuration(op, duration.name, duration.value, false); err != nil {
			return err
		}
	}
	for _, side := range []*observations.SideParams{o.Buy, o.Sell} {
		if side != nil {
			if side.Kind != "" && side.Kind != observations.KindExactInput {
				return v.Invalid(op, "observation_kind", "invalid")
			}
			if err := side.Validate(); err != nil {
				return fmt.Errorf("failed to validate scalping run observation: %w", err)
			}
		}
	}
	return nil
}

func validateRunMarketCount(op string, count int) error {
	if count == 0 {
		return v.Invalid(op, "markets", "empty")
	}
	if count > observations.MaximumMarkets {
		return v.Invalid(op, "markets", "too_long")
	}
	return nil
}

func validateRunMarket(m MarketSelector) error {
	const op = "validate scalping run market"
	if err := m.Venue.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping run market: %w: %w", apperror.InvalidParameter(), err)
	}
	if m.Chain != "" {
		if err := m.Chain.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping run market: %w: %w", apperror.InvalidParameter(), err)
		}
	}
	if m.PoolID != "" && m.VenueSymbol != "" {
		return v.Invalid(op, "market_identifiers", "invalid")
	}
	for _, id := range []struct{ name, value string }{{"pool_id", m.PoolID}, {"venue_symbol", m.VenueSymbol}} {
		if id.value != "" {
			if err := v.Text(op, id.name, id.value, 256); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateRunExecution(r ExecutionRule, marketType market.MarketType, observed []MarketSelector) error {
	const op = "validate scalping run execution rule"
	if err := validateRunMarketCount(op, len(r.Markets)); err != nil {
		return err
	}
	seen := make(map[MarketSelector]bool, len(r.Markets))
	for i, m := range r.Markets {
		if err := validateRunMarket(m.MarketSelector); err != nil {
			return fmt.Errorf("failed to validate scalping run execution rule: %w: market_index=%d", err, i)
		}
		if seen[m.MarketSelector] {
			return v.Invalid(op, "duplicate_market", "invalid")
		}
		seen[m.MarketSelector] = true
		if err := v.Text(op, "account_address", m.AccountAddress, 256); err != nil {
			return err
		}
		if !runMarketMayBeObserved(m.MarketSelector, observed) {
			return v.Invalid(op, "execution_market_scope", "invalid")
		}
		if marketType == market.MarketTypeSpot {
			if m.Perpetual != nil {
				return v.Invalid(op, "perpetual", "invalid")
			}
		} else {
			if m.Perpetual == nil {
				return v.Invalid(op, "perpetual", "null")
			}
			if m.Perpetual.Leverage == 0 {
				return v.Invalid(op, "leverage", "empty")
			}
			if m.Perpetual.MarginMode != rule.MarginModeCross && m.Perpetual.MarginMode != rule.MarginModeIsolated {
				return v.Invalid(op, "margin_mode", "invalid")
			}
		}
	}
	if r.Entry.Side != SideBuy && r.Entry.Side != SideSell {
		return v.Invalid(op, "side", "invalid")
	}
	if err := r.Entry.MaximumQuantity.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping run execution rule: %w", err)
	}
	if scaledQuantity(r.Entry.MaximumQuantity).Sign() == 0 {
		return v.Invalid(op, "maximum_quantity", "empty")
	}
	if err := r.Entry.Condition.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping run execution rule: %w: rule=%q", err, "entry")
	}
	if r.Entry.LimitPrice != nil {
		price, err := v.Number(op, "limit_price", *r.Entry.LimitPrice, false, false)
		if err != nil {
			return err
		}
		if price.Sign() <= 0 {
			return v.Invalid(op, "limit_price", "out_of_range")
		}
	}
	if err := validateRunExit(r.Exit, r.Entry.Side); err != nil {
		return fmt.Errorf("failed to validate scalping run execution rule: %w", err)
	}
	if err := validateRunDuration(op, "minimum_order_interval_ms", r.MinimumOrderIntervalMS, true); err != nil {
		return err
	}
	if err := validateRunDuration(op, "execution_ttl_ms", r.ExecutionTTLMS, false); err != nil {
		return err
	}
	switch marketType {
	case market.MarketTypeSpot:
		if r.Perpetual != nil {
			return v.Invalid(op, "perpetual", "invalid")
		}
		if r.MaximumUnsettledOrders == nil || *r.MaximumUnsettledOrders == 0 {
			return v.Invalid(op, "maximum_unsettled_orders", "empty")
		}
	case market.MarketTypePerpetual:
		if r.MaximumUnsettledOrders != nil {
			return v.Invalid(op, "maximum_unsettled_orders", "invalid")
		}
		if r.Perpetual == nil || r.Perpetual.MaximumPositionQuantity == nil {
			return v.Invalid(op, "maximum_position_quantity", "null")
		}
		q := *r.Perpetual.MaximumPositionQuantity
		if err := q.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping run position limit: %w", err)
		}
		if scaledQuantity(q).Sign() == 0 {
			return v.Invalid(op, "maximum_position_quantity", "empty")
		}
	}
	if *r.MaximumSlippageBPS >= 10_000 {
		return v.Invalid(op, "maximum_slippage_bps", "out_of_range")
	}
	return nil
}

// Selectors with omitted identifiers need catalog resolution. This only rejects
// explicit conflicts; it does not prove the required resolved subset relation.
func runMarketMayBeObserved(m MarketSelector, observed []MarketSelector) bool {
	for _, o := range observed {
		if m.Venue != o.Venue || (m.Chain != "" && o.Chain != "" && m.Chain != o.Chain) ||
			(m.PoolID != "" && o.PoolID != "" && m.PoolID != o.PoolID) ||
			(m.VenueSymbol != "" && o.VenueSymbol != "" && m.VenueSymbol != o.VenueSymbol) {
			continue
		}
		return true
	}
	return false
}

func validateRunDuration(op, field string, value *uint64, allowZero bool) error {
	if value == nil {
		return nil
	}
	if *value == 0 && !allowZero {
		return v.Invalid(op, field, "empty")
	}
	if *value > MaximumRunDurationMS {
		return v.Invalid(op, field, "out_of_range")
	}
	return nil
}

func validateRunExit(exit ExitRule, side Side) error {
	const op = "validate scalping run exit"
	if exit.Condition == nil && exit.TakeProfit == nil && exit.StopLoss == nil && exit.MaximumHoldingMS == nil {
		return v.Invalid(op, "exit_conditions", "empty")
	}
	if exit.Condition != nil {
		if err := exit.Condition.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping run exit: %w", err)
		}
	}
	var values [2]*big.Rat
	for i, t := range []*rule.Trigger{exit.TakeProfit, exit.StopLoss} {
		if t == nil {
			continue
		}
		if err := t.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping run exit: %w: trigger_index=%d", err, i)
		}
		n, err := v.Number(op, "trigger_value", t.Value, false, true)
		if err != nil {
			return err
		}
		if t.Type == rule.TriggerTypeReturnBPS && ((i == 0 && n.Sign() <= 0) || (i == 1 && n.Sign() >= 0)) {
			return v.Invalid(op, "trigger_sign", "invalid")
		}
		values[i] = n
	}
	if values[0] != nil && values[1] != nil && exit.TakeProfit.Type == exit.StopLoss.Type {
		comparison := values[0].Cmp(values[1])
		if (exit.TakeProfit.Type == rule.TriggerTypePrice && side == SideSell && comparison >= 0) ||
			((exit.TakeProfit.Type != rule.TriggerTypePrice || side == SideBuy) && comparison <= 0) {
			return v.Invalid(op, "trigger_order", "invalid")
		}
	}
	return validateRunDuration(op, "maximum_holding_ms", exit.MaximumHoldingMS, false)
}

// Validate checks inclusive AND bounds for the nine supported observation metrics.
// Missing specified metrics must hold execution; omitted metrics impose no test.
//
// Version:
//   - 2026-10-01: Added.
func (c Condition) Validate() error {
	const op = "validate scalping run condition"
	c = c.Normalize()
	count := 0
	for _, item := range []struct {
		name         string
		bounds       *DecimalRange
		signed       bool
		lower, upper *big.Rat
	}{
		{"price_change_bps", c.PriceChangeBPS, true, nil, nil},
		{"buy_volume_ratio_bps", c.BuyVolumeRatioBPS, false, nil, big.NewRat(10000, 1)},
		{"price_change_delta_bps", c.PriceChangeDeltaBPS, true, nil, nil},
		{"quote_volume_change_bps", c.QuoteVolumeChangeBPS, true, nil, nil},
		{"buy_volume_ratio_delta_bps", c.BuyVolumeRatioDeltaBPS, true, big.NewRat(-10000, 1), big.NewRat(10000, 1)},
		{"realized_volatility_bps", c.RealizedVolatilityBPS, false, nil, nil},
		{"spread_bps", c.SpreadBPS, true, nil, nil},
	} {
		if item.bounds == nil {
			continue
		}
		count++
		if err := validateRange(op, item.name, item.bounds.Minimum, item.bounds.Maximum, false, item.signed, item.upper); err != nil {
			return err
		}
		if item.lower != nil {
			for _, bound := range []*string{item.bounds.Minimum, item.bounds.Maximum} {
				if bound == nil {
					continue
				}
				n, err := v.Number(op, item.name, *bound, false, item.signed)
				if err != nil {
					return err
				}
				if n.Cmp(item.lower) < 0 {
					return v.Invalid(op, item.name, "out_of_range")
				}
			}
		}
	}
	if c.QuoteVolume != nil {
		count++
		if err := c.QuoteVolume.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping run condition: %w", err)
		}
	}
	if r := c.TradeCount; r != nil {
		count++
		if r.Minimum == nil && r.Maximum == nil {
			return v.Invalid(op, "trade_count", "empty")
		}
		if r.Minimum != nil && r.Maximum != nil && *r.Minimum > *r.Maximum {
			return v.Invalid(op, "trade_count", "out_of_range")
		}
	}
	if count == 0 {
		return v.Invalid(op, "condition", "empty")
	}
	return nil
}
