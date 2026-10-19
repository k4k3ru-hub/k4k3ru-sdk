package candles

import (
	"fmt"
	"regexp"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

const (
	DefaultLimit = 300
	MaxLimit     = 300
	MaxRange     = 7 * 24 * time.Hour
)

var assetPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

type Params struct {
	AssetType      market.AssetClass    `json:"assetType"` // Required; only crypto is supported.
	Asset          string               `json:"asset"`
	MarketType     string               `json:"marketType"`       // spot or perp
	Market         *market.MarketTarget `json:"market,omitempty"` // Omitted selects Composite; otherwise resolve one mainnet DEX market.
	From           int64                `json:"from"`             // Inclusive Unix seconds.
	To             int64                `json:"to"`               // Exclusive Unix seconds.
	Interval       string               `json:"interval"`
	Limit          int                  `json:"limit,omitempty"`          // Returned candles; zero selects DefaultLimit.
	IncludeCurrent bool                 `json:"includeCurrent,omitempty"` // Include developing candles within the selected range.
}

// Validate validates bounded, interval-aligned candle selectors and an optional market.
// The server checks the requested end against its clock and resolves market identity.
//
// Version:
//   - 2026-10-19: Accept one market target and optional developing candles.
//   - 2026-10-19: Require the crypto asset type without changing candle selectors.
//   - 2026-10-11: Add the cursor-free contract with at most 300 returned candles.
func (p Params) Validate() error {
	if p.AssetType == market.AssetClassUnknown {
		return fmt.Errorf("failed to validate candle parameters: %w: asset_type=empty", apperror.InvalidParameter())
	}
	if p.AssetType != market.AssetClassCrypto {
		return fmt.Errorf("failed to validate candle parameters: %w: asset_type=invalid", apperror.InvalidParameter())
	}
	if !assetPattern.MatchString(p.Asset) || (p.MarketType != "spot" && p.MarketType != "perp") {
		return fmt.Errorf("failed to validate candle parameters: %w: asset_or_market_type=invalid", apperror.InvalidParameter())
	}
	if p.Market != nil {
		if err := p.Market.Validate(); err != nil {
			return fmt.Errorf("failed to validate candle parameters: %w", err)
		}
	}
	step := int64(p.Step() / time.Second)
	if step == 0 {
		return fmt.Errorf("failed to validate candle parameters: %w: interval=invalid", apperror.InvalidParameter())
	}
	if p.From <= 0 || p.To <= p.From || p.To-p.From > int64(MaxRange/time.Second) || p.From%step != 0 || p.To%step != 0 {
		return fmt.Errorf("failed to validate candle parameters: %w: range=out_of_range", apperror.InvalidParameter())
	}
	if p.Limit < 0 || p.Limit > MaxLimit {
		return fmt.Errorf("failed to validate candle parameters: %w: limit=out_of_range min_value=0 max_value=%d", apperror.InvalidParameter(), MaxLimit)
	}
	return nil
}

// Step returns the selected interval duration, or zero for an unsupported interval.
//
// Version:
//   - 2026-10-11: Added.
func (p Params) Step() time.Duration {
	switch p.Interval {
	case "1m":
		return time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "1h":
		return time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	}
	return 0
}
