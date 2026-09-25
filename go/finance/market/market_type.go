package market

import (
	"fmt"
	"strings"
)

type MarketType string

const (
	MarketTypeUnknown   MarketType = ""
	MarketTypeSpot      MarketType = "spot"
	MarketTypePerpetual MarketType = "perpetual"
	MarketTypeFuture    MarketType = "future"
	MarketTypeOption    MarketType = "option"
	MarketTypeCFD       MarketType = "cfd"
)

// Normalize normalizes market type spelling without accepting legacy aliases.
//
// Version:
//   - 2026-09-25: Added.
func (t MarketType) Normalize() MarketType {
	return MarketType(strings.ToLower(strings.TrimSpace(string(t))))
}

// Validate validates a canonical financial market type.
// API-specific market support must be checked separately.
//
// Version:
//   - 2026-09-25: Copied to the SDK with perpetual as the canonical spelling.
func (t MarketType) Validate() error {
	if t == "" {
		return fmt.Errorf("failed to validate market type: market_type=empty")
	}
	if len(t) > 16 {
		return fmt.Errorf("failed to validate market type: market_type=too_long actual_length=%d max_length=16", len(t))
	}
	switch t {
	case MarketTypeSpot, MarketTypePerpetual, MarketTypeFuture, MarketTypeOption, MarketTypeCFD:
		return nil
	default:
		return fmt.Errorf("failed to validate market type: market_type=invalid")
	}
}
