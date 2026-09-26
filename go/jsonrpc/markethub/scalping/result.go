package scalping

import "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"

type Result struct {
	EvaluatedAt int64         `json:"evaluatedAt"`
	OHLC        *OHLC         `json:"ohlc,omitempty"`
	Metrics     *Metrics      `json:"metrics,omitempty"`
	Buy         []MarketPrice `json:"buy"`
	Sell        []MarketPrice `json:"sell"`
}

type OHLC struct {
	Open  string `json:"open"`
	High  string `json:"high"`
	Low   string `json:"low"`
	Close string `json:"close"`
}

type Metrics struct {
	PriceChangeBPS        *string          `json:"priceChangeBps,omitempty"`
	QuoteVolume           *market.Quantity `json:"quoteVolume,omitempty"`
	TradeCount            *uint64          `json:"tradeCount,omitempty"`
	BuyVolumeRatioBPS     *string          `json:"buyVolumeRatioBps,omitempty"`
	TradeVWAP             *string          `json:"tradeVwap,omitempty"`
	RealizedVolatilityBPS *string          `json:"realizedVolatilityBps,omitempty"`
	Trend                 *Trend           `json:"trend,omitempty"`
	Spread                *SpreadMetric    `json:"spread,omitempty"`
}

type Trend struct {
	PriceChangeDeltaBPS    *string `json:"priceChangeDeltaBps,omitempty"`
	QuoteVolumeChangeBPS   *string `json:"quoteVolumeChangeBps,omitempty"`
	BuyVolumeRatioDeltaBPS *string `json:"buyVolumeRatioDeltaBps,omitempty"`
}

type PriceStatus string

const (
	PriceStatusReference         PriceStatus = "reference"
	PriceStatusVWAP              PriceStatus = "vwap"
	PriceStatusFallbackReference PriceStatus = "fallback_reference"
	PriceStatusUnavailable       PriceStatus = "unavailable"
)

type MarketPrice struct {
	Market market.MarketRef `json:"market"`
	Status PriceStatus      `json:"status"`
	Price  *string          `json:"price,omitempty"`
	// ReceiveQuantity is gross output: Base for Buy and Quote for Sell.
	ReceiveQuantity *market.Quantity `json:"receiveQuantity,omitempty"`
	ObservedAt      *int64           `json:"observedAt,omitempty"`
	// LastTradeAt is the last observed, non-canceled Trade/Swap event time in Unix ms.
	// It is independent of price observation time and omitted when unknown.
	LastTradeAt *int64 `json:"lastTradeAt,omitempty"`
	Fees        *Fees  `json:"fees,omitempty"`
}

type SpreadMetric struct {
	Status PriceStatus `json:"status"`
	BPS    *string     `json:"bps,omitempty"`
}
