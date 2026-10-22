package pricerange

import (
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type CoverageStatus string

const (
	CoverageStatusSufficient   CoverageStatus = "sufficient"
	CoverageStatusInsufficient CoverageStatus = "insufficient"
	CoverageStatusUnavailable  CoverageStatus = "unavailable"
)

var decimalPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

type Result struct {
	Asset      string            `json:"asset"`
	MarketType market.MarketType `json:"marketType"`
	Definition string            `json:"definition"`
	Currency   string            `json:"currency"`
	Method     string            `json:"method"`
	Ranges     []Range           `json:"ranges"`
	SentAt     int64             `json:"sentAt"` // Gateway send time in Unix microseconds.
}

type Range struct {
	Period             Period         `json:"period"`
	Low                *string        `json:"low"`                // USD price; nil when unavailable.
	High               *string        `json:"high"`               // Includes the observed developing candle.
	From               int64          `json:"from"`               // Calculation start rounded down to a UTC minute, in Unix microseconds.
	To                 int64          `json:"to"`                 // Common calculation time in Unix microseconds, not send or last-observation time.
	CoveragePercentage string         `json:"coveragePercentage"` // Usable closed minutes / expected closed minutes * 100.
	CoverageStatus     CoverageStatus `json:"coverageStatus"`     // Historical coverage, not price freshness.
}

// Validate validates selectors, decimal prices, shared calculation time and coverage states.
// The server owns candle eligibility and exact count-based coverage calculation.
//
// Version:
//   - 2026-10-22: Validate the reference-price cutover contract.
//   - 2026-10-19: Added.
func (r Result) Validate() error {
	if err := (Params{AssetType: market.AssetClassCrypto, Asset: r.Asset, MarketType: r.MarketType}).Validate(); err != nil {
		return fmt.Errorf("failed to validate price range response: %w", err)
	}
	if r.Definition != "composite" || r.Currency != "USD" || r.Method != "volume_weighted_reference" || r.SentAt <= 0 {
		return fmt.Errorf("failed to validate price range response: metadata=invalid")
	}
	if len(r.Ranges) == 0 {
		return fmt.Errorf("failed to validate price range response: ranges=empty")
	}
	if len(r.Ranges) > maxPeriods {
		return fmt.Errorf("failed to validate price range response: ranges=too_long actual_length=%d max_length=%d", len(r.Ranges), maxPeriods)
	}
	seen := make(map[Period]bool, len(r.Ranges))
	for i, value := range r.Ranges {
		if value.Period.Duration() == 0 || seen[value.Period] {
			return fmt.Errorf("failed to validate price range response: period=invalid index=%d", i)
		}
		seen[value.Period] = true
		if value.To != r.Ranges[0].To {
			return fmt.Errorf("failed to validate price range response: calculation_time=invalid index=%d", i)
		}
		if err := validateRange(value, r.SentAt); err != nil {
			return fmt.Errorf("failed to validate price range response: %w: index=%d", err, i)
		}
	}
	return nil
}

func validateRange(r Range, sentAt int64) error {
	minute := time.Minute.Microseconds()
	duration := r.Period.Duration().Microseconds()
	if r.From <= 0 || r.To <= duration || r.To > sentAt || r.From != (r.To-duration)/minute*minute {
		return fmt.Errorf("failed to validate period range: calculation_range=invalid")
	}
	coverage, err := parseDecimal(r.CoveragePercentage)
	if err != nil {
		return fmt.Errorf("failed to validate period range: %w: field=coverage_percentage", err)
	}
	if coverage.Cmp(big.NewRat(100, 1)) > 0 {
		return fmt.Errorf("failed to validate period range: coverage_percentage=out_of_range")
	}
	if r.CoverageStatus == CoverageStatusUnavailable {
		if r.Low != nil || r.High != nil || coverage.Sign() != 0 {
			return fmt.Errorf("failed to validate period range: unavailable_values=invalid")
		}
		return nil
	}
	if r.Low == nil || r.High == nil {
		return fmt.Errorf("failed to validate period range: price=null")
	}
	low, err := parseDecimal(*r.Low)
	if err != nil {
		return fmt.Errorf("failed to validate period range: %w: field=low", err)
	}
	high, err := parseDecimal(*r.High)
	if err != nil {
		return fmt.Errorf("failed to validate period range: %w: field=high", err)
	}
	if low.Sign() <= 0 || high.Cmp(low) < 0 {
		return fmt.Errorf("failed to validate period range: prices=out_of_range")
	}
	switch r.CoverageStatus {
	case CoverageStatusSufficient:
		if coverage.Cmp(big.NewRat(50, 1)) < 0 {
			return fmt.Errorf("failed to validate period range: sufficient_coverage=out_of_range")
		}
	case CoverageStatusInsufficient:
		if coverage.Cmp(big.NewRat(50, 1)) >= 0 {
			return fmt.Errorf("failed to validate period range: insufficient_coverage=out_of_range")
		}
	default:
		return fmt.Errorf("failed to validate period range: coverage_status=invalid")
	}
	return nil
}

func parseDecimal(value string) (*big.Rat, error) {
	if value == "" {
		return nil, fmt.Errorf("failed to parse range decimal: value=empty")
	}
	if len(value) > 512 {
		return nil, fmt.Errorf("failed to parse range decimal: value=too_long actual_length=%d max_length=512", len(value))
	}
	if !decimalPattern.MatchString(value) {
		return nil, fmt.Errorf("failed to parse range decimal: value=invalid")
	}
	number, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, fmt.Errorf("failed to parse range decimal: value=invalid")
	}
	return number, nil
}
