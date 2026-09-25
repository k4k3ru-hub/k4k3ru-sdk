package market

import (
	"fmt"
	"math"
	"time"
)

type FundingRateKind string

const (
	FundingRateKindCurrentEstimate FundingRateKind = "current_estimate"
	FundingRateKindSettled         FundingRateKind = "settled"
)

// FundingRate represents one normalized perpetual funding-rate observation.
type FundingRate struct {
	AssetClass        AssetClass
	Venue             Venue
	MarketType        MarketType
	Symbol            Symbol
	VenueSymbol       string
	EventTimestamp    time.Time
	ReceivedTimestamp time.Time
	FundingTimestamp  time.Time
	Rate              float64
	Kind              FundingRateKind
	IntervalMinutes   int32
	MarkPrice         *float64
	IndexPrice        *float64
	PremiumRate       *float64
}

// Validate validates a normalized perpetual funding-rate observation.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-26: Added.
func (r FundingRate) Validate() error {
	if r.AssetClass == AssetClassUnknown {
		return fmt.Errorf("failed to validate funding rate: asset_class=invalid")
	}
	if !r.Venue.IsValid() {
		return fmt.Errorf("failed to validate funding rate: venue=invalid")
	}
	if r.MarketType != MarketTypePerpetual {
		return fmt.Errorf("failed to validate funding rate: market_type=invalid")
	}
	if err := r.Symbol.Validate(); err != nil {
		return fmt.Errorf("failed to validate funding rate: %w", err)
	}
	if r.VenueSymbol == "" {
		return fmt.Errorf("failed to validate funding rate: venue_symbol=empty")
	}
	if r.EventTimestamp.IsZero() || r.ReceivedTimestamp.IsZero() || r.FundingTimestamp.IsZero() {
		return fmt.Errorf("failed to validate funding rate: timestamp=empty")
	}
	if math.IsNaN(r.Rate) || math.IsInf(r.Rate, 0) {
		return fmt.Errorf("failed to validate funding rate: rate=invalid")
	}
	if r.Kind != FundingRateKindCurrentEstimate && r.Kind != FundingRateKindSettled {
		return fmt.Errorf("failed to validate funding rate: kind=invalid")
	}
	if r.IntervalMinutes <= 0 {
		return fmt.Errorf("failed to validate funding rate: interval_minutes=out_of_range min_value=1")
	}
	for _, value := range []*float64{r.MarkPrice, r.IndexPrice} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value <= 0) {
			return fmt.Errorf("failed to validate funding rate: price=out_of_range min_value=0")
		}
	}
	return nil
}
