package orderbook

import (
	"strconv"
	"sync"
	"testing"
	"time"

	sdkMarket "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

// TestStorePublishesSnapshotAndDerivesBBO verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestStorePublishesSnapshotAndDerivesBBO(t *testing.T) {
	store := testStore(t)
	result, err := store.ReplaceSnapshot(testUpdate(
		"10",
		[]sdkMarket.PriceLevel{{Price: "100", Quantity: "1"}, {Price: "99", Quantity: "2"}},
		[]sdkMarket.PriceLevel{{Price: "101", Quantity: "3"}, {Price: "102", Quantity: "4"}},
	))
	if err != nil {
		t.Fatalf("ReplaceSnapshot() error = %v", err)
	}
	if !result.BookChanged || !result.BBOChanged || result.Version != 1 {
		t.Fatalf("ReplaceSnapshot() = %+v", result)
	}
	snapshot := store.Snapshot(1)
	if snapshot == nil || snapshot.Version != 1 || snapshot.VenueSequence != "10" || len(snapshot.Bids) != 1 || len(snapshot.Asks) != 1 || !snapshot.Synchronized {
		t.Fatalf("Snapshot(1) = %+v", snapshot)
	}
	bbo, err := store.SnapshotBBO()
	if err != nil {
		t.Fatalf("SnapshotBBO() error = %v", err)
	}
	if bbo.Bid.Price != "100" || bbo.Ask.Price != "101" || bbo.Sequence != "10" {
		t.Fatalf("SnapshotBBO() = %+v", bbo)
	}
}

// TestStoreApplyDeltasPublishesOnlyFinalState verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestStoreApplyDeltasPublishesOnlyFinalState(t *testing.T) {
	store := testStore(t)
	if _, err := store.ReplaceSnapshot(testUpdate(
		"10",
		[]sdkMarket.PriceLevel{{Price: "100", Quantity: "1"}, {Price: "99", Quantity: "2"}},
		[]sdkMarket.PriceLevel{{Price: "101", Quantity: "3"}, {Price: "102", Quantity: "4"}},
	)); err != nil {
		t.Fatalf("ReplaceSnapshot() error = %v", err)
	}
	result, err := store.ApplyDeltas([]Update{
		testUpdate("11", []sdkMarket.PriceLevel{{Price: "99", Quantity: "5"}}, nil),
		testUpdate("12", nil, []sdkMarket.PriceLevel{{Price: "102", Quantity: "6"}}),
	})
	if err != nil {
		t.Fatalf("ApplyDeltas() error = %v", err)
	}
	if !result.BookChanged || result.BBOChanged || result.Version != 2 {
		t.Fatalf("ApplyDeltas() = %+v", result)
	}
	snapshot := store.Snapshot(0)
	if snapshot.Version != 2 || snapshot.VenueSequence != "12" || snapshot.Bids[1].Quantity != "5" || snapshot.Asks[1].Quantity != "6" {
		t.Fatalf("Snapshot(0) = %+v", snapshot)
	}
}

// TestStoreSnapshotIsDetached verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestStoreSnapshotIsDetached(t *testing.T) {
	store := testStore(t)
	if _, err := store.ReplaceSnapshot(testUpdate("10", []sdkMarket.PriceLevel{{Price: "100", Quantity: "1"}}, []sdkMarket.PriceLevel{{Price: "101", Quantity: "2"}})); err != nil {
		t.Fatalf("ReplaceSnapshot() error = %v", err)
	}
	first := store.Snapshot(0)
	first.Bids[0].Price = "1"
	second := store.Snapshot(0)
	if second.Bids[0].Price != "100" {
		t.Fatalf("Snapshot() shared mutable levels = %+v", second.Bids)
	}
}

// TestStoreCloneIsDetachedAndPreservesVersion verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestStoreCloneIsDetachedAndPreservesVersion(t *testing.T) {
	store := testStore(t)
	if _, err := store.ReplaceSnapshot(testUpdate("10", []sdkMarket.PriceLevel{{Price: "100", Quantity: "1"}}, []sdkMarket.PriceLevel{{Price: "101", Quantity: "2"}})); err != nil {
		t.Fatal(err)
	}
	clone := store.Clone()
	if clone == nil || clone.Params() != store.Params() || clone.Snapshot(0).Version != store.Snapshot(0).Version {
		t.Fatalf("Clone() = %+v", clone)
	}
	if _, err := clone.ApplyDelta(testUpdate("11", []sdkMarket.PriceLevel{{Price: "100", Quantity: "9"}}, nil)); err != nil {
		t.Fatal(err)
	}
	if store.Snapshot(0).Bids[0].Quantity != "1" {
		t.Fatal("Clone() shared mutable state with its source")
	}
}

// TestStoreResetRequiresNewSnapshot verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestStoreResetRequiresNewSnapshot(t *testing.T) {
	store := testStore(t)
	if _, err := store.ReplaceSnapshot(testUpdate("10", []sdkMarket.PriceLevel{{Price: "100", Quantity: "1"}}, []sdkMarket.PriceLevel{{Price: "101", Quantity: "2"}})); err != nil {
		t.Fatalf("ReplaceSnapshot() error = %v", err)
	}
	store.Reset()
	if snapshot := store.Snapshot(0); snapshot != nil {
		t.Fatalf("Snapshot() after Reset = %+v", snapshot)
	}
	if _, err := store.ApplyDelta(testUpdate("11", []sdkMarket.PriceLevel{{Price: "99", Quantity: "1"}}, nil)); err == nil {
		t.Fatal("ApplyDelta() after Reset error = nil")
	}
}

// TestStoreSupportsConcurrentSnapshotsAndSingleWriter verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestStoreSupportsConcurrentSnapshotsAndSingleWriter(t *testing.T) {
	store := testStore(t)
	if _, err := store.ReplaceSnapshot(testUpdate("0", []sdkMarket.PriceLevel{{Price: "100", Quantity: "1"}}, []sdkMarket.PriceLevel{{Price: "101", Quantity: "1"}})); err != nil {
		t.Fatalf("ReplaceSnapshot() error = %v", err)
	}
	var readers sync.WaitGroup
	for range 8 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for range 100 {
				snapshot := store.Snapshot(1)
				if snapshot == nil || len(snapshot.Bids) != 1 || len(snapshot.Asks) != 1 {
					t.Errorf("Snapshot(1) = %+v", snapshot)
					return
				}
			}
		}()
	}
	for index := 1; index <= 100; index++ {
		quantity := strconv.Itoa(index)
		if _, err := store.ApplyDelta(testUpdate(quantity, []sdkMarket.PriceLevel{{Price: "100", Quantity: quantity}}, nil)); err != nil {
			t.Fatalf("ApplyDelta() error = %v", err)
		}
	}
	readers.Wait()
}

func testStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(Params{
		AssetClass: sdkMarket.AssetClassCrypto,
		MarketType: sdkMarket.MarketTypePerpetual,
		Symbol:     sdkMarket.BTCUSDC, Venue: sdkMarket.DYDX, VenueSymbol: "BTC-USD", Kind: KindVenue,
	})
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	return store
}

func testUpdate(sequence string, bids, asks []sdkMarket.PriceLevel) Update {
	now := testTime()
	return Update{Bids: bids, Asks: asks, VenueSequence: sequence, EventTimestamp: now, ReceivedTimestamp: now.Add(time.Millisecond)}
}

func testTime() time.Time {
	return time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)
}
