package openinterest

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type Status string
type Valuation string

const (
	StatusLive        Status = "live"
	StatusStale       Status = "stale"
	StatusUnavailable Status = "unavailable"

	ValuationMark          Valuation = "mark"
	ValuationIndex         Valuation = "index"
	ValuationOracle        Valuation = "oracle"
	ValuationVenueReported Valuation = "venue_reported"
)

type Result struct {
	Symbol    market.Symbol `json:"symbol"`
	Interests []Interest    `json:"interests"`
	SentAt    int64         `json:"sentAt"` // Gateway send time, Unix microseconds; not freshness.
}

type Interest struct {
	Market           market.MarketRef `json:"market"`
	Status           Status           `json:"status"`
	Quantity         *market.Quantity `json:"quantity,omitempty"`      // Base-asset equivalent, not raw contracts.
	NotionalValue    *string          `json:"notionalValue,omitempty"` // Decimal units of NotionalCurrency.
	NotionalCurrency string           `json:"notionalCurrency,omitempty"`
	Valuation        Valuation        `json:"valuation,omitempty"`
}

var notionalPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// Validate validates normalized quantities, valuations and unavailable-field omission.
// Zero quantity and zero notional are valid observations.
//
// Version:
//   - 2026-10-20: Added.
func (r Result) Validate() error {
	if err := validateSymbol(r.Symbol); err != nil {
		return fmt.Errorf("failed to validate open interest response: %w", err)
	}
	if r.SentAt <= 0 {
		return invalid("sent_at", "out_of_range")
	}
	if len(r.Interests) == 0 {
		return invalid("interests", "empty")
	}
	if len(r.Interests) > MaximumMarkets {
		return invalid("interests", "too_long")
	}
	seen := make(map[market.MarketRef]bool, len(r.Interests))
	for i, interest := range r.Interests {
		if err := interest.Market.Validate(); err != nil {
			return fmt.Errorf("failed to validate open interest response: %w: interest_index=%d", err, i)
		}
		key := interest.Market.Normalize()
		if seen[key] {
			return invalid("duplicate_interest", "invalid")
		}
		seen[key] = true
		if err := validateInterest(interest); err != nil {
			return fmt.Errorf("failed to validate open interest response: %w: interest_index=%d", err, i)
		}
	}
	return nil
}

func validateInterest(interest Interest) error {
	switch interest.Status {
	case StatusUnavailable:
		if interest.Quantity != nil || interest.NotionalValue != nil || interest.NotionalCurrency != "" || interest.Valuation != "" {
			return invalid("unavailable_values", "invalid")
		}
		return nil
	case StatusLive, StatusStale:
	default:
		return invalid("status", "invalid")
	}
	if interest.Quantity == nil {
		return invalid("quantity", "null")
	}
	if err := interest.Quantity.Validate(); err != nil {
		return fmt.Errorf("failed to validate open interest quantity: %w", err)
	}
	if interest.NotionalValue == nil {
		return invalid("notional_value", "null")
	}
	if len(*interest.NotionalValue) > 512 {
		return invalid("notional_value", "too_long")
	}
	if !notionalPattern.MatchString(*interest.NotionalValue) {
		return invalid("notional_value", state(*interest.NotionalValue == ""))
	}
	if len(interest.NotionalCurrency) > 64 {
		return invalid("notional_currency", "too_long")
	}
	if interest.NotionalCurrency == "" || strings.IndexFunc(interest.NotionalCurrency, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0 {
		return invalid("notional_currency", state(interest.NotionalCurrency == ""))
	}
	switch interest.Valuation {
	case ValuationMark, ValuationIndex, ValuationOracle, ValuationVenueReported:
		return nil
	default:
		return invalid("valuation", state(interest.Valuation == ""))
	}
}
