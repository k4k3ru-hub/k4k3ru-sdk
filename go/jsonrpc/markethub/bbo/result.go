package bbo

type Result struct {
	AssetClass       AssetClass      `json:"ac"`
	MarketType       MarketType      `json:"mt"`
	Symbol           Symbol          `json:"s"`
	AggregationMode  AggregationMode `json:"am"`
	Bid              Level           `json:"b"`
	Ask              Level           `json:"a"`
	SourceVenueCount uint16          `json:"svc"`
	Version          uint64          `json:"v"`
	Timestamp        int64           `json:"ts"`
	SourceFilter     *SourceFilter   `json:"sourceFilter,omitempty"`
}
