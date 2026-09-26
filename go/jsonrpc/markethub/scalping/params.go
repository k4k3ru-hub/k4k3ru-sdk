// Package scalping defines single-symbol MarketHub scalping observations.
package scalping

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
)

const (
	DefaultWindowMS uint64 = 60_000
	MaximumWindowMS uint64 = 60_000
	MaximumMarkets         = 64
)

type Params struct {
	MarketType market.MarketType     `json:"marketType"`
	Symbol     market.Symbol         `json:"symbol"`
	WindowMS   uint64                `json:"windowMs"`
	Markets    []market.MarketTarget `json:"markets"`
	Buy        *SideParams           `json:"buy,omitempty"`
	Sell       *SideParams           `json:"sell,omitempty"`
}

// Normalize returns independent canonical parameters without defaulting explicit zeros.
//
// Version:
//   - 2026-09-26: Use independent Buy Quote and Sell Base input quantities.
//   - 2026-09-25: Added.
func (p Params) Normalize() Params {
	p.MarketType = p.MarketType.Normalize()
	p.Symbol = market.Symbol(strings.ToUpper(strings.TrimSpace(string(p.Symbol))))
	p.Buy = normalizeSide(p.Buy)
	p.Sell = normalizeSide(p.Sell)
	if p.Markets != nil {
		p.Markets = append([]market.MarketTarget{}, p.Markets...)
		for i := range p.Markets {
			p.Markets[i] = p.Markets[i].Normalize()
		}
	}
	return p
}

// Validate validates one symbol and bounded observation scopes.
// The server must verify each market's symbol, assets, metadata and data quality.
//
// Version:
//   - 2026-09-26: Use independent Buy Quote and Sell Base input quantities.
//   - 2026-09-25: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if p.MarketType != market.MarketTypeSpot && p.MarketType != market.MarketTypePerpetual {
		return invalid("market_type", "invalid")
	}
	if err := p.Symbol.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping observations: %w: %w", apperror.InvalidParameter(), err)
	}
	parts := strings.Split(string(p.Symbol), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == parts[1] || strings.ContainsAny(string(p.Symbol), ",;*?[]") || strings.IndexFunc(string(p.Symbol), func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return invalid("symbol", "invalid")
	}
	if p.WindowMS == 0 {
		return invalid("window_ms", "empty")
	}
	if p.WindowMS > MaximumWindowMS {
		return invalid("window_ms", "out_of_range")
	}
	if len(p.Markets) == 0 {
		return invalid("markets", "empty")
	}
	if len(p.Markets) > MaximumMarkets {
		return invalid("markets", "too_long")
	}
	for _, side := range []*SideParams{p.Buy, p.Sell} {
		if side != nil {
			if err := side.Validate(); err != nil {
				return fmt.Errorf("failed to validate scalping observations: %w", err)
			}
		}
	}
	seen := make(map[market.MarketTarget]bool, len(p.Markets))
	for i, reference := range p.Markets {
		if err := reference.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping observations: %w: market_index=%d", err, i)
		}
		if seen[reference] {
			return invalid("duplicate_market", "invalid")
		}
		seen[reference] = true
	}
	return nil
}

// UnmarshalJSON decodes strict parameters and defaults only an omitted window.
// Decode failure leaves the receiver unchanged.
//
// Version:
//   - 2026-09-26: Use independent Buy Quote and Sell Base input quantities.
//   - 2026-09-25: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type fields Params
	window := DefaultWindowMS
	decoded := struct {
		fields
		WindowMS *uint64 `json:"windowMs"`
	}{WindowMS: &window}
	if err := jsonobject.Decode(data, &decoded, "marketType", "symbol", "markets"); err != nil {
		return fmt.Errorf("failed to decode scalping observations: %w", err)
	}
	if decoded.WindowMS == nil {
		return invalid("window_ms", "null")
	}
	value := Params(decoded.fields).Normalize()
	value.WindowMS = *decoded.WindowMS
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping observations: %w", err)
	}
	*p = value
	return nil
}

func invalid(field, state string) error {
	return fmt.Errorf("failed to validate scalping observations: %w: %s=%s", apperror.InvalidParameter(), field, state)
}

// SideParams specifies exact input units: Quote for Buy, Base for Sell.
type SideParams struct {
	Quantity *market.Quantity `json:"quantity,omitempty"`
}

func normalizeSide(side *SideParams) *SideParams {
	if side == nil || side.Quantity == nil {
		return nil
	}
	q := *side.Quantity
	return &SideParams{Quantity: &q}
}

// Validate validates an optional positive exact input quantity.
//
// Version:
//   - 2026-09-26: Added.
func (p SideParams) Validate() error {
	if p.Quantity == nil {
		return nil
	}
	if err := p.Quantity.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping side: %w", err)
	}
	if strings.Trim(p.Quantity.Amount, "0") == "" {
		return invalid("quantity", "out_of_range")
	}
	return nil
}

// UnmarshalJSON rejects unknown side fields and validates explicit quantities.
//
// Version:
//   - 2026-09-26: Added.
func (p *SideParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type fields SideParams
	var value fields
	if err := jsonobject.Decode(data, &value); err != nil {
		return fmt.Errorf("failed to decode scalping side: %w", err)
	}
	if err := SideParams(value).Validate(); err != nil {
		return err
	}
	*p = SideParams(value)
	return nil
}
