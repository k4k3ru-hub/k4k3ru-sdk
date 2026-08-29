package markethub

type ListSymbolsParams struct {
	Venues []ListSymbolsVenueParams `json:"venues,omitempty"`
}

type ListSymbolsVenueParams struct {
	Name        string   `json:"name"`
	Page        uint64   `json:"page,omitempty"`
	Limit       uint64   `json:"limit,omitempty"`
	MarketTypes []string `json:"marketTypes,omitempty"`
}

type ListSymbolsSymbol struct {
	Symbol      string   `json:"symbol"`
	MarketTypes []string `json:"marketTypes"`
}

type ListSymbolsVenue struct {
	Name    string              `json:"name"`
	Page    uint64              `json:"page"`
	Limit   uint64              `json:"limit"`
	Total   uint64              `json:"total"`
	Symbols []ListSymbolsSymbol `json:"symbols"`
}

type ListSymbolsResult struct {
	Venues []ListSymbolsVenue `json:"venues"`
}
