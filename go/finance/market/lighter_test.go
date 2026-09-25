package market

import "testing"

// TestLighterSourcesAreIndependentDEXOrderBooks verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestLighterSourcesAreIndependentDEXOrderBooks(t *testing.T) {
	filter := AggregationSourceFilter{VenueCategories: []VenueCategory{VenueCategoryDEX}, LiquidityModels: []LiquidityModel{LiquidityModelOrderBook}}
	for _, v := range []Venue{Lighter, LighterRobinhood} {
		source := MarketSourceKey{Venue: v, Chain: ChainNone}
		if err := source.Validate(); err != nil {
			t.Fatal(err)
		}
		if !filter.Matches(source) {
			t.Fatalf("missing source %s", v)
		}
		if (AggregationSourceFilter{LiquidityModels: []LiquidityModel{LiquidityModelAMM}}).Matches(source) {
			t.Fatal("classified as AMM")
		}
	}
}
