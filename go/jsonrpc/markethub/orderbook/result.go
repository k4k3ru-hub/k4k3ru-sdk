package orderbook

type Result struct {
	AssetClass       AssetClass      `json:"ac"`
	MarketType       MarketType      `json:"mt"`
	Symbol           Symbol          `json:"s"`
	Depth            uint16          `json:"d"`
	AggregationMode  AggregationMode `json:"am"`
	Bids             []Level         `json:"b"`
	Asks             []Level         `json:"a"`
	SourceVenueCount uint16          `json:"svc"`
	Version          uint64          `json:"v"`
	Timestamp        int64           `json:"ts"`
	Synchronized     bool            `json:"sync"`
	SourceFilter     *SourceFilter   `json:"sourceFilter,omitempty"`
}
