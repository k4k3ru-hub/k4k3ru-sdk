package price

import (
	"fmt"
	"math/big"
	"regexp"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type Status string

var decimalPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

const (
	StatusLive        Status = "live"
	StatusStale       Status = "stale"
	StatusUnavailable Status = "unavailable"
)

type Result struct {
	Asset      string            `json:"asset"`
	MarketType market.MarketType `json:"marketType"`
	Definition string            `json:"definition"`
	Currency   string            `json:"currency"`
	Method     string            `json:"method"`
	Interval   string            `json:"interval"`
	Price      *string           `json:"price"`
	Status     Status            `json:"status"`
	VenueCount uint32            `json:"venueCount"`
	SentAt     int64             `json:"sentAt"` // Gateway response time in Unix microseconds, not trade time.
}

// Validate validates the response values and their availability relationship.
//
// Version:
//   - 2026-10-18: Validate latest-price weighting with a rolling 60-second volume window.
func (r Result) Validate() error {
	if err := (Params{Asset: r.Asset, MarketType: r.MarketType}).Validate(); err != nil {
		return fmt.Errorf("failed to validate price response: %w", err)
	}
	if r.Definition != "composite" || r.Currency != "USD" || r.Method != "volume_weighted_latest" || r.Interval != "60s" || r.SentAt <= 0 {
		return fmt.Errorf("failed to validate price response: metadata=invalid")
	}
	switch r.Status {
	case StatusUnavailable:
		if r.Price != nil || r.VenueCount != 0 {
			return fmt.Errorf("failed to validate price response: unavailable_price=invalid")
		}
	case StatusLive, StatusStale:
		if r.Price == nil || r.VenueCount == 0 || len(*r.Price) > 512 {
			return fmt.Errorf("failed to validate price response: price_or_venue_count=invalid")
		}
		if !decimalPattern.MatchString(*r.Price) {
			return fmt.Errorf("failed to validate price response: price=invalid")
		}
		value, ok := new(big.Rat).SetString(*r.Price)
		if !ok || value.Sign() <= 0 {
			return fmt.Errorf("failed to validate price response: price=out_of_range")
		}
	default:
		return fmt.Errorf("failed to validate price response: status=invalid")
	}
	return nil
}
