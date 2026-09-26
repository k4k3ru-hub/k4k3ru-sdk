package scalping

import (
	market "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
)

const (
	DefaultWindowMS uint64 = 60_000
	MaximumWindowMS uint64 = 60_000
)

type Params struct {
	MarketType    market.MarketType     `json:"marketType"`
	Symbol        market.Symbol         `json:"symbol"`
	BaseAsset     rule.AssetRef         `json:"baseAsset"`
	QuoteAsset    rule.AssetRef         `json:"quoteAsset"`
	Markets       []market.MarketTarget `json:"markets"`
	BaseQuantity  *market.Quantity      `json:"baseQuantity,omitempty"`
	Conditions    Conditions            `json:"conditions"`
	ExecutionRule rule.Rule             `json:"executionRule"`
}

type Conditions struct {
	// WindowMS uses DefaultWindowMS when omitted from JSON. Go callers must
	// supply a positive value explicitly; zero is invalid in both forms.
	WindowMS             uint64         `json:"windowMs"`
	MaximumDataAgeMS     uint64         `json:"maximumDataAgeMs"`
	MaximumSnapshotAgeMS *uint64        `json:"maximumSnapshotAgeMs,omitempty"`
	PriceChangeBPS       *DecimalRange  `json:"priceChangeBps,omitempty"`
	QuoteVolume          *QuantityRange `json:"quoteVolume,omitempty"`
	TradeCount           *CountRange    `json:"tradeCount,omitempty"`
	BuyVolumeRatioBPS    *DecimalRange  `json:"buyVolumeRatioBps,omitempty"`
}

type DecimalRange struct {
	Minimum *string `json:"minimum,omitempty"`
	Maximum *string `json:"maximum,omitempty"`
}

type QuantityRange struct {
	Minimum *market.Quantity `json:"minimum,omitempty"`
	Maximum *market.Quantity `json:"maximum,omitempty"`
}

type CountRange struct {
	Minimum *uint64 `json:"minimum,omitempty"`
	Maximum *uint64 `json:"maximum,omitempty"`
}
