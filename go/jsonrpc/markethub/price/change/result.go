package change

import (
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type Status string

const (
	StatusLive        Status = "live"
	StatusStale       Status = "stale"
	StatusUnavailable Status = "unavailable"
)

var decimalPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)

type Result struct {
	Asset      string            `json:"asset"`
	MarketType market.MarketType `json:"marketType"`
	Definition string            `json:"definition"`
	Currency   string            `json:"currency"`
	Method     string            `json:"method"`
	Price      *string           `json:"price"` // Common current Composite price in USD.
	Changes    []Change          `json:"changes"`
	SentAt     int64             `json:"sentAt"` // Gateway send time in Unix microseconds.
}

type Change struct {
	Period           Period  `json:"period"`
	ChangePercentage *string `json:"changePercentage"` // Signed percentage, not bps; nil when unavailable.
	ReferencePrice   *string `json:"referencePrice"`   // Close of the selected completed one-minute candle.
	ReferenceAt      *int64  `json:"referenceAt"`      // Candle end in Unix microseconds, not its last trade time.
	Status           Status  `json:"status"`
}

// Validate validates metadata, distinct periods, decimal values and availability.
// It checks reference time against the send-time upper bound, not the undisclosed
// calculation time. Candle selection, observation coverage and freshness remain
// server responsibilities; the SDK does not recompute or round the percentage.
//
// Version:
//   - 2026-10-19: Validate the crypto-only response independently of request defaults.
//   - 2026-10-18: Added.
func (r Result) Validate() error {
	if err := (Params{AssetType: market.AssetClassCrypto, Asset: r.Asset, MarketType: r.MarketType}).Validate(); err != nil {
		return fmt.Errorf("failed to validate price change response: %w", err)
	}
	if r.Definition != "composite" || r.Currency != "USD" || r.Method != "volume_weighted_latest" || r.SentAt <= 0 {
		return fmt.Errorf("failed to validate price change response: metadata=invalid")
	}
	if r.Price != nil {
		if err := validateDecimal(*r.Price, "price", true); err != nil {
			return fmt.Errorf("failed to validate price change response: %w", err)
		}
	}
	if len(r.Changes) == 0 {
		return fmt.Errorf("failed to validate price change response: changes=empty")
	}
	if len(r.Changes) > maxPeriods {
		return fmt.Errorf("failed to validate price change response: changes=too_long actual_length=%d max_length=%d", len(r.Changes), maxPeriods)
	}
	seen := make(map[Period]bool, len(r.Changes))
	for i, change := range r.Changes {
		if change.Period.Duration() == 0 || seen[change.Period] {
			return fmt.Errorf("failed to validate price change response: period=invalid index=%d", i)
		}
		seen[change.Period] = true
		if err := validateChange(change, r.Price, r.SentAt); err != nil {
			return fmt.Errorf("failed to validate price change response: %w: index=%d", err, i)
		}
	}
	return nil
}

func validateChange(c Change, current *string, sentAt int64) error {
	if (c.ReferencePrice == nil) != (c.ReferenceAt == nil) {
		return fmt.Errorf("failed to validate period change: reference=invalid")
	}
	if c.ReferencePrice != nil {
		if err := validateDecimal(*c.ReferencePrice, "reference_price", true); err != nil {
			return fmt.Errorf("failed to validate period change: %w", err)
		}
		if *c.ReferenceAt <= 0 || *c.ReferenceAt%time.Minute.Microseconds() != 0 || *c.ReferenceAt > sentAt-c.Period.Duration().Microseconds() {
			return fmt.Errorf("failed to validate period change: reference_at=out_of_range")
		}
	}
	switch c.Status {
	case StatusUnavailable:
		if c.ChangePercentage != nil {
			return fmt.Errorf("failed to validate period change: unavailable_percentage=invalid")
		}
	case StatusLive, StatusStale:
		if current == nil || c.ReferencePrice == nil || c.ChangePercentage == nil {
			return fmt.Errorf("failed to validate period change: required_value=null")
		}
		if err := validateDecimal(*c.ChangePercentage, "change_percentage", false); err != nil {
			return fmt.Errorf("failed to validate period change: %w", err)
		}
	default:
		return fmt.Errorf("failed to validate period change: status=invalid")
	}
	return nil
}

func validateDecimal(value, field string, positive bool) error {
	if len(value) == 0 {
		return fmt.Errorf("failed to validate price change decimal: %s=empty", field)
	}
	if len(value) > 512 {
		return fmt.Errorf("failed to validate price change decimal: %s=too_long actual_length=%d max_length=512", field, len(value))
	}
	if !decimalPattern.MatchString(value) {
		return fmt.Errorf("failed to validate price change decimal: %s=invalid", field)
	}
	number, ok := new(big.Rat).SetString(value)
	if !ok {
		return fmt.Errorf("failed to validate price change decimal: %s=invalid", field)
	}
	if positive && number.Sign() <= 0 || !positive && number.Cmp(big.NewRat(-100, 1)) < 0 {
		return fmt.Errorf("failed to validate price change decimal: %s=out_of_range", field)
	}
	return nil
}
