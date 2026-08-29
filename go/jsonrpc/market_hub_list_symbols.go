package jsonrpc

type MarketHubListSymbolsParams struct {
	Venues []MarketHubListSymbolsVenueParams `json:"venues,omitempty"`
}

type MarketHubListSymbolsVenueParams struct {
	Name        string   `json:"name"`
	Page        uint64   `json:"page,omitempty"`
	Limit       uint64   `json:"limit,omitempty"`
	MarketTypes []string `json:"marketTypes,omitempty"`
}

type MarketHubListSymbolsSymbol struct {
	Symbol      string   `json:"symbol"`
	MarketTypes []string `json:"marketTypes"`
}

type MarketHubListSymbolsVenue struct {
	Name    string                       `json:"name"`
	Page    uint64                       `json:"page"`
	Limit   uint64                       `json:"limit"`
	Total   uint64                       `json:"total"`
	Symbols []MarketHubListSymbolsSymbol `json:"symbols"`
}

type MarketHubListSymbolsResult struct {
	Venues []MarketHubListSymbolsVenue `json:"venues"`
}
