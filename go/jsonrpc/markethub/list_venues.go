package markethub

type ListVenuesParams struct {
	SourceFilter *ListSourceFilter `json:"sourceFilter,omitempty"`
}

type ListVenuesVenue struct {
	LiquidityModel string `json:"liquidityModel"`
	Name           string `json:"name"`
	DisplayName    string `json:"displayName"`
	Status         string `json:"status"`
	UpdatedAt      string `json:"updatedAt"`
}

type ListVenuesResult struct {
	Venues []ListVenuesVenue `json:"venues"`
}
