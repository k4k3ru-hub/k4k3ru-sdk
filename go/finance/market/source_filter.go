package market

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

type AggregationSourceFilter struct {
	VenueCategories []VenueCategory  `json:"venueCategories,omitempty"`
	LiquidityModels []LiquidityModel `json:"liquidityModels,omitempty"`
	AMMPoolChains   []Chain          `json:"ammPoolChains,omitempty"`
}

func containsVenueCategory(values []VenueCategory, target VenueCategory) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func containsLiquidityModel(values []LiquidityModel, target LiquidityModel) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (f AggregationSourceFilter) matches(source MarketSourceKey) bool {
	category, liquidityModel, ok := sourceClassification(source)
	if !ok {
		return false
	}
	if f.VenueCategories != nil && !containsVenueCategory(f.VenueCategories, category) {
		return false
	}
	if f.LiquidityModels != nil && !containsLiquidityModel(f.LiquidityModels, liquidityModel) {
		return false
	}
	if f.AMMPoolChains != nil && liquidityModel == LiquidityModelAMM {
		for _, chain := range f.AMMPoolChains {
			if chain == source.Chain {
				return true
			}
		}
		return false
	}
	return true
}

// Matches reports whether a market source satisfies the aggregation source filter.
//
// Parameters:
//   - source: Market source key.
//
// Returns:
//   - True when the source matches.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-07: Classified Arcus perpetual order books.
//   - 2026-09-06: Classified both Lighter deployments as DEX order books.
//   - 2026-08-30: Added.
func (f AggregationSourceFilter) Matches(source MarketSourceKey) bool {
	return f.matches(source)
}

func sourceClassification(source MarketSourceKey) (VenueCategory, LiquidityModel, bool) {
	source = source.Normalize()
	switch source.Venue {
	case Binance, BTSE, Bybit, Coinbase, OKX:
		return VenueCategoryCEX, LiquidityModelOrderBook, true
	case Arcus, DYDX, Hyperliquid, Lighter, LighterRobinhood:
		return VenueCategoryDEX, LiquidityModelOrderBook, true
	case Aerodrome, Bluefin, Cetus, Meteora, Momentum, Raydium, Turbos, UniswapV3, UniswapV4:
		return VenueCategoryDEX, LiquidityModelAMM, true
	default:
		return VenueCategoryUnknown, LiquidityModelUnknown, false
	}
}

// LiquidityModel returns the venue's centrally configured liquidity model.
// Unknown venues return LiquidityModelUnknown.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-22: Added.
func (v Venue) LiquidityModel() LiquidityModel {
	_, model, _ := sourceClassification(MarketSourceKey{Venue: v})
	return model
}
