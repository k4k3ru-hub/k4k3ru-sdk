package market

import "testing"

// TestMarketSourceKeyValidate verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestMarketSourceKeyValidate(t *testing.T) {
	tests := []struct {
		name    string
		source  MarketSourceKey
		wantErr bool
	}{
		{name: "cex without chain", source: MarketSourceKey{Venue: Binance, Chain: ChainNone}},
		{name: "uniswap ethereum", source: MarketSourceKey{Venue: UniswapV4, Chain: ChainEthereum}},
		{name: "uniswap base", source: MarketSourceKey{Venue: UniswapV4, Chain: ChainBase}},
		{name: "uniswap bnb", source: MarketSourceKey{Venue: UniswapV4, Chain: ChainBNB}},
		{name: "uniswap v3 ethereum", source: MarketSourceKey{Venue: UniswapV3, Chain: ChainEthereum}},
		{name: "aerodrome base", source: MarketSourceKey{Venue: Aerodrome, Chain: ChainBase}},
		{name: "bluefin sui", source: MarketSourceKey{Venue: Bluefin, Chain: ChainSui}},
		{name: "turbos sui", source: MarketSourceKey{Venue: Turbos, Chain: ChainSui}},
		{name: "momentum sui", source: MarketSourceKey{Venue: Momentum, Chain: ChainSui}},
		{name: "meteora solana", source: MarketSourceKey{Venue: Meteora, Chain: ChainSolana}},
		{name: "raydium solana", source: MarketSourceKey{Venue: Raydium, Chain: ChainSolana}},
		{name: "unknown chain", source: MarketSourceKey{Venue: Binance, Chain: ChainUnknown}, wantErr: true},
		{name: "cex with chain", source: MarketSourceKey{Venue: Binance, Chain: ChainBase}, wantErr: true},
		{name: "uniswap without chain", source: MarketSourceKey{Venue: UniswapV4, Chain: ChainNone}, wantErr: true},
		{name: "uniswap v3 without chain", source: MarketSourceKey{Venue: UniswapV3, Chain: ChainNone}, wantErr: true},
		{name: "aerodrome without chain", source: MarketSourceKey{Venue: Aerodrome, Chain: ChainNone}, wantErr: true},
		{name: "bluefin without chain", source: MarketSourceKey{Venue: Bluefin, Chain: ChainNone}, wantErr: true},
		{name: "turbos without chain", source: MarketSourceKey{Venue: Turbos, Chain: ChainNone}, wantErr: true},
		{name: "momentum without chain", source: MarketSourceKey{Venue: Momentum, Chain: ChainNone}, wantErr: true},
		{name: "raydium without chain", source: MarketSourceKey{Venue: Raydium, Chain: ChainNone}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.source.Validate()
			if test.wantErr && err == nil {
				t.Fatal("Validate() error = nil")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

// TestChainValidateAcceptsSolanaAndSui verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestChainValidateAcceptsSolanaAndSui(t *testing.T) {
	for _, chain := range []Chain{ChainSolana, ChainSui} {
		if err := chain.Validate(); err != nil {
			t.Fatalf("Chain(%q).Validate() error = %v", chain, err)
		}
	}
}
