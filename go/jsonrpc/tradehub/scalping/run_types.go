package scalping

import (
	"math"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observations "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
	"github.com/k4k3ru-hub/onchain/go/core"
)

const (
	DefaultMinimumOrderIntervalMS uint64 = 1_000
	DefaultMaximumUnsettledOrders uint64 = 1
	DefaultReserveBufferBPS       uint64 = 100
	DefaultExecutionTTLMS         uint64 = 30_000
	MaximumRunDurationMS          uint64 = uint64(math.MaxInt64 / int64(time.Millisecond))
)

// RunParams encodes either a flat idempotent start or an execution-ID-only resume.
// Params is flattened by MarshalJSON, without a nested configuration property.
type RunParams struct {
	ExecutionID    string            `json:"executionId,omitempty"`
	IdempotencyKey string            `json:"idempotencyKey,omitempty"`
	Params         *RunConfiguration `json:"-"`
}

// RunConfiguration owns the new Run contract independently of Subscribe's Params.
type RunConfiguration struct {
	MarketType    market.MarketType `json:"marketType"`
	Symbol        market.Symbol     `json:"symbol"`
	TestMode      bool              `json:"testMode,omitempty"`
	Observation   Observation       `json:"observation"`
	ExecutionRule ExecutionRule     `json:"executionRule"`
}

type Observation struct {
	Markets              []MarketSelector         `json:"markets"`
	WindowMS             *uint64                  `json:"windowMs,omitempty"`
	Buy                  *observations.SideParams `json:"buy,omitempty"`
	Sell                 *observations.SideParams `json:"sell,omitempty"`
	MaximumTradeAgeMS    *uint64                  `json:"maximumTradeAgeMs,omitempty"`
	MaximumSnapshotAgeMS *uint64                  `json:"maximumSnapshotAgeMs,omitempty"`
}

// MarketSelector leaves network resolution to TradeHub using TestMode.
type MarketSelector struct {
	Venue       market.Venue `json:"venue"`
	Chain       core.Chain   `json:"chain,omitempty"`
	PoolID      string       `json:"poolId,omitempty"`
	VenueSymbol string       `json:"venueSymbol,omitempty"`
}

type ExecutionMarket struct {
	MarketSelector
	AccountAddress string             `json:"accountAddress"`
	Perpetual      *PerpetualSettings `json:"perpetual,omitempty"`
}

type PerpetualSettings struct {
	Leverage   uint32          `json:"leverage"`
	MarginMode rule.MarginMode `json:"marginMode"`
}

type ExecutionRule struct {
	Markets                []ExecutionMarket `json:"markets"`
	Entry                  EntryRule         `json:"entry"`
	Exit                   ExitRule          `json:"exit"`
	MinimumOrderIntervalMS *uint64           `json:"minimumOrderIntervalMs,omitempty"`
	MaximumUnsettledOrders *uint64           `json:"maximumUnsettledOrders,omitempty"`
	MaximumSlippageBPS     *uint64           `json:"maximumSlippageBps,omitempty"`
	ReserveBufferBPS       *uint64           `json:"reserveBufferBps,omitempty"`
	ExecutionTTLMS         *uint64           `json:"executionTtlMs,omitempty"`
}

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type EntryRule struct {
	Side Side `json:"side"`
	// MaximumQuantity caps Quote payment for Spot Buy, Base sold for Spot Sell,
	// and Base position size for either Perpetual side, before reserve buffers.
	MaximumQuantity market.Quantity `json:"maximumQuantity"`
	Condition       Condition       `json:"condition"`
	// LimitPrice bounds the execution price excluding fees in Quote per Base.
	LimitPrice *string `json:"limitPrice,omitempty"`
}

type ExitRule struct {
	Condition  *Condition    `json:"condition,omitempty"`
	TakeProfit *rule.Trigger `json:"takeProfit,omitempty"`
	StopLoss   *rule.Trigger `json:"stopLoss,omitempty"`
	// MaximumHoldingMS runs from the initial order's first valid fill, including
	// downtime. Further fills do not extend it; reaching it triggers settlement.
	MaximumHoldingMS *uint64 `json:"maximumHoldingMs,omitempty"`
}

// Condition combines all specified metrics with AND, without supplying thresholds.
type Condition struct {
	PriceChangeBPS         *DecimalRange  `json:"priceChangeBps,omitempty"`
	QuoteVolume            *QuantityRange `json:"quoteVolume,omitempty"`
	TradeCount             *CountRange    `json:"tradeCount,omitempty"`
	BuyVolumeRatioBPS      *DecimalRange  `json:"buyVolumeRatioBps,omitempty"`
	PriceChangeDeltaBPS    *DecimalRange  `json:"priceChangeDeltaBps,omitempty"`
	QuoteVolumeChangeBPS   *DecimalRange  `json:"quoteVolumeChangeBps,omitempty"`
	BuyVolumeRatioDeltaBPS *DecimalRange  `json:"buyVolumeRatioDeltaBps,omitempty"`
	RealizedVolatilityBPS  *DecimalRange  `json:"realizedVolatilityBps,omitempty"`
	SpreadBPS              *DecimalRange  `json:"spreadBps,omitempty"`
}
