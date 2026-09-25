package orderbook

import (
	"strings"
	"time"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	sdkMarket "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

// Kind identifies whether an order book is venue-reported or synthetic.
type Kind string

const (
	KindVenue     Kind = "venue"
	KindSynthetic Kind = "synthetic"
)

// Params identifies one order book store.
type Params struct {
	AssetClass  sdkMarket.AssetClass
	MarketType  sdkMarket.MarketType
	Symbol      sdkMarket.Symbol
	Venue       sdkMarket.Venue
	VenueSymbol string
	Kind        Kind
}

// Validate validates order book store parameters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (p Params) Validate() error {
	switch p.AssetClass {
	case sdkMarket.AssetClassCrypto,
		sdkMarket.AssetClassFX,
		sdkMarket.AssetClassStock,
		sdkMarket.AssetClassIndex,
		sdkMarket.AssetClassFund:
	default:
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w: asset_class=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	switch p.MarketType {
	case sdkMarket.MarketTypeSpot,
		sdkMarket.MarketTypePerpetual,
		sdkMarket.MarketTypeFuture,
		sdkMarket.MarketTypeOption,
		sdkMarket.MarketTypeCFD:
	default:
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w: market_type=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := p.Symbol.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w", err)
	}
	if err := p.Venue.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w", err)
	}
	if strings.TrimSpace(p.VenueSymbol) == "" {
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w: venue_symbol=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if len(p.VenueSymbol) > 128 {
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w: venue_symbol=too_long actual_length=%d max_length=%d", k4k3ruSDKAppError.InvalidParameter(), len(p.VenueSymbol), 128)
	}
	if p.Kind != KindVenue && p.Kind != KindSynthetic {
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w: kind=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	return nil
}

// Update contains one normalized snapshot or delta payload. A zero quantity
// deletes a price level when the update is applied as a delta.
type Update struct {
	Bids              []sdkMarket.PriceLevel
	Asks              []sdkMarket.PriceLevel
	VenueSequence     string
	EventTimestamp    time.Time
	ReceivedTimestamp time.Time
}

// ValidateSnapshot validates a complete order book snapshot update.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (u Update) ValidateSnapshot() error {
	return u.validate(false)
}

// ValidateDelta validates an incremental order book update.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (u Update) ValidateDelta() error {
	return u.validate(true)
}

func (u Update) validate(allowDelete bool) error {
	if u.EventTimestamp.IsZero() || u.ReceivedTimestamp.IsZero() {
		return k4k3ruSDKAppError.Tracef("failed to validate order book update: %w: timestamp=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if len(u.Bids) == 0 && len(u.Asks) == 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate order book update: %w: levels=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	for _, sideLevels := range []struct {
		side   string
		levels []sdkMarket.PriceLevel
	}{{side: "bid", levels: u.Bids}, {side: "ask", levels: u.Asks}} {
		side := sideLevels.side
		levels := sideLevels.levels
		for index, level := range levels {
			if _, _, err := parseLevel(level, allowDelete); err != nil {
				return k4k3ruSDKAppError.Tracef("failed to validate order book update: %w: side=%q level_index=%d", err, side, index)
			}
		}
	}
	return nil
}

// Snapshot is one immutable, published view of an order book.
type Snapshot struct {
	ConfirmedAt        time.Time `json:"-"`
	AssetClass         sdkMarket.AssetClass
	MarketType         sdkMarket.MarketType
	Symbol             sdkMarket.Symbol
	Venue              sdkMarket.Venue
	VenueSymbol        string
	Kind               Kind
	Bids               []sdkMarket.PriceLevel
	Asks               []sdkMarket.PriceLevel
	VenueSequence      string
	Version            uint64
	EventTimestamp     time.Time
	ReceivedTimestamp  time.Time
	PublishedTimestamp time.Time
	Synchronized       bool
}

// BBO converts the top levels of the snapshot to a BBO.
//
// Returns:
//   - Best bid and offer.
//   - Conversion error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-04: Added.
func (s Snapshot) BBO() (sdkMarket.BBO, error) {
	if len(s.Bids) == 0 || len(s.Asks) == 0 {
		return sdkMarket.BBO{}, k4k3ruSDKAppError.Tracef("failed to convert order book snapshot to bbo: %w: top_levels=empty", k4k3ruSDKAppError.NotFound())
	}
	bbo := sdkMarket.BBO{
		AssetClass: s.AssetClass,
		MarketType: s.MarketType,
		Symbol:     s.Symbol, Venue: s.Venue, VenueSymbol: s.VenueSymbol,
		Bid: s.Bids[0], Ask: s.Asks[0],
		Timestamp: s.EventTimestamp.UnixMicro(), Sequence: s.VenueSequence,
	}
	if err := bbo.Validate(); err != nil {
		return sdkMarket.BBO{}, k4k3ruSDKAppError.Tracef("failed to convert order book snapshot to bbo: %w", err)
	}
	return bbo, nil
}

func validateSide(side string) error {
	if side != "bid" && side != "ask" {
		return k4k3ruSDKAppError.Tracef("failed to validate order book side: %w: side=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	return nil
}

// FreshnessTime returns receipt time or the latest verified unchanged-state time.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-09: Added.
func (s *Snapshot) FreshnessTime() time.Time {
	if s == nil {
		return time.Time{}
	}
	if s.ConfirmedAt.After(s.ReceivedTimestamp) {
		return s.ConfirmedAt
	}
	return s.ReceivedTimestamp
}
