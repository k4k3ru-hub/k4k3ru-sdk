package market

import (
	"testing"
)

func validTrade() Trade {
	return Trade{
		AssetClass:  AssetClassCrypto,
		MarketType:  MarketTypePerpetual,
		Symbol:      BTCUSDC,
		Venue:       Hyperliquid,
		VenueSymbol: "BTC",
		Side:        TradeSideBuy,
		Price:       "118000.25",
		Quantity:    "1.25",
		TradeID:     "42",
		Timestamp:   1786845600000000,
	}
}

// TestTradeValidateAcceptsCompleteTrade verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestTradeValidateAcceptsCompleteTrade(t *testing.T) {
	if err := validTrade().Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

// TestTradeValidateRejectsInvalidValues verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestTradeValidateRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Trade)
	}{
		{name: "asset class", mutate: func(trade *Trade) { trade.AssetClass = AssetClassUnknown }},
		{name: "market type", mutate: func(trade *Trade) { trade.MarketType = MarketTypeUnknown }},
		{name: "symbol", mutate: func(trade *Trade) { trade.Symbol = "" }},
		{name: "venue", mutate: func(trade *Trade) { trade.Venue = Unknown }},
		{name: "venue symbol", mutate: func(trade *Trade) { trade.VenueSymbol = "" }},
		{name: "side", mutate: func(trade *Trade) { trade.Side = "unknown" }},
		{name: "price", mutate: func(trade *Trade) { trade.Price = "NaN" }},
		{name: "quantity", mutate: func(trade *Trade) { trade.Quantity = "0" }},
		{name: "trade id", mutate: func(trade *Trade) { trade.TradeID = "  " }},
		{name: "timestamp", mutate: func(trade *Trade) { trade.Timestamp = 0 }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			trade := validTrade()
			test.mutate(&trade)
			if err := trade.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}
