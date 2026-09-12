package ammpool

type ListResult struct {
	Pools []PoolMetadata `json:"pools"`
}

type PoolMetadata struct {
	Venue         string `json:"venue"`
	Chain         string `json:"chain"`
	Network       string `json:"network"`
	PoolID        string `json:"poolId"`
	Symbol        string `json:"symbol"`
	BaseAssetID   string `json:"baseAssetId"`
	QuoteAssetID  string `json:"quoteAssetId"`
	BaseDecimals  uint8  `json:"baseDecimals"`
	QuoteDecimals uint8  `json:"quoteDecimals"`
	Protocol      string `json:"protocol"`
}
