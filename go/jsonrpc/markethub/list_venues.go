package markethub

type ListVenuesParams struct {
	SourceFilter *ListSourceFilter `json:"sourceFilter,omitempty"`
}

type ListVenuesVenue struct {
	// MarketLiquidityModels identifies each enabled market type when the venue supports multiple models.
	MarketLiquidityModels *MarketLiquidityModels `json:"marketLiquidityModels,omitempty"`
	LiquidityModel        string                 `json:"liquidityModel"`
	Name                  string                 `json:"name"`
	DisplayName           string                 `json:"displayName"`
	Status                string                 `json:"status"`
	UpdatedAt             string                 `json:"updatedAt"`
}

type MarketLiquidityModels struct {
	Spot string `json:"spot,omitempty"`
	Perp string `json:"perp,omitempty"`
}

type ListVenuesResult struct {
	Venues []ListVenuesVenue `json:"venues"`
}
