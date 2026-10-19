package volume

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
	AssetType  market.AssetClass `json:"assetType"` // Required; only crypto is supported.
	Asset      string            `json:"asset"`
	MarketType market.MarketType `json:"marketType"`
	Periods    []Period          `json:"periods,omitempty"` // Empty selects all six periods.
}

// Duration returns the fixed elapsed duration, or zero for an unsupported period.
// Days are 24 hours, independently of calendar and timezone boundaries.
//
// Version:
//   - 2026-10-19: Added.
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

// Normalize copies the period selection and fills omitted periods without defaulting selectors.
// Explicit selections keep their order, and invalid values remain invalid.
//
// Version:
//   - 2026-10-19: Added.
func (p Params) Normalize() Params {
	if len(p.Periods) == 0 {
		p.Periods = []Period{Period5m, Period15m, Period1h, Period4h, Period24h, Period7d}
	} else {
		p.Periods = slices.Clone(p.Periods)
	}
	return p
}

// Validate validates crypto selectors and at most six distinct supported periods.
//
// Version:
//   - 2026-10-19: Added.
func (p Params) Validate() error {
	if err := (price.Params{AssetType: p.AssetType, Asset: p.Asset, MarketType: p.MarketType}).Validate(); err != nil {
		return fmt.Errorf("failed to validate volume parameters: %w", err)
	}
	if len(p.Periods) > maxPeriods {
		return fmt.Errorf("failed to validate volume parameters: %w: periods=too_long actual_length=%d max_length=%d", apperror.InvalidParameter(), len(p.Periods), maxPeriods)
	}
	seen := make(map[Period]bool, len(p.Periods))
	for i, period := range p.Periods {
		if period.Duration() == 0 {
			return fmt.Errorf("failed to validate volume parameters: %w: period=invalid index=%d", apperror.InvalidParameter(), i)
		}
		if seen[period] {
			return fmt.Errorf("failed to validate volume parameters: duplicate period: %w: periods=invalid index=%d", apperror.InvalidParameter(), i)
		}
		seen[period] = true
	}
	return nil
}
