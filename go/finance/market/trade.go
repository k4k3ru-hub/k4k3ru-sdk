package market

import (
	"math/big"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// TradeSide identifies the aggressor side of a trade.
type TradeSide string

const (
	TradeSideBuy  TradeSide = "buy"
	TradeSideSell TradeSide = "sell"
)

// Trade represents one public trade for a venue market. Quantity is expressed
// in units of the canonical symbol's base asset; venue adapters must convert
// contract counts or other venue-specific units before constructing a trade.
type Trade struct {
	AssetClass AssetClass `json:"ac"`
	MarketType MarketType `json:"mt"`

	Symbol      Symbol `json:"s"`
	Venue       Venue  `json:"v"`
	VenueSymbol string `json:"vs"`

	Side     TradeSide `json:"side"`
	Price    string    `json:"p"`
	Quantity string    `json:"q"`
	TradeID  string    `json:"tid,omitempty"`

	Timestamp int64 `json:"ts"`
}

// Validate validates a public trade.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-22: Required a non-empty trade identifier.
//   - 2026-08-16: Added.
func (t Trade) Validate() error {
	if !isValidBBOAssetClass(t.AssetClass) {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w: asset_class=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if !isValidBBOMarketType(t.MarketType) {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w: market_type=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := t.Symbol.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w", err)
	}
	if err := t.Venue.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w", err)
	}
	if strings.TrimSpace(t.VenueSymbol) == "" {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w: venue_symbol=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if len(t.VenueSymbol) > 128 {
		return k4k3ruSDKAppError.Tracef(
			"failed to validate trade: %w: venue_symbol=too_long actual_length=%d max_length=%d",
			k4k3ruSDKAppError.InvalidParameter(),
			len(t.VenueSymbol),
			128,
		)
	}
	if t.Side != TradeSideBuy && t.Side != TradeSideSell {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w: side=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := validateTradeValue("price", t.Price); err != nil {
		return err
	}
	if err := validateTradeValue("quantity", t.Quantity); err != nil {
		return err
	}
	if strings.TrimSpace(t.TradeID) == "" {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w: trade_id=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if t.Timestamp <= 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w: timestamp=out_of_range", k4k3ruSDKAppError.InvalidParameter())
	}

	return nil
}

func validateTradeValue(field, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w: %s=empty", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	if strings.ContainsAny(value, "/eE") {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w: %s=invalid", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	decimal := new(big.Rat)
	if _, ok := decimal.SetString(value); !ok {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w: %s=invalid", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	if decimal.Sign() <= 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate trade: %w: %s=out_of_range", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	return nil
}
