package orderbook

import (
	"testing"

	sdkMarket "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

// TestBookReplaceSortsLevelsAndCanonicalizesPriceKeys verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestBookReplaceSortsLevelsAndCanonicalizesPriceKeys(t *testing.T) {
	book := NewBook()
	if err := book.Replace(
		[]sdkMarket.PriceLevel{{Price: "99", Quantity: "2"}, {Price: "100.0", Quantity: "1"}, {Price: "100.00", Quantity: "3"}},
		[]sdkMarket.PriceLevel{{Price: "102", Quantity: "4"}, {Price: "101", Quantity: "5"}},
	); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	bids, asks := book.Levels(0)
	if len(bids) != 2 || bids[0].Price != "100.00" || bids[0].Quantity != "3" || bids[1].Price != "99" {
		t.Fatalf("Levels() bids = %+v", bids)
	}
	if len(asks) != 2 || asks[0].Price != "101" || asks[1].Price != "102" {
		t.Fatalf("Levels() asks = %+v", asks)
	}
}

// TestBookApplyDeltasUsesEveryDeltaAndCommitsAtomically verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestBookApplyDeltasUsesEveryDeltaAndCommitsAtomically(t *testing.T) {
	book := NewBook()
	if err := book.Replace(
		[]sdkMarket.PriceLevel{{Price: "100", Quantity: "1"}, {Price: "99", Quantity: "2"}},
		[]sdkMarket.PriceLevel{{Price: "101", Quantity: "3"}, {Price: "102", Quantity: "4"}},
	); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	updates := []Update{
		{Bids: []sdkMarket.PriceLevel{{Price: "100", Quantity: "0"}}},
		{Bids: []sdkMarket.PriceLevel{{Price: "99", Quantity: "5"}}, Asks: []sdkMarket.PriceLevel{{Price: "101", Quantity: "0"}}},
		{Asks: []sdkMarket.PriceLevel{{Price: "103", Quantity: "6"}}},
	}
	for index := range updates {
		updates[index].EventTimestamp = testTime()
		updates[index].ReceivedTimestamp = testTime()
	}
	if err := book.ApplyDeltas(updates); err != nil {
		t.Fatalf("ApplyDeltas() error = %v", err)
	}
	bids, asks := book.Levels(0)
	if len(bids) != 1 || bids[0] != (sdkMarket.PriceLevel{Price: "99", Quantity: "5"}) {
		t.Fatalf("Levels() bids = %+v", bids)
	}
	if len(asks) != 2 || asks[0].Price != "102" || asks[1].Price != "103" {
		t.Fatalf("Levels() asks = %+v", asks)
	}

	if err := book.ApplyDelta([]sdkMarket.PriceLevel{{Price: "104", Quantity: "1"}}, nil); err == nil {
		t.Fatal("ApplyDelta(crossed) error = nil")
	}
	bids, _ = book.Levels(0)
	if len(bids) != 1 || bids[0].Price != "99" {
		t.Fatalf("failed delta changed bids = %+v", bids)
	}
}

// TestBookLevelsLimitsEachSide verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestBookLevelsLimitsEachSide(t *testing.T) {
	book := NewBook()
	if err := book.Replace(
		[]sdkMarket.PriceLevel{{Price: "100", Quantity: "1"}, {Price: "99", Quantity: "2"}},
		[]sdkMarket.PriceLevel{{Price: "101", Quantity: "3"}, {Price: "102", Quantity: "4"}},
	); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}
	bids, asks := book.Levels(1)
	if len(bids) != 1 || len(asks) != 1 || bids[0].Price != "100" || asks[0].Price != "101" {
		t.Fatalf("Levels(1) = bids %+v, asks %+v", bids, asks)
	}
}
