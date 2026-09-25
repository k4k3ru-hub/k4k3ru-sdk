package orderbook

import (
	"sync"
	"sync/atomic"
	"time"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	sdkMarket "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

// UpdateResult describes one accepted publication.
type UpdateResult struct {
	BookChanged bool
	BBOChanged  bool
	Version     uint64
}

// Store maintains a mutable working book and atomically publishes immutable snapshots.
type Store struct {
	params Params

	mu      sync.Mutex
	working *Book
	version uint64

	published atomic.Pointer[Snapshot]
}

// Params returns the immutable identity configured for the store.
//
// Returns:
//   - Order book identity.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (s *Store) Params() Params {
	if s == nil {
		return Params{}
	}
	return s.params
}

// Clone returns a detached store containing the same working and published state.
//
// Returns:
//   - Detached order book store, or nil when the receiver is nil.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (s *Store) Clone() *Store {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := &Store{params: s.params, working: s.working.Clone(), version: s.version}
	if current := s.published.Load(); current != nil {
		cloned.published.Store(cloneSnapshot(current, 0))
	}
	return cloned
}

// NewStore creates an empty order book store.
//
// Parameters:
//   - params: Immutable market identity.
//
// Returns:
//   - Order book store.
//   - Construction error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func NewStore(params Params) (*Store, error) {
	if err := params.Validate(); err != nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to create order book store: %w", err)
	}
	return &Store{params: params, working: NewBook()}, nil
}

// ReplaceSnapshot replaces the working book and publishes one immutable snapshot.
//
// Parameters:
//   - update: Complete normalized snapshot.
//
// Returns:
//   - Publication result.
//   - Replacement error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (s *Store) ReplaceSnapshot(update Update) (UpdateResult, error) {
	if s == nil {
		return UpdateResult{}, k4k3ruSDKAppError.Tracef("failed to replace order book store snapshot: %w: order_book_store=null", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := update.ValidateSnapshot(); err != nil {
		return UpdateResult{}, k4k3ruSDKAppError.Tracef("failed to replace order book store snapshot: %w", err)
	}
	next := NewBook()
	if err := next.Replace(update.Bids, update.Asks); err != nil {
		return UpdateResult{}, k4k3ruSDKAppError.Tracef("failed to replace order book store snapshot: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.publishLocked(next, update), nil
}

// ApplyDelta applies and publishes one incremental absolute-quantity update.
//
// Parameters:
//   - update: Normalized delta.
//
// Returns:
//   - Publication result.
//   - Application error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (s *Store) ApplyDelta(update Update) (UpdateResult, error) {
	return s.ApplyDeltas([]Update{update})
}

// ApplyDeltas applies ordered deltas to the working book and publishes only the final state.
//
// Parameters:
//   - updates: Ordered, normalized deltas from one venue market.
//
// Returns:
//   - Publication result.
//   - Application error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (s *Store) ApplyDeltas(updates []Update) (UpdateResult, error) {
	if s == nil {
		return UpdateResult{}, k4k3ruSDKAppError.Tracef("failed to apply order book store deltas: %w: order_book_store=null", k4k3ruSDKAppError.InvalidParameter())
	}
	if len(updates) == 0 {
		return UpdateResult{}, k4k3ruSDKAppError.Tracef("failed to apply order book store deltas: %w: updates=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	for index, update := range updates {
		if err := update.ValidateDelta(); err != nil {
			return UpdateResult{}, k4k3ruSDKAppError.Tracef("failed to apply order book store deltas: %w: update_index=%d", err, index)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.published.Load() == nil {
		return UpdateResult{}, k4k3ruSDKAppError.Tracef("failed to apply order book store deltas: snapshot not ready: %w", k4k3ruSDKAppError.InvalidParameter())
	}
	next := s.working.Clone()
	if err := next.ApplyDeltas(updates); err != nil {
		return UpdateResult{}, k4k3ruSDKAppError.Tracef("failed to apply order book store deltas: %w", err)
	}
	return s.publishLocked(next, updates[len(updates)-1]), nil
}

// Snapshot returns a detached latest snapshot limited to depth levels per side.
//
// Parameters:
//   - depth: Maximum levels per side, or zero for every published level.
//
// Returns:
//   - Latest snapshot, or nil when unavailable or depth is invalid.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (s *Store) Snapshot(depth uint32) *Snapshot {
	if s == nil {
		return nil
	}
	current := s.published.Load()
	if current == nil {
		return nil
	}
	return cloneSnapshot(current, depth)
}

// SnapshotBBO returns the BBO derived from the latest published order book.
//
// Returns:
//   - Latest BBO.
//   - Snapshot or conversion error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (s *Store) SnapshotBBO() (sdkMarket.BBO, error) {
	snapshot := s.Snapshot(1)
	if snapshot == nil {
		return sdkMarket.BBO{}, k4k3ruSDKAppError.Tracef("failed to snapshot order book bbo: %w: order_book_snapshot=null", k4k3ruSDKAppError.NotFound())
	}
	bbo, err := snapshot.BBO()
	if err != nil {
		return sdkMarket.BBO{}, k4k3ruSDKAppError.Tracef("failed to snapshot order book bbo: %w", err)
	}
	return bbo, nil
}

// Reset clears the working and published order books after synchronization is lost.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (s *Store) Reset() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.working = NewBook()
	s.published.Store(nil)
}

func (s *Store) publishLocked(next *Book, update Update) UpdateResult {
	previous := s.published.Load()
	bids, asks := next.Levels(0)
	s.version++
	snapshot := &Snapshot{
		AssetClass: s.params.AssetClass, MarketType: s.params.MarketType,
		Symbol: s.params.Symbol, Venue: s.params.Venue, VenueSymbol: s.params.VenueSymbol, Kind: s.params.Kind,
		Bids: bids, Asks: asks, VenueSequence: update.VenueSequence, Version: s.version,
		EventTimestamp: update.EventTimestamp.UTC(), ReceivedTimestamp: update.ReceivedTimestamp.UTC(),
		PublishedTimestamp: time.Now().UTC(), Synchronized: true,
	}
	result := UpdateResult{BookChanged: !sameLevels(previous, snapshot), BBOChanged: !sameTop(previous, snapshot), Version: snapshot.Version}
	s.working = next
	s.published.Store(snapshot)
	return result
}

func cloneSnapshot(value *Snapshot, depth uint32) *Snapshot {
	cloned := *value
	cloned.Bids = cloneLevels(value.Bids, depth)
	cloned.Asks = cloneLevels(value.Asks, depth)
	return &cloned
}

func cloneLevels(levels []sdkMarket.PriceLevel, depth uint32) []sdkMarket.PriceLevel {
	length := len(levels)
	if depth > 0 && uint64(length) > uint64(depth) {
		length = int(depth)
	}
	cloned := make([]sdkMarket.PriceLevel, length)
	copy(cloned, levels[:length])
	return cloned
}

func sameLevels(left, right *Snapshot) bool {
	if left == nil || right == nil || len(left.Bids) != len(right.Bids) || len(left.Asks) != len(right.Asks) {
		return false
	}
	for index := range left.Bids {
		if left.Bids[index] != right.Bids[index] {
			return false
		}
	}
	for index := range left.Asks {
		if left.Asks[index] != right.Asks[index] {
			return false
		}
	}
	return true
}

func sameTop(left, right *Snapshot) bool {
	return left != nil && right != nil && len(left.Bids) > 0 && len(right.Bids) > 0 && len(left.Asks) > 0 && len(right.Asks) > 0 && left.Bids[0] == right.Bids[0] && left.Asks[0] == right.Asks[0]
}
