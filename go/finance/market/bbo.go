package market

import (
	"math/big"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// PriceLevel represents a price and base-asset quantity at one side of a market.
// Venue adapters must convert contract counts or other venue-specific units
// before constructing a price level.
type PriceLevel struct {
	Price    string `json:"p"`
	Quantity string `json:"q"`
}

// BBO represents the best bid and offer for a venue market.
type BBO struct {
	AssetClass AssetClass `json:"ac"`
	MarketType MarketType `json:"mt"`

	Symbol      Symbol `json:"s"`
	Venue       Venue  `json:"v"`
	VenueSymbol string `json:"vs"`

	Bid PriceLevel `json:"bid"`
	Ask PriceLevel `json:"ask"`

	Timestamp int64  `json:"ts"`
	Sequence  string `json:"seq,omitempty"`
}

// Validate validates the best bid and offer.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-16: Added.
func (b BBO) Validate() error {
	if !isValidBBOAssetClass(b.AssetClass) {
		return k4k3ruSDKAppError.Tracef(
			"failed to validate bbo: %w: asset_class=invalid",
			k4k3ruSDKAppError.InvalidParameter(),
		)
	}
	if !isValidBBOMarketType(b.MarketType) {
		return k4k3ruSDKAppError.Tracef(
			"failed to validate bbo: %w: market_type=invalid",
			k4k3ruSDKAppError.InvalidParameter(),
		)
	}
	if err := b.Symbol.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate bbo: %w", err)
	}
	if err := b.Venue.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate bbo: %w", err)
	}
	if strings.TrimSpace(b.VenueSymbol) == "" {
		return k4k3ruSDKAppError.Tracef(
			"failed to validate bbo: %w: venue_symbol=empty",
			k4k3ruSDKAppError.InvalidParameter(),
		)
	}
	if len(b.VenueSymbol) > 128 {
		return k4k3ruSDKAppError.Tracef(
			"failed to validate bbo: %w: venue_symbol=too_long actual_length=%d max_length=%d",
			k4k3ruSDKAppError.InvalidParameter(),
			len(b.VenueSymbol),
			128,
		)
	}

	if _, err := validateBBOValue("bid_price", b.Bid.Price); err != nil {
		return err
	}
	if _, err := validateBBOValue("bid_quantity", b.Bid.Quantity); err != nil {
		return err
	}
	if _, err := validateBBOValue("ask_price", b.Ask.Price); err != nil {
		return err
	}
	if _, err := validateBBOValue("ask_quantity", b.Ask.Quantity); err != nil {
		return err
	}
	if b.Timestamp <= 0 {
		return k4k3ruSDKAppError.Tracef(
			"failed to validate bbo: %w: timestamp=out_of_range",
			k4k3ruSDKAppError.InvalidParameter(),
		)
	}

	return nil
}

func validateBBOValue(field, value string) (*big.Rat, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, k4k3ruSDKAppError.Tracef(
			"failed to validate bbo: %w: %s=empty",
			k4k3ruSDKAppError.InvalidParameter(),
			field,
		)
	}
	if strings.ContainsAny(value, "/eE") {
		return nil, k4k3ruSDKAppError.Tracef(
			"failed to validate bbo: %w: %s=invalid",
			k4k3ruSDKAppError.InvalidParameter(),
			field,
		)
	}

	decimal := new(big.Rat)
	if _, ok := decimal.SetString(value); !ok {
		return nil, k4k3ruSDKAppError.Tracef(
			"failed to validate bbo: %w: %s=invalid",
			k4k3ruSDKAppError.InvalidParameter(),
			field,
		)
	}
	if decimal.Sign() <= 0 {
		return nil, k4k3ruSDKAppError.Tracef(
			"failed to validate bbo: %w: %s=out_of_range",
			k4k3ruSDKAppError.InvalidParameter(),
			field,
		)
	}

	return decimal, nil
}

func isValidBBOAssetClass(value AssetClass) bool {
	switch value {
	case AssetClassCrypto,
		AssetClassFX,
		AssetClassStock,
		AssetClassIndex,
		AssetClassFund:
		return true
	default:
		return false
	}
}

func isValidBBOMarketType(value MarketType) bool {
	switch value {
	case MarketTypeSpot,
		MarketTypePerpetual,
		MarketTypeFuture,
		MarketTypeOption,
		MarketTypeCFD:
		return true
	default:
		return false
	}
}
