package market

import "testing"

// TestVenueDisplayName verifies brand capitalization and source disambiguation.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-22: Added.
func TestVenueDisplayName(t *testing.T) {
	for venue, want := range map[Venue]string{
		Aerodrome:          "Aerodrome",
		Arcus:              "Arcus",
		Binance:            "Binance",
		Bluefin:            "Bluefin",
		BTSE:               "BTSE",
		Bybit:              "Bybit",
		Cetus:              "Cetus",
		Coinbase:           "Coinbase",
		DYDX:               "dYdX",
		Hyperliquid:        "Hyperliquid",
		Lighter:            "Lighter",
		LighterRobinhood:   "Lighter (Robinhood Chain)",
		Meteora:            "Meteora",
		Momentum:           "Momentum",
		OKX:                "OKX",
		Raydium:            "Raydium",
		Turbos:             "Turbos",
		UniswapV3:          "Uniswap v3",
		UniswapV4:          "Uniswap v4",
		Venue("new-venue"): "new-venue",
	} {
		if got := venue.DisplayName(); got != want {
			t.Fatalf("venue=%s got=%s want=%s", venue, got, want)
		}
	}
}
