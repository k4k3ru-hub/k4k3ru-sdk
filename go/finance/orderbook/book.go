package orderbook

import (
	"math/big"
	"sort"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	sdkMarket "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type bookLevel struct {
	price *big.Rat
	level sdkMarket.PriceLevel
}

// Book stores mutable, normalized L2 price levels. Book is not safe for
// concurrent use; OrderBookStore provides synchronized publication.
type Book struct {
	bids map[string]bookLevel
	asks map[string]bookLevel
}

// NewBook creates an empty mutable L2 order book.
//
// Returns:
//   - Empty order book.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func NewBook() *Book {
	return &Book{bids: make(map[string]bookLevel), asks: make(map[string]bookLevel)}
}

// Clone creates a detached copy of the order book.
//
// Returns:
//   - Detached order book.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (b *Book) Clone() *Book {
	if b == nil {
		return NewBook()
	}
	out := NewBook()
	for key, value := range b.bids {
		out.bids[key] = cloneBookLevel(value)
	}
	for key, value := range b.asks {
		out.asks[key] = cloneBookLevel(value)
	}
	return out
}

// Replace replaces every price level with a complete snapshot.
//
// Parameters:
//   - bids: Complete bid levels.
//   - asks: Complete ask levels.
//
// Returns:
//   - Replacement error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (b *Book) Replace(bids, asks []sdkMarket.PriceLevel) error {
	if b == nil {
		return k4k3ruSDKAppError.Tracef("failed to replace order book: %w: order_book=null", k4k3ruSDKAppError.InvalidParameter())
	}
	next := NewBook()
	if err := next.applySide("bid", bids, false); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to replace order book: %w", err)
	}
	if err := next.applySide("ask", asks, false); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to replace order book: %w", err)
	}
	if err := next.validateUncrossed(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to replace order book: %w", err)
	}
	b.bids = next.bids
	b.asks = next.asks
	return nil
}

// ApplyDelta atomically applies incremental absolute-quantity updates.
//
// Parameters:
//   - bids: Bid updates; zero quantity deletes a level.
//   - asks: Ask updates; zero quantity deletes a level.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (b *Book) ApplyDelta(bids, asks []sdkMarket.PriceLevel) error {
	if b == nil {
		return k4k3ruSDKAppError.Tracef("failed to apply order book delta: %w: order_book=null", k4k3ruSDKAppError.InvalidParameter())
	}
	next := b.Clone()
	if err := next.applySide("bid", bids, true); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to apply order book delta: %w", err)
	}
	if err := next.applySide("ask", asks, true); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to apply order book delta: %w", err)
	}
	if err := next.validateUncrossed(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to apply order book delta: %w", err)
	}
	b.bids = next.bids
	b.asks = next.asks
	return nil
}

// ApplyDeltas atomically applies ordered incremental absolute-quantity updates.
//
// Parameters:
//   - updates: Ordered deltas; zero quantity deletes a level.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (b *Book) ApplyDeltas(updates []Update) error {
	if b == nil {
		return k4k3ruSDKAppError.Tracef("failed to apply order book deltas: %w: order_book=null", k4k3ruSDKAppError.InvalidParameter())
	}
	if len(updates) == 0 {
		return k4k3ruSDKAppError.Tracef("failed to apply order book deltas: %w: updates=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	for index, update := range updates {
		if err := update.ValidateDelta(); err != nil {
			return k4k3ruSDKAppError.Tracef("failed to apply order book deltas: %w: update_index=%d", err, index)
		}
	}
	next := b.Clone()
	for index, update := range updates {
		if err := next.applySide("bid", update.Bids, true); err != nil {
			return k4k3ruSDKAppError.Tracef("failed to apply order book deltas: %w: update_index=%d", err, index)
		}
		if err := next.applySide("ask", update.Asks, true); err != nil {
			return k4k3ruSDKAppError.Tracef("failed to apply order book deltas: %w: update_index=%d", err, index)
		}
	}
	if err := next.validateUncrossed(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to apply order book deltas: %w", err)
	}
	b.bids = next.bids
	b.asks = next.asks
	return nil
}

// Levels returns detached, price-sorted levels, limited per side when depth is positive.
//
// Parameters:
//   - depth: Maximum levels per side, or zero for every level.
//
// Returns:
//   - Bid levels in descending price order.
//   - Ask levels in ascending price order.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (b *Book) Levels(depth uint32) ([]sdkMarket.PriceLevel, []sdkMarket.PriceLevel) {
	if b == nil {
		return nil, nil
	}
	return sortedLevels(b.bids, true, depth), sortedLevels(b.asks, false, depth)
}

func (b *Book) applySide(side string, levels []sdkMarket.PriceLevel, allowDelete bool) error {
	if err := validateSide(side); err != nil {
		return err
	}
	destination := b.bids
	if side == "ask" {
		destination = b.asks
	}
	for index, level := range levels {
		price, quantity, err := parseLevel(level, allowDelete)
		if err != nil {
			return k4k3ruSDKAppError.Tracef("failed to apply order book side: %w: side=%q level_index=%d", err, side, index)
		}
		key := price.RatString()
		if quantity.Sign() == 0 {
			delete(destination, key)
			continue
		}
		destination[key] = bookLevel{price: price, level: sdkMarket.PriceLevel{Price: strings.TrimSpace(level.Price), Quantity: strings.TrimSpace(level.Quantity)}}
	}
	return nil
}

func (b *Book) validateUncrossed() error {
	bids := sortedLevels(b.bids, true, 1)
	asks := sortedLevels(b.asks, false, 1)
	if len(bids) == 0 || len(asks) == 0 {
		return nil
	}
	bestBid, _, _ := parseLevel(bids[0], false)
	bestAsk, _, _ := parseLevel(asks[0], false)
	if bestBid.Cmp(bestAsk) >= 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate order book: crossed book: %w", k4k3ruSDKAppError.InvalidParameter())
	}
	return nil
}

func parseLevel(level sdkMarket.PriceLevel, allowDelete bool) (*big.Rat, *big.Rat, error) {
	price, err := parseDecimal(level.Price, false)
	if err != nil {
		return nil, nil, k4k3ruSDKAppError.Tracef("failed to parse order book level: %w: price=invalid", err)
	}
	quantity, err := parseDecimal(level.Quantity, allowDelete)
	if err != nil {
		return nil, nil, k4k3ruSDKAppError.Tracef("failed to parse order book level: %w: quantity=invalid", err)
	}
	return price, quantity, nil
}

func parseDecimal(value string, allowZero bool) (*big.Rat, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "/eE") {
		return nil, k4k3ruSDKAppError.InvalidParameter()
	}
	parsed := new(big.Rat)
	if _, ok := parsed.SetString(value); !ok || parsed.Sign() < 0 || (!allowZero && parsed.Sign() == 0) {
		return nil, k4k3ruSDKAppError.InvalidParameter()
	}
	return parsed, nil
}

func sortedLevels(values map[string]bookLevel, descending bool, depth uint32) []sdkMarket.PriceLevel {
	entries := make([]bookLevel, 0, len(values))
	for _, value := range values {
		entries = append(entries, value)
	}
	sort.Slice(entries, func(i, j int) bool {
		comparison := entries[i].price.Cmp(entries[j].price)
		if descending {
			return comparison > 0
		}
		return comparison < 0
	})
	if depth > 0 && uint64(len(entries)) > uint64(depth) {
		entries = entries[:int(depth)]
	}
	levels := make([]sdkMarket.PriceLevel, len(entries))
	for index, entry := range entries {
		levels[index] = entry.level
	}
	return levels
}

func cloneBookLevel(value bookLevel) bookLevel {
	return bookLevel{price: new(big.Rat).Set(value.price), level: value.level}
}
