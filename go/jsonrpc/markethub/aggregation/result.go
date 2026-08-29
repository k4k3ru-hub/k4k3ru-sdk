package aggregation

type Result struct {
	AssetClass       AssetClass               `json:"ac"`
	MarketType       MarketType               `json:"mt"`
	Symbol           Symbol                   `json:"s"`
	AggregationMode  AggregationMode          `json:"am"`
	Price            string                   `json:"p"`
	SourceVenueCount uint16                   `json:"svc"`
	Timestamp        int64                    `json:"ts"`
	SourceFilter     *AggregationSourceFilter `json:"sourceFilter,omitempty"`
}
