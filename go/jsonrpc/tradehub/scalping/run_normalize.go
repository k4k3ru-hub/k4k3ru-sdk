package scalping

import (
	"strings"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observations "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// Normalize returns detached parameters with defaults and zero-return triggers removed.
// Explicit zero controls are preserved, and resume requests gain no configuration.
//
// Version:
//   - 2026-10-01: Added.
//   - 2026-10-03: Default position limits for Perpetual and order counts for Spot.
func (p RunParams) Normalize() RunParams {
	p.ExecutionID = strings.TrimSpace(p.ExecutionID)
	p.IdempotencyKey = strings.TrimSpace(p.IdempotencyKey)
	if p.Params != nil {
		config := p.Params.Normalize()
		p.Params = &config
	}
	return p
}

// Normalize copies every mutable field and defaults only omitted controls.
// Malformed triggers remain available to Validate instead of being discarded.
//
// Version:
//   - 2026-10-01: Added.
//   - 2026-10-03: Separate and detach Run-wide Perpetual position limits.
func (p RunConfiguration) Normalize() RunConfiguration {
	p.MarketType = p.MarketType.Normalize()
	p.Symbol = market.Symbol(strings.ToUpper(strings.TrimSpace(string(p.Symbol))))
	o := &p.Observation
	o.Markets = append([]MarketSelector(nil), o.Markets...)
	for i := range o.Markets {
		o.Markets[i] = normalizeRunMarket(o.Markets[i])
	}
	o.WindowMS = runDefault(o.WindowMS, DefaultWindowMS)
	o.MaximumTradeAgeMS = v.Pointer(o.MaximumTradeAgeMS)
	o.MaximumSnapshotAgeMS = v.Pointer(o.MaximumSnapshotAgeMS)
	o.Buy, o.Sell = copyRunSide(o.Buy), copyRunSide(o.Sell)
	r := &p.ExecutionRule
	r.Markets = append([]ExecutionMarket(nil), r.Markets...)
	for i := range r.Markets {
		m := &r.Markets[i]
		m.MarketSelector = normalizeRunMarket(m.MarketSelector)
		m.AccountAddress = strings.TrimSpace(m.AccountAddress)
		m.Perpetual = v.Pointer(m.Perpetual)
	}
	r.Entry.Side = Side(strings.ToLower(strings.TrimSpace(string(r.Entry.Side))))
	r.Entry.LimitPrice = v.StringPointer(r.Entry.LimitPrice)
	r.Entry.Condition = r.Entry.Condition.Normalize()
	if r.Exit.Condition != nil {
		condition := r.Exit.Condition.Normalize()
		r.Exit.Condition = &condition
	}
	r.Exit.TakeProfit = normalizeRunTrigger(r.Exit.TakeProfit)
	r.Exit.StopLoss = normalizeRunTrigger(r.Exit.StopLoss)
	r.Exit.MaximumHoldingMS = v.Pointer(r.Exit.MaximumHoldingMS)
	r.MinimumOrderIntervalMS = runDefault(r.MinimumOrderIntervalMS, DefaultMinimumOrderIntervalMS)
	r.MaximumUnsettledOrders = v.Pointer(r.MaximumUnsettledOrders)
	r.Perpetual = v.Pointer(r.Perpetual)
	if r.Perpetual != nil {
		r.Perpetual.MaximumPositionQuantity = v.Pointer(r.Perpetual.MaximumPositionQuantity)
	}
	switch p.MarketType {
	case market.MarketTypeSpot:
		r.MaximumUnsettledOrders = runDefault(r.MaximumUnsettledOrders, DefaultMaximumUnsettledOrders)
	case market.MarketTypePerpetual:
		if r.Perpetual == nil {
			r.Perpetual = &PerpetualRunSettings{}
		}
		if r.Perpetual.MaximumPositionQuantity == nil {
			r.Perpetual.MaximumPositionQuantity = v.Pointer(&r.Entry.MaximumQuantity)
		}
	}
	r.MaximumSlippageBPS = runDefault(r.MaximumSlippageBPS, rule.DefaultMaximumSlippageBPS)
	r.ReserveBufferBPS = runDefault(r.ReserveBufferBPS, DefaultReserveBufferBPS)
	r.ExecutionTTLMS = runDefault(r.ExecutionTTLMS, DefaultExecutionTTLMS)
	return p
}

// Normalize copies metric bounds without introducing an omitted condition.
//
// Version:
//   - 2026-10-01: Added.
func (c Condition) Normalize() Condition {
	c.PriceChangeBPS = normalizeDecimalRange(c.PriceChangeBPS)
	c.BuyVolumeRatioBPS = normalizeDecimalRange(c.BuyVolumeRatioBPS)
	c.PriceChangeDeltaBPS = normalizeDecimalRange(c.PriceChangeDeltaBPS)
	c.QuoteVolumeChangeBPS = normalizeDecimalRange(c.QuoteVolumeChangeBPS)
	c.BuyVolumeRatioDeltaBPS = normalizeDecimalRange(c.BuyVolumeRatioDeltaBPS)
	c.RealizedVolatilityBPS = normalizeDecimalRange(c.RealizedVolatilityBPS)
	c.SpreadBPS = normalizeDecimalRange(c.SpreadBPS)
	if c.QuoteVolume != nil {
		c.QuoteVolume = &QuantityRange{Minimum: v.Pointer(c.QuoteVolume.Minimum), Maximum: v.Pointer(c.QuoteVolume.Maximum)}
	}
	if c.TradeCount != nil {
		c.TradeCount = &CountRange{Minimum: v.Pointer(c.TradeCount.Minimum), Maximum: v.Pointer(c.TradeCount.Maximum)}
	}
	return c
}

func runDefault(value *uint64, fallback uint64) *uint64 {
	if value == nil {
		return &fallback
	}
	return v.Pointer(value)
}

func copyRunSide(side *observations.SideParams) *observations.SideParams {
	if side == nil || side.Quantity == nil && side.Kind == "" {
		return nil
	}
	kind := side.Kind
	if kind == observations.KindExactInput {
		kind = ""
	}
	return &observations.SideParams{Kind: kind, Quantity: v.Pointer(side.Quantity)}
}

func normalizeRunMarket(m MarketSelector) MarketSelector {
	r := (market.MarketRef{Venue: m.Venue, Chain: m.Chain, PoolID: m.PoolID, VenueSymbol: m.VenueSymbol}).Normalize()
	return MarketSelector{Venue: r.Venue, Chain: r.Chain, PoolID: r.PoolID, VenueSymbol: r.VenueSymbol}
}

func normalizeRunTrigger(trigger *rule.Trigger) *rule.Trigger {
	if trigger == nil {
		return nil
	}
	result := *trigger
	result.Value = strings.TrimSpace(result.Value)
	if result.Type == rule.TriggerTypeReturnBPS {
		n, err := v.Number("normalize scalping run trigger", "value", result.Value, false, true)
		// Keep invalid input intact so validation cannot turn it into an omission.
		if err == nil && n.Sign() == 0 {
			return nil
		}
	}
	return &result
}
