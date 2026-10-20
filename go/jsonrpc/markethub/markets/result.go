package markets

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	onchain "github.com/k4k3ru-hub/onchain/go/core"
)

type Result struct {
	Asset      string            `json:"asset"`
	MarketType market.MarketType `json:"marketType"`
	Markets    []Market          `json:"markets"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	Total      int               `json:"total"`  // All matching markets before pagination.
	SentAt     int64             `json:"sentAt"` // Gateway send time in Unix microseconds, not source freshness.
}

type Market struct {
	Market            market.MarketRef      `json:"market"`
	Symbol            market.Symbol         `json:"symbol"`
	LiquidityModel    market.LiquidityModel `json:"liquidityModel"`
	BaseAssetID       string                `json:"baseAssetId"`
	QuoteAssetID      string                `json:"quoteAssetId"`
	CompositeEligible bool                  `json:"compositeEligible"` // Eligibility, not contribution to the current price.
	CandlesSupported  bool                  `json:"candlesSupported"`  // API support, not proof of retained or fresh candles.
}

// Validate validates production identities, stable ordering and complete page bounds.
// Token identity and asset membership are resolved by the server's catalog, not symbols.
//
// Version:
//   - 2026-10-20: Added.
func (r Result) Validate() error {
	if err := (Params{AssetType: market.AssetClassCrypto, Asset: r.Asset, MarketType: r.MarketType, Page: r.Page, Limit: r.Limit}).Validate(); err != nil {
		return fmt.Errorf("failed to validate markets response: %w", err)
	}
	if r.Page < 1 || r.Limit < 1 || r.Total < 0 {
		return invalid("pagination", "out_of_range")
	}
	if r.SentAt <= 0 {
		return invalid("sent_at", "out_of_range")
	}
	if r.Markets == nil {
		return invalid("markets", "null")
	}
	// Divide before multiplying so an arbitrarily distant page cannot overflow.
	expected := 0
	if r.Total > 0 && r.Page-1 <= (r.Total-1)/r.Limit {
		expected = min(r.Limit, r.Total-(r.Page-1)*r.Limit)
	}
	if len(r.Markets) != expected {
		return invalid("page_size", "invalid")
	}
	for i, entry := range r.Markets {
		if err := validateMarket(entry); err != nil {
			return fmt.Errorf("failed to validate markets response: %w: market_index=%d", err, i)
		}
		if i > 0 && !referenceBefore(r.Markets[i-1].Market, entry.Market) {
			return invalid("market_order_or_duplicate", "invalid")
		}
	}
	return nil
}

func validateMarket(entry Market) error {
	ref := entry.Market
	if err := ref.Validate(); err != nil {
		return fmt.Errorf("failed to validate listed market: %w", err)
	}
	if ref != ref.Normalize() {
		return invalid("market", "invalid")
	}
	if ref.Network != onchain.NetworkMainnet {
		return invalid("network", "invalid")
	}
	switch entry.LiquidityModel {
	case market.LiquidityModelAMM:
		if ref.PoolID == "" {
			return invalid("pool_id", "empty")
		}
	case market.LiquidityModelOrderBook:
		if ref.VenueSymbol == "" {
			return invalid("venue_symbol", "empty")
		}
	default:
		return invalid("liquidity_model", "invalid")
	}
	if err := entry.Symbol.Validate(); err != nil {
		return fmt.Errorf("failed to validate listed market: %w", err)
	}
	symbol := string(entry.Symbol)
	parts := strings.Split(symbol, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == parts[1] ||
		symbol != strings.ToUpper(symbol) || strings.ContainsAny(symbol, ",;*?[]") ||
		strings.IndexFunc(symbol, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return invalid("symbol", "invalid")
	}
	for _, id := range []struct{ name, value string }{{"base_asset_id", entry.BaseAssetID}, {"quote_asset_id", entry.QuoteAssetID}} {
		if id.value == "" {
			return invalid(id.name, "empty")
		}
		if len(id.value) > 512 {
			return invalid(id.name, "too_long")
		}
		if id.value != strings.TrimSpace(id.value) || strings.IndexFunc(id.value, unicode.IsControl) >= 0 {
			return invalid(id.name, "invalid")
		}
	}
	return nil
}

func referenceBefore(a, b market.MarketRef) bool {
	left := [...]string{string(a.Venue), string(a.Network), string(a.Chain), a.PoolID, a.VenueSymbol}
	right := [...]string{string(b.Venue), string(b.Network), string(b.Chain), b.PoolID, b.VenueSymbol}
	for i, value := range left {
		if value != right[i] {
			return value < right[i]
		}
	}
	return false
}
