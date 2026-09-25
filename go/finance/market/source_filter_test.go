package market

import "testing"

// TestAggregationSourceFilterMatchesCetus verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestAggregationSourceFilterMatchesCetus(t *testing.T) {
	t.Parallel()

	cetusSui := MarketSourceKey{Venue: Cetus, Chain: ChainSui}
	tests := []struct {
		name   string
		filter AggregationSourceFilter
		want   bool
	}{
		{
			name:   "dex",
			filter: AggregationSourceFilter{VenueCategories: []VenueCategory{VenueCategoryDEX}},
			want:   true,
		},
		{
			name: "dex amm",
			filter: AggregationSourceFilter{
				VenueCategories: []VenueCategory{VenueCategoryDEX},
				LiquidityModels: []LiquidityModel{LiquidityModelAMM},
			},
			want: true,
		},
		{
			name: "sui amm pool",
			filter: AggregationSourceFilter{
				VenueCategories: []VenueCategory{VenueCategoryDEX},
				LiquidityModels: []LiquidityModel{LiquidityModelAMM},
				AMMPoolChains:   []Chain{ChainSui},
			},
			want: true,
		},
		{
			name: "base amm pool",
			filter: AggregationSourceFilter{
				VenueCategories: []VenueCategory{VenueCategoryDEX},
				LiquidityModels: []LiquidityModel{LiquidityModelAMM},
				AMMPoolChains:   []Chain{ChainBase},
			},
			want: false,
		},
		{
			name:   "cex",
			filter: AggregationSourceFilter{VenueCategories: []VenueCategory{VenueCategoryCEX}},
			want:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.filter.Matches(cetusSui); got != test.want {
				t.Fatalf("Matches() = %t, want %t", got, test.want)
			}
		})
	}
}

// TestAggregationSourceFilterMatchesBluefin verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestAggregationSourceFilterMatchesBluefin(t *testing.T) {
	t.Parallel()

	filter := AggregationSourceFilter{VenueCategories: []VenueCategory{VenueCategoryDEX}, LiquidityModels: []LiquidityModel{LiquidityModelAMM}, AMMPoolChains: []Chain{ChainSui}}
	if !filter.Matches(MarketSourceKey{Venue: Bluefin, Chain: ChainSui}) {
		t.Fatal("Matches(bluefin-sui) = false, want true")
	}
}

// TestAggregationSourceFilterMatchesTurbos verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestAggregationSourceFilterMatchesTurbos(t *testing.T) {
	t.Parallel()

	filter := AggregationSourceFilter{VenueCategories: []VenueCategory{VenueCategoryDEX}, LiquidityModels: []LiquidityModel{LiquidityModelAMM}, AMMPoolChains: []Chain{ChainSui}}
	if !filter.Matches(MarketSourceKey{Venue: Turbos, Chain: ChainSui}) {
		t.Fatal("Matches(turbos-sui) = false, want true")
	}
}

// TestAggregationSourceFilterMatchesMomentum verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestAggregationSourceFilterMatchesMomentum(t *testing.T) {
	t.Parallel()

	filter := AggregationSourceFilter{VenueCategories: []VenueCategory{VenueCategoryDEX}, LiquidityModels: []LiquidityModel{LiquidityModelAMM}, AMMPoolChains: []Chain{ChainSui}}
	if !filter.Matches(MarketSourceKey{Venue: Momentum, Chain: ChainSui}) {
		t.Fatal("Matches(momentum-sui) = false, want true")
	}
}

// TestAggregationSourceFilterMatchesSolanaAMMs verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestAggregationSourceFilterMatchesSolanaAMMs(t *testing.T) {
	t.Parallel()

	filter := AggregationSourceFilter{
		VenueCategories: []VenueCategory{VenueCategoryDEX},
		LiquidityModels: []LiquidityModel{LiquidityModelAMM},
		AMMPoolChains:   []Chain{ChainSolana},
	}
	for _, venue := range []Venue{Meteora, Raydium} {
		if !filter.Matches(MarketSourceKey{Venue: venue, Chain: ChainSolana}) {
			t.Fatalf("Matches(%q) = false, want true", venue)
		}
	}
}
