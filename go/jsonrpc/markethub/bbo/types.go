package bbo

type AssetClass string

const (
	AssetClassUnknown AssetClass = ""
	AssetClassCrypto  AssetClass = "crypto"
)

type MarketType string

const (
	MarketTypeUnknown MarketType = ""
	MarketTypeSpot    MarketType = "spot"
	MarketTypePerp    MarketType = "perp"
)

type Symbol string

type AggregationMode string

const (
	AggregationModeUnknown         AggregationMode = ""
	AggregationModeConsolidatedBBO AggregationMode = "consolidated-bbo"
)

type VenueCategory string

const (
	VenueCategoryUnknown VenueCategory = ""
	VenueCategoryCEX     VenueCategory = "cex"
	VenueCategoryDEX     VenueCategory = "dex"
)

type LiquidityModel string

const (
	LiquidityModelUnknown   LiquidityModel = ""
	LiquidityModelOrderBook LiquidityModel = "order-book"
	LiquidityModelAMM       LiquidityModel = "amm"
)

type Chain string

const (
	ChainUnknown  Chain = ""
	ChainNone     Chain = "none"
	ChainEthereum Chain = "ethereum"
	ChainBase     Chain = "base"
	ChainBNB      Chain = "bnb"
	ChainSolana   Chain = "solana"
	ChainSui      Chain = "sui"
)

type SourceFilter struct {
	VenueCategories []VenueCategory  `json:"venueCategories,omitempty"`
	LiquidityModels []LiquidityModel `json:"liquidityModels,omitempty"`
	AMMPoolChains   []Chain          `json:"ammPoolChains,omitempty"`
}

type Level struct {
	Price    string `json:"p"`
	Quantity string `json:"q"`
}
