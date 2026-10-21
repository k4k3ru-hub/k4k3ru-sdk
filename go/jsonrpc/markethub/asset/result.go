package asset

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type Result struct {
	AssetType   market.AssetClass   `json:"assetType"`
	Asset       string              `json:"asset"`
	Symbol      string              `json:"symbol"`
	Name        string              `json:"name"`
	MarketTypes []market.MarketType `json:"marketTypes"` // Registered coverage, not price freshness or trading support.
	SentAt      int64               `json:"sentAt"`      // Gateway send time in Unix microseconds.
}

// Validate validates metadata, unique Spot-first coverage and a positive send time.
//
// Version:
//   - 2026-10-21: Added.
func (r Result) Validate() error {
	if err := (Params{AssetType: r.AssetType, Asset: r.Asset}).Validate(); err != nil {
		return fmt.Errorf("failed to validate asset result: %w", err)
	}
	for _, field := range []struct{ name, value string }{{"symbol", r.Symbol}, {"name", r.Name}} {
		if field.value == "" {
			return invalid(field.name, "empty")
		}
		if strings.TrimSpace(field.value) != field.value || strings.IndexFunc(field.value, unicode.IsControl) >= 0 {
			return invalid(field.name, "invalid")
		}
	}
	if r.MarketTypes == nil {
		return invalid("market_types", "null")
	}
	if len(r.MarketTypes) == 0 {
		return invalid("market_types", "empty")
	}
	if len(r.MarketTypes) > 2 {
		return invalid("market_types", "too_long")
	}
	previous := 0
	for _, kind := range r.MarketTypes {
		rank := 0
		switch kind {
		case market.MarketTypeSpot:
			rank = 1
		case market.MarketTypePerpetual:
			rank = 2
		default:
			return invalid("market_types", "invalid")
		}
		if rank <= previous {
			return invalid("market_types", "invalid")
		}
		previous = rank
	}
	if r.SentAt <= 0 {
		return invalid("sent_at", "out_of_range")
	}
	return nil
}
