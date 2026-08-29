package jsonrpc

type MarketHubListVenuesParams struct{}

type MarketHubListVenuesVenue struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
}

type MarketHubListVenuesResult struct {
	Venues []MarketHubListVenuesVenue `json:"venues"`
}
