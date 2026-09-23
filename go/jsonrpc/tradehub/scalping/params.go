package scalping

import rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"

type Params struct {
	MarketType    rule.MarketType  `json:"marketType"`
	BaseAsset     rule.AssetRef    `json:"baseAsset"`
	QuoteAsset    rule.AssetRef    `json:"quoteAsset"`
	Markets       []rule.MarketRef `json:"markets"`
	Conditions    Conditions       `json:"conditions"`
	ExecutionRule rule.Rule        `json:"executionRule"`
}

type Conditions struct {
	WindowMS          uint64        `json:"windowMs"`
	MaximumDataAgeMS  uint64        `json:"maximumDataAgeMs"`
	PriceChangeBPS    *DecimalRange `json:"priceChangeBps,omitempty"`
	QuoteVolume       *IntegerRange `json:"quoteVolume,omitempty"`
	TradeCount        *CountRange   `json:"tradeCount,omitempty"`
	BuyVolumeRatioBPS *DecimalRange `json:"buyVolumeRatioBps,omitempty"`
}

type DecimalRange struct {
	Minimum *string `json:"minimum,omitempty"`
	Maximum *string `json:"maximum,omitempty"`
}

type IntegerRange struct {
	Minimum *string `json:"minimum,omitempty"`
	Maximum *string `json:"maximum,omitempty"`
}

type CountRange struct {
	Minimum *uint64 `json:"minimum,omitempty"`
	Maximum *uint64 `json:"maximum,omitempty"`
}
