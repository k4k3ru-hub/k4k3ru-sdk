// Package openinterest defines current normalized perpetual open interest snapshots.
package openinterest

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
)

const MaximumMarkets = 64

type Params struct {
	AssetType market.AssetClass     `json:"assetType"`
	Symbol    market.Symbol         `json:"symbol"`
	Markets   []market.MarketTarget `json:"markets"`
}

// Normalize returns independent normalized market selectors.
//
// Version:
//   - 2026-10-20: Added.
func (p Params) Normalize() Params {
	p.Symbol = market.Symbol(strings.ToUpper(strings.TrimSpace(string(p.Symbol))))
	if p.Markets != nil {
		p.Markets = append([]market.MarketTarget{}, p.Markets...)
		for i := range p.Markets {
			p.Markets[i] = p.Markets[i].Normalize()
		}
	}
	return p
}

// Validate validates crypto-only open interest scopes without defaulting required fields.
//
// Version:
//   - 2026-10-20: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if p.AssetType != market.AssetClassCrypto {
		return invalid("asset_type", state(p.AssetType == ""))
	}
	if err := validateSymbol(p.Symbol); err != nil {
		return fmt.Errorf("failed to validate open interest parameters: %w", err)
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
			return fmt.Errorf("failed to validate open interest parameters: %w: market_index=%d", err, i)
		}
		if seen[target] {
			return invalid("duplicate_market", "invalid")
		}
		seen[target] = true
	}
	return nil
}

// UnmarshalJSON decodes strict required selectors without partially changing the receiver.
//
// Version:
//   - 2026-10-20: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type wire Params
	var decoded wire
	if err := jsonobject.Decode(data, &decoded, "assetType", "symbol", "markets"); err != nil {
		return fmt.Errorf("failed to decode open interest parameters: %w", err)
	}
	value := Params(decoded).Normalize()
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode open interest parameters: %w", err)
	}
	*p = value
	return nil
}

func validateSymbol(symbol market.Symbol) error {
	if err := symbol.Validate(); err != nil {
		return fmt.Errorf("failed to validate open interest symbol: %w: %w", apperror.InvalidParameter(), err)
	}
	parts := strings.Split(string(symbol), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == parts[1] ||
		strings.ContainsAny(string(symbol), ",;*?[]") ||
		strings.IndexFunc(string(symbol), func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return invalid("symbol", "invalid")
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
	return fmt.Errorf("failed to validate open interest: %w: %s=%s", apperror.InvalidParameter(), field, state)
}
