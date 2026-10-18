package change

import (
	"fmt"
	"slices"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price"
)

type Period string

const (
	Period5m  Period = "5m"
	Period15m Period = "15m"
	Period1h  Period = "1h"
	Period4h  Period = "4h"
	Period24h Period = "24h"
	Period7d  Period = "7d"

	maxPeriods = 6
)

type Params struct {
	Asset      string            `json:"asset"`
	MarketType market.MarketType `json:"marketType"`
	Periods    []Period          `json:"periods,omitempty"` // Empty selects all six periods.
}

// Duration returns the fixed elapsed duration, or zero for an unsupported period.
// Days are 24 hours; periods are not calendar or timezone boundaries.
//
// Version:
//   - 2026-10-18: Added.
func (p Period) Duration() time.Duration {
	switch p {
	case Period5m:
		return 5 * time.Minute
	case Period15m:
		return 15 * time.Minute
	case Period1h:
		return time.Hour
	case Period4h:
		return 4 * time.Hour
	case Period24h:
		return 24 * time.Hour
	case Period7d:
		return 7 * 24 * time.Hour
	default:
		return 0
	}
}

// Normalize copies the request and fills omitted periods without changing selectors.
// Explicit periods retain their order and invalid values for Validate to reject.
//
// Version:
//   - 2026-10-18: Added.
func (p Params) Normalize() Params {
	if len(p.Periods) == 0 {
		p.Periods = []Period{Period5m, Period15m, Period1h, Period4h, Period24h, Period7d}
	} else {
		p.Periods = slices.Clone(p.Periods)
	}
	return p
}

// Validate validates canonical selectors and at most six distinct supported periods.
// Omitted or empty periods select the complete default set.
//
// Version:
//   - 2026-10-18: Added.
func (p Params) Validate() error {
	if err := (price.Params{Asset: p.Asset, MarketType: p.MarketType}).Validate(); err != nil {
		return fmt.Errorf("failed to validate price change parameters: %w", err)
	}
	if len(p.Periods) > maxPeriods {
		return fmt.Errorf("failed to validate price change parameters: %w: periods=too_long actual_length=%d max_length=%d", apperror.InvalidParameter(), len(p.Periods), maxPeriods)
	}
	seen := make(map[Period]bool, len(p.Periods))
	for i, period := range p.Periods {
		if period.Duration() == 0 {
			return fmt.Errorf("failed to validate price change parameters: %w: period=invalid index=%d", apperror.InvalidParameter(), i)
		}
		if seen[period] {
			return fmt.Errorf("failed to validate price change parameters: duplicate period: %w: periods=invalid index=%d", apperror.InvalidParameter(), i)
		}
		seen[period] = true
	}
	return nil
}
