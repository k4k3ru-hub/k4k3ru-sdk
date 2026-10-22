package price

import (
	"fmt"
	"math/big"
	"regexp"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type Status string

var decimalPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

const (
	StatusLive        Status = "live"
	StatusStale       Status = "stale"
	StatusUnavailable Status = "unavailable"
)

type Result struct {
	Asset      string            `json:"asset"`
	MarketType market.MarketType `json:"marketType"`
	Definition string            `json:"definition"`
	Currency   string            `json:"currency"`
	Method     string            `json:"method"`
	Interval   string            `json:"interval"`
	Price      *string           `json:"price"`
	Status     Status            `json:"status"`
	VenueCount uint32            `json:"venueCount"`
	SentAt     int64             `json:"sentAt"`            // Gateway response time in Unix microseconds, not trade time.
	Markets    []MarketPrice     `json:"markets,omitempty"` // Requested order; shares the top-level send time.
}

type MarketPrice struct {
	Market   market.MarketRef `json:"market"`
	Symbol   market.Symbol    `json:"symbol"`
	Currency string           `json:"currency"`
	Method   string           `json:"method"`
	Price    *string          `json:"price"`
	Status   Status           `json:"status"`
}

// Validate validates the response values and their availability relationship.
//
// Version:
//   - 2026-10-22: Validate the reference-price cutover contract.
//   - 2026-10-21: Validate independently available market prices and exact identities.
//   - 2026-10-19: Validate the crypto-only response independently of request defaults.
//   - 2026-10-18: Validate latest-price weighting with a rolling 60-second volume window.
func (r Result) Validate() error {
	if err := (Params{AssetType: market.AssetClassCrypto, Asset: r.Asset, MarketType: r.MarketType}).Validate(); err != nil {
		return fmt.Errorf("failed to validate price response: %w", err)
	}
	if r.Definition != "composite" || r.Currency != "USD" || r.Method != "volume_weighted_reference" || r.Interval != "60s" || r.SentAt <= 0 {
		return fmt.Errorf("failed to validate price response: metadata=invalid")
	}
	switch r.Status {
	case StatusUnavailable:
		if r.Price != nil || r.VenueCount != 0 {
			return fmt.Errorf("failed to validate price response: unavailable_price=invalid")
		}
	case StatusLive, StatusStale:
		if r.Price == nil || r.VenueCount == 0 || len(*r.Price) > 512 {
			return fmt.Errorf("failed to validate price response: price_or_venue_count=invalid")
		}
		if !decimalPattern.MatchString(*r.Price) {
			return fmt.Errorf("failed to validate price response: price=invalid")
		}
		value, ok := new(big.Rat).SetString(*r.Price)
		if !ok || value.Sign() <= 0 {
			return fmt.Errorf("failed to validate price response: price=out_of_range")
		}
	default:
		return fmt.Errorf("failed to validate price response: status=invalid")
	}
	if len(r.Markets) > MaximumMarkets {
		return fmt.Errorf("failed to validate price response: markets=too_long")
	}
	seen := make(map[market.MarketRef]bool, len(r.Markets))
	for i, entry := range r.Markets {
		if err := validateMarketRef(entry.Market); err != nil {
			return fmt.Errorf("failed to validate price response: %w: market_index=%d", err, i)
		}
		if entry.Market != entry.Market.Normalize() || seen[entry.Market] {
			return fmt.Errorf("failed to validate price response: market=invalid")
		}
		seen[entry.Market] = true
		if err := entry.Symbol.Validate(); err != nil {
			return fmt.Errorf("failed to validate price response: %w", err)
		}
		expectedMethod := "last_trade"
		if entry.Market.PoolID != "" {
			expectedMethod = "pool_price"
		}
		if entry.Currency != "USD" || entry.Method != expectedMethod {
			return fmt.Errorf("failed to validate price response: market_metadata=invalid")
		}
		if entry.Status == StatusUnavailable {
			if entry.Price != nil {
				return fmt.Errorf("failed to validate price response: unavailable_market_price=invalid")
			}
		} else if entry.Status == StatusLive || entry.Status == StatusStale {
			if entry.Price == nil || len(*entry.Price) > 512 || !decimalPattern.MatchString(*entry.Price) {
				return fmt.Errorf("failed to validate price response: market_price=invalid")
			}
			value, ok := new(big.Rat).SetString(*entry.Price)
			if !ok || value.Sign() <= 0 {
				return fmt.Errorf("failed to validate price response: market_price=out_of_range")
			}
		} else {
			return fmt.Errorf("failed to validate price response: market_status=invalid")
		}
	}
	return nil
}

// ValidateFor verifies response values and the exact requested market order and scope.
//
// Version:
//   - 2026-10-22: Validate the reference-price cutover contract.
//   - 2026-10-21: Added.
func (r Result) ValidateFor(params Params) error {
	if err := params.Validate(); err != nil {
		return fmt.Errorf("failed to validate requested prices: %w", err)
	}
	if err := r.Validate(); err != nil {
		return fmt.Errorf("failed to validate requested prices: %w", err)
	}
	if r.Asset != params.Asset || r.MarketType != params.MarketType || len(r.Markets) != len(params.Markets) {
		return fmt.Errorf("failed to validate requested prices: selectors=invalid")
	}
	for i, ref := range params.Markets {
		if r.Markets[i].Market != ref.Normalize() {
			return fmt.Errorf("failed to validate requested prices: market=invalid market_index=%d", i)
		}
	}
	return nil
}
