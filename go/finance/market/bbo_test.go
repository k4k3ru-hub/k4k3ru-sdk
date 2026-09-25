package market

import (
	"testing"
)

func validBBO() BBO {
	return BBO{
		AssetClass:  AssetClassCrypto,
		MarketType:  MarketTypePerpetual,
		Symbol:      BTCUSDC,
		Venue:       DYDX,
		VenueSymbol: "BTC-USD",
		Bid:         PriceLevel{Price: "118000.25", Quantity: "1.25"},
		Ask:         PriceLevel{Price: "118000.50", Quantity: "0.75"},
		Timestamp:   1786845600000000,
		Sequence:    "42",
	}
}

// TestBBOValidateAcceptsCompleteBBO verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestBBOValidateAcceptsCompleteBBO(t *testing.T) {
	if err := validBBO().Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

// TestBBOValidateAcceptsCrossedMarket verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestBBOValidateAcceptsCrossedMarket(t *testing.T) {
	bbo := validBBO()
	bbo.Bid.Price = "118001"

	if err := bbo.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

// TestBBOValidateRejectsInvalidValues verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestBBOValidateRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*BBO)
	}{
		{name: "asset class", mutate: func(bbo *BBO) { bbo.AssetClass = AssetClassUnknown }},
		{name: "market type", mutate: func(bbo *BBO) { bbo.MarketType = MarketTypeUnknown }},
		{name: "symbol", mutate: func(bbo *BBO) { bbo.Symbol = "" }},
		{name: "venue", mutate: func(bbo *BBO) { bbo.Venue = Unknown }},
		{name: "venue symbol", mutate: func(bbo *BBO) { bbo.VenueSymbol = "" }},
		{name: "bid price", mutate: func(bbo *BBO) { bbo.Bid.Price = "invalid" }},
		{name: "bid quantity", mutate: func(bbo *BBO) { bbo.Bid.Quantity = "0" }},
		{name: "ask price", mutate: func(bbo *BBO) { bbo.Ask.Price = "NaN" }},
		{name: "ask quantity", mutate: func(bbo *BBO) { bbo.Ask.Quantity = "-1" }},
		{name: "timestamp", mutate: func(bbo *BBO) { bbo.Timestamp = 0 }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bbo := validBBO()
			test.mutate(&bbo)
			if err := bbo.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}
