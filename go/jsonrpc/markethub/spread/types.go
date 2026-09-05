package spread

type AssetClass string
type Symbol string
type Asset string
type Venue string
type MarketType string
type Side string
type RouteFamily string
type VenueCategory string
type LiquidityModel string
type Chain string
type Network string

const (
	AssetClassUnknown       AssetClass     = ""
	AssetClassCrypto        AssetClass     = "crypto"
	MarketTypeUnknown       MarketType     = ""
	MarketTypeSpot          MarketType     = "spot"
	MarketTypePerp          MarketType     = "perp"
	SideUnknown             Side           = ""
	SideBuy                 Side           = "buy"
	SideSell                Side           = "sell"
	RouteFamilySpotSpot     RouteFamily    = "spot-spot"
	RouteFamilySpotPerp     RouteFamily    = "spot-perp"
	RouteFamilyPerpSpot     RouteFamily    = "perp-spot"
	RouteFamilyPerpPerp     RouteFamily    = "perp-perp"
	VenueCategoryCEX        VenueCategory  = "cex"
	VenueCategoryDEX        VenueCategory  = "dex"
	LiquidityModelOrderBook LiquidityModel = "order-book"
	LiquidityModelAMM       LiquidityModel = "amm"
	ChainEthereum           Chain          = "ethereum"
	ChainBase               Chain          = "base"
	ChainBNB                Chain          = "bnb"
	ChainSolana             Chain          = "solana"
	ChainSui                Chain          = "sui"
	NetworkMainnet          Network        = "mainnet"
)

var allRouteFamilies = []RouteFamily{RouteFamilySpotSpot, RouteFamilySpotPerp, RouteFamilyPerpSpot, RouteFamilyPerpPerp}

type SourceFilter struct {
	VenueCategories []VenueCategory  `json:"venueCategories,omitempty"`
	LiquidityModels []LiquidityModel `json:"liquidityModels,omitempty"`
	AMMPoolChains   []Chain          `json:"ammPoolChains,omitempty"`
}
