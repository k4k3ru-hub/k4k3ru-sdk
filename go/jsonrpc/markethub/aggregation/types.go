package aggregation

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
	AggregationModeUnknown      AggregationMode = ""
	AggregationModeCompositeMid AggregationMode = "composite-mid"
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

type AggregationSourceFilter struct {
	VenueCategories []VenueCategory  `json:"venueCategories,omitempty"`
	LiquidityModels []LiquidityModel `json:"liquidityModels,omitempty"`
	AMMPoolChains   []Chain          `json:"ammPoolChains,omitempty"`
}
