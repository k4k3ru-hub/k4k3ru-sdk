package aggregation

type Params struct {
	AssetClass      AssetClass               `json:"assetClass,omitempty"`
	MarketType      MarketType               `json:"marketType"`
	Symbol          Symbol                   `json:"symbol"`
	AggregationMode AggregationMode          `json:"aggregationMode,omitempty"`
	SourceFilter    *AggregationSourceFilter `json:"sourceFilter,omitempty"`
}
