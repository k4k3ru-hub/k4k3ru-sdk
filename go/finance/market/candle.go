package market

type CandleInterval string

type Candle struct {
	AssetClass AssetClass `json:"ac"`
	MarketType MarketType `json:"mt"`

	Symbol      Symbol         `json:"s"`
	Venue       Venue          `json:"v"`
	VenueSymbol string         `json:"vs"`
	Interval    CandleInterval `json:"i"`

	Open        string `json:"o"`
	High        string `json:"h"`
	Low         string `json:"l"`
	Close       string `json:"c"`
	BaseVolume  string `json:"bv"`
	QuoteVolume string `json:"qv"`
	TradeCount  uint64 `json:"tc"`

	StartTimestamp int64 `json:"sts"`
	EndTimestamp   int64 `json:"ets"`
	Closed         bool  `json:"closed"`
}
