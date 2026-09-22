package markethub

type ListSymbolsParams struct {
	SourceFilter *ListSourceFilter `json:"sourceFilter,omitempty"`
	// BaseAsset filters by the exact, case-sensitive canonical base asset. Empty selects all assets.
	BaseAsset string                   `json:"baseAsset,omitempty"`
	Venues    []ListSymbolsVenueParams `json:"venues,omitempty"`
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
	LiquidityModel string              `json:"liquidityModel"`
	Name           string              `json:"name"`
	Page           uint64              `json:"page"`
	Limit          uint64              `json:"limit"`
	Total          uint64              `json:"total"`
	Symbols        []ListSymbolsSymbol `json:"symbols"`
}

type ListSymbolsResult struct {
	Venues []ListSymbolsVenue `json:"venues"`
}
