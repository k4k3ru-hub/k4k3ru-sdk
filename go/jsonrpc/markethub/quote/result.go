package quote

import (
	"fmt"
	"math/big"
	"regexp"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type Status string

const (
	StatusVWAP              Status = "vwap"
	StatusFallbackReference Status = "fallback_reference"
	StatusUnavailable       Status = "unavailable"
)

type Result struct {
	MarketType market.MarketType `json:"marketType"`
	Symbol     market.Symbol     `json:"symbol"`
	Side       Side              `json:"side"`
	Quantity   market.Quantity   `json:"quantity"` // Echo of the requested quantity, in the same units.
	Quotes     []Quote           `json:"quotes"`
	SentAt     int64             `json:"sentAt"` // Gateway send time, Unix microseconds; not state freshness.
}

type Quote struct {
	Market             market.MarketRef `json:"market"`
	Status             Status           `json:"status"`             // Calculation capability, not freshness or execution certainty.
	NetPrice           *string          `json:"netPrice,omitempty"` // Quote per Base, including trading fees and quantity impact.
	NetPayQuantity     *market.Quantity `json:"netPayQuantity,omitempty"`
	NetReceiveQuantity *market.Quantity `json:"netReceiveQuantity,omitempty"`
	Fees               *Fees            `json:"fees,omitempty"`
}

var pricePattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// Validate validates resolved markets and the status-dependent estimate fields.
// Only vwap covers the whole input and qualifies for a best-estimate comparison.
// Perpetual estimates never expose Spot payment or receipt quantities.
//
// Version:
//   - 2026-10-20: Added.
func (r Result) Validate() error {
	if err := validateSelectors(r.MarketType, r.Symbol, r.Side, r.Quantity); err != nil {
		return fmt.Errorf("failed to validate quote response: %w", err)
	}
	if r.SentAt <= 0 {
		return invalid("sent_at", "out_of_range")
	}
	if len(r.Quotes) == 0 {
		return invalid("quotes", "empty")
	}
	if len(r.Quotes) > MaximumMarkets {
		return invalid("quotes", "too_long")
	}
	seen := make(map[market.MarketRef]bool, len(r.Quotes))
	for i, q := range r.Quotes {
		if err := q.Market.Validate(); err != nil {
			return fmt.Errorf("failed to validate quote response: %w: quote_index=%d", err, i)
		}
		key := q.Market.Normalize()
		if seen[key] {
			return invalid("duplicate_quote", "invalid")
		}
		seen[key] = true
		if err := validateQuote(q, r.MarketType); err != nil {
			return fmt.Errorf("failed to validate quote response: %w: quote_index=%d", err, i)
		}
	}
	return nil
}

func validateQuote(q Quote, typ market.MarketType) error {
	if q.Status == StatusUnavailable {
		if q.NetPrice != nil || q.NetPayQuantity != nil || q.NetReceiveQuantity != nil || q.Fees != nil {
			return invalid("unavailable_values", "invalid")
		}
		return nil
	}
	if q.Status != StatusVWAP && q.Status != StatusFallbackReference {
		return invalid("status", "invalid")
	}
	if q.NetPrice == nil {
		return invalid("net_price", "null")
	}
	if len(*q.NetPrice) > 512 {
		return invalid("net_price", "too_long")
	}
	if !pricePattern.MatchString(*q.NetPrice) {
		return invalid("net_price", state(*q.NetPrice == ""))
	}
	price, ok := new(big.Rat).SetString(*q.NetPrice)
	if !ok || price.Sign() <= 0 {
		return invalid("net_price", "out_of_range")
	}
	if q.Status == StatusFallbackReference {
		if q.NetPayQuantity != nil || q.NetReceiveQuantity != nil || q.Fees != nil {
			return invalid("reference_quantities", "invalid")
		}
		return nil
	}
	if typ == market.MarketTypePerpetual {
		if q.NetPayQuantity != nil || q.NetReceiveQuantity != nil {
			return invalid("perpetual_quantities", "invalid")
		}
	} else {
		if q.NetPayQuantity == nil || q.NetReceiveQuantity == nil {
			return invalid("spot_quantities", "null")
		}
		for _, quantity := range []*market.Quantity{q.NetPayQuantity, q.NetReceiveQuantity} {
			if err := positiveQuantity(*quantity); err != nil {
				return fmt.Errorf("failed to validate spot quote: %w", err)
			}
		}
	}
	return validateFees(q.Fees)
}
