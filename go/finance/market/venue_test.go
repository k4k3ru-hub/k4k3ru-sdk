package market

import "testing"

// TestVenueValidate verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestVenueValidate(t *testing.T) {
	tests := []struct {
		name    string
		venue   Venue
		wantErr bool
	}{
		{name: "binance", venue: Binance},
		{name: "hyperliquid", venue: Hyperliquid},
		{name: "dydx", venue: DYDX},
		{name: "btse", venue: BTSE},
		{name: "bybit", venue: Bybit},
		{name: "okx", venue: OKX},
		{name: "coinbase", venue: Coinbase},
		{name: "uniswap v4", venue: UniswapV4},
		{name: "aerodrome", venue: Aerodrome},
		{name: "bluefin", venue: Bluefin},
		{name: "cetus", venue: Cetus},
		{name: "turbos", venue: Turbos},
		{name: "momentum", venue: Momentum},
		{name: "meteora", venue: Meteora},
		{name: "raydium", venue: Raydium},
		{name: "empty", venue: Unknown, wantErr: true},
		{name: "unsupported", venue: Venue("unsupported"), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.venue.Validate()
			if (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %t", err, test.wantErr)
			}
		})
	}
}
