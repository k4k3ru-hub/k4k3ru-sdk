package volume

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
	Definition string            `json:"definition"` // composite
	Currency   string            `json:"currency"`   // USD
	Method     string            `json:"method"`     // sum
	Volumes    []PeriodVolume    `json:"volumes"`
	SentAt     int64             `json:"sentAt"` // Gateway send time in Unix microseconds.
}

type PeriodVolume struct {
	Period             Period         `json:"period"`
	Volume             *string        `json:"volume"`             // Observed USD turnover, including the developing minute; nil when unavailable.
	From               int64          `json:"from"`               // Calculation start rounded down to a UTC minute, in Unix microseconds.
	To                 int64          `json:"to"`                 // Common calculation time in Unix microseconds.
	CoveragePercentage string         `json:"coveragePercentage"` // Fully observed closed minutes / expected closed minutes * 100.
	CoverageStatus     CoverageStatus `json:"coverageStatus"`     // Coverage of all expected markets, not price freshness.
}

// Validate validates selectors, observed amounts, shared calculation time and coverage states.
// Zero is valid only with an available coverage status; the server owns observation evidence.
//
// Version:
//   - 2026-10-19: Added.
func (r Result) Validate() error {
	if err := (Params{AssetType: market.AssetClassCrypto, Asset: r.Asset, MarketType: r.MarketType}).Validate(); err != nil {
		return fmt.Errorf("failed to validate volume response: %w", err)
	}
	if r.Definition != "composite" || r.Currency != "USD" || r.Method != "sum" || r.SentAt <= 0 {
		return fmt.Errorf("failed to validate volume response: metadata=invalid")
	}
	if len(r.Volumes) == 0 {
		return fmt.Errorf("failed to validate volume response: volumes=empty")
	}
	if len(r.Volumes) > maxPeriods {
		return fmt.Errorf("failed to validate volume response: volumes=too_long actual_length=%d max_length=%d", len(r.Volumes), maxPeriods)
	}
	seen := make(map[Period]bool, len(r.Volumes))
	for i, value := range r.Volumes {
		if value.Period.Duration() == 0 || seen[value.Period] {
			return fmt.Errorf("failed to validate volume response: period=invalid index=%d", i)
		}
		seen[value.Period] = true
		if value.To != r.Volumes[0].To {
			return fmt.Errorf("failed to validate volume response: calculation_time=invalid index=%d", i)
		}
		if err := validateVolume(value, r.SentAt); err != nil {
			return fmt.Errorf("failed to validate volume response: %w: index=%d", err, i)
		}
	}
	return nil
}

func validateVolume(v PeriodVolume, sentAt int64) error {
	minute, duration := time.Minute.Microseconds(), v.Period.Duration().Microseconds()
	if v.From <= 0 || v.To <= duration || v.To > sentAt || v.From != (v.To-duration)/minute*minute {
		return fmt.Errorf("failed to validate period volume: calculation_range=invalid")
	}
	coverage, err := parseDecimal(v.CoveragePercentage)
	if err != nil {
		return fmt.Errorf("failed to validate period volume: %w: field=coverage_percentage", err)
	}
	if coverage.Cmp(big.NewRat(100, 1)) > 0 {
		return fmt.Errorf("failed to validate period volume: coverage_percentage=out_of_range")
	}
	if v.CoverageStatus == CoverageStatusUnavailable {
		if v.Volume != nil || coverage.Sign() != 0 {
			return fmt.Errorf("failed to validate period volume: unavailable_values=invalid")
		}
		return nil
	}
	if v.Volume == nil {
		return fmt.Errorf("failed to validate period volume: volume=null")
	}
	if _, err := parseDecimal(*v.Volume); err != nil {
		return fmt.Errorf("failed to validate period volume: %w: field=volume", err)
	}
	switch v.CoverageStatus {
	case CoverageStatusSufficient:
		if coverage.Cmp(big.NewRat(50, 1)) < 0 {
			return fmt.Errorf("failed to validate period volume: sufficient_coverage=out_of_range")
		}
	case CoverageStatusInsufficient:
		if coverage.Cmp(big.NewRat(50, 1)) >= 0 {
			return fmt.Errorf("failed to validate period volume: insufficient_coverage=out_of_range")
		}
	default:
		return fmt.Errorf("failed to validate period volume: coverage_status=invalid")
	}
	return nil
}

func parseDecimal(value string) (*big.Rat, error) {
	if value == "" {
		return nil, fmt.Errorf("failed to parse volume decimal: value=empty")
	}
	if len(value) > 512 {
		return nil, fmt.Errorf("failed to parse volume decimal: value=too_long actual_length=%d max_length=512", len(value))
	}
	if !decimalPattern.MatchString(value) {
		return nil, fmt.Errorf("failed to parse volume decimal: value=invalid")
	}
	number, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, fmt.Errorf("failed to parse volume decimal: value=invalid")
	}
	return number, nil
}
