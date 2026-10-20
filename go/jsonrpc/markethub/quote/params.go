// Package quote defines quantity-specific estimates from retained market state.
package quote

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
)

const MaximumMarkets = 64

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type Params struct {
	AssetType   market.AssetClass     `json:"assetType"`
	MarketType  market.MarketType     `json:"marketType"`
	Symbol      market.Symbol         `json:"symbol"`
	Side        Side                  `json:"side"`
	Quantity    market.Quantity       `json:"quantity"` // Spot: Quote payment for Buy, Base payment for Sell. Perpetual: Base contracts for either side.
	Markets     []market.MarketTarget `json:"markets"`
	FeeAccounts []FeeAccount          `json:"feeAccounts,omitempty"`
}

// Normalize returns independent canonical selectors without defaulting required fields.
//
// Version:
//   - 2026-10-20: Added.
func (p Params) Normalize() Params {
	p.MarketType = p.MarketType.Normalize()
	p.Symbol = market.Symbol(strings.ToUpper(strings.TrimSpace(string(p.Symbol))))
	p.Side = Side(strings.ToLower(strings.TrimSpace(string(p.Side))))
	if p.Markets != nil {
		p.Markets = append([]market.MarketTarget{}, p.Markets...)
		for i := range p.Markets {
			p.Markets[i] = p.Markets[i].Normalize()
		}
	}
	if p.FeeAccounts != nil {
		p.FeeAccounts = append([]FeeAccount{}, p.FeeAccounts...)
		for i := range p.FeeAccounts {
			p.FeeAccounts[i] = p.FeeAccounts[i].Normalize()
		}
	}
	return p
}

// Validate validates crypto selectors, positive quantity and bounded market scopes.
// The server resolves instruments, token precision and supported local estimates.
//
// Version:
//   - 2026-10-20: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if p.AssetType != market.AssetClassCrypto {
		return invalid("asset_type", state(p.AssetType == ""))
	}
	if err := validateSelectors(p.MarketType, p.Symbol, p.Side, p.Quantity); err != nil {
		return fmt.Errorf("failed to validate quote parameters: %w", err)
	}
	if len(p.Markets) == 0 {
		return invalid("markets", "empty")
	}
	if len(p.Markets) > MaximumMarkets {
		return invalid("markets", "too_long")
	}
	seen := make(map[market.MarketTarget]bool, len(p.Markets))
	for i, target := range p.Markets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf("failed to validate quote parameters: %w: market_index=%d", err, i)
		}
		if seen[target] {
			return invalid("duplicate_market", "invalid")
		}
		seen[target] = true
	}
	return validateFeeAccounts(p)
}

// UnmarshalJSON decodes strict required inputs atomically, preserving invalid-parameter errors.
//
// Version:
//   - 2026-10-20: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type wire Params
	var decoded wire
	if err := jsonobject.Decode(data, &decoded, "assetType", "marketType", "symbol", "side", "quantity", "markets"); err != nil {
		return fmt.Errorf("failed to decode quote parameters: %w", err)
	}
	value := Params(decoded).Normalize()
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode quote parameters: %w", err)
	}
	*p = value
	return nil
}

func validateSelectors(typ market.MarketType, symbol market.Symbol, side Side, quantity market.Quantity) error {
	if typ != market.MarketTypeSpot && typ != market.MarketTypePerpetual {
		return invalid("market_type", state(typ == ""))
	}
	if err := symbol.Validate(); err != nil {
		return fmt.Errorf("failed to validate quote symbol: %w: %w", apperror.InvalidParameter(), err)
	}
	parts := strings.Split(string(symbol), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == parts[1] || strings.ContainsAny(string(symbol), ",;*?[]") || strings.IndexFunc(string(symbol), func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return invalid("symbol", "invalid")
	}
	if side != SideBuy && side != SideSell {
		return invalid("side", state(side == ""))
	}
	return positiveQuantity(quantity)
}

func positiveQuantity(q market.Quantity) error {
	if err := q.Validate(); err != nil {
		return fmt.Errorf("failed to validate quote quantity: %w", err)
	}
	if strings.Trim(q.Amount, "0") == "" {
		return invalid("quantity", "out_of_range")
	}
	return nil
}

func state(empty bool) string {
	if empty {
		return "empty"
	}
	return "invalid"
}

func invalid(field, state string) error {
	return fmt.Errorf("failed to validate quote: %w: %s=%s", apperror.InvalidParameter(), field, state)
}
