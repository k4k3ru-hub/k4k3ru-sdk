package funding

import (
	"fmt"
	"regexp"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type Status string
type Kind string

const (
	StatusLive          Status = "live"
	StatusStale         Status = "stale"
	StatusUnavailable   Status = "unavailable"
	KindCurrentEstimate Kind   = "current_estimate"
)

type Result struct {
	Symbol market.Symbol `json:"symbol"`
	Rates  []Rate        `json:"rates"`
	SentAt int64         `json:"sentAt"` // Gateway send time, Unix microseconds; not freshness.
}

type Rate struct {
	Market          market.MarketRef `json:"market"`
	Status          Status           `json:"status"`
	Kind            Kind             `json:"kind,omitempty"`
	Rate            *string          `json:"rate,omitempty"` // Signed ratio, not percent. Positive means longs pay shorts.
	IntervalMinutes *int32           `json:"intervalMinutes,omitempty"`
	FundingAt       *int64           `json:"fundingAt,omitempty"` // Target settlement time, Unix microseconds, when known.
}

var ratePattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

// Validate validates estimated rates, resolved identities and unavailable-field omission.
// Zero and negative rates are valid. Stale records retain their original target time.
//
// Version:
//   - 2026-10-20: Added.
func (r Result) Validate() error {
	if err := validateSymbol(r.Symbol); err != nil {
		return fmt.Errorf("failed to validate funding response: %w", err)
	}
	if r.SentAt <= 0 {
		return invalid("sent_at", "out_of_range")
	}
	if len(r.Rates) == 0 {
		return invalid("rates", "empty")
	}
	if len(r.Rates) > MaximumMarkets {
		return invalid("rates", "too_long")
	}
	seen := make(map[market.MarketRef]bool, len(r.Rates))
	for i, rate := range r.Rates {
		if err := rate.Market.Validate(); err != nil {
			return fmt.Errorf("failed to validate funding response: %w: rate_index=%d", err, i)
		}
		key := rate.Market.Normalize()
		if seen[key] {
			return invalid("duplicate_rate", "invalid")
		}
		seen[key] = true
		if err := validateRate(rate); err != nil {
			return fmt.Errorf("failed to validate funding response: %w: rate_index=%d", err, i)
		}
	}
	return nil
}

func validateRate(rate Rate) error {
	switch rate.Status {
	case StatusUnavailable:
		if rate.Kind != "" || rate.Rate != nil || rate.IntervalMinutes != nil || rate.FundingAt != nil {
			return invalid("unavailable_values", "invalid")
		}
		return nil
	case StatusLive, StatusStale:
	default:
		return invalid("status", "invalid")
	}
	if rate.Kind != KindCurrentEstimate {
		return invalid("kind", "invalid")
	}
	if rate.Rate == nil {
		return invalid("rate", "null")
	}
	if len(*rate.Rate) > 512 {
		return invalid("rate", "too_long")
	}
	if !ratePattern.MatchString(*rate.Rate) {
		return invalid("rate", state(*rate.Rate == ""))
	}
	if rate.IntervalMinutes == nil {
		return invalid("interval_minutes", "null")
	}
	if *rate.IntervalMinutes <= 0 {
		return invalid("interval_minutes", "out_of_range")
	}
	if rate.FundingAt != nil && *rate.FundingAt <= 0 {
		return invalid("funding_at", "out_of_range")
	}
	return nil
}
