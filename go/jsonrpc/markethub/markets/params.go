// Package markets defines discovery of registered production markets for an asset.
package markets

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price"
)

const (
	DefaultPage  = 1
	DefaultLimit = 100
	MaxLimit     = 500
)

type Params struct {
	AssetType  market.AssetClass `json:"assetType"`
	Asset      string            `json:"asset"`
	MarketType market.MarketType `json:"marketType"`
	Page       int               `json:"page,omitempty"`  // Go zero omits the field; explicit JSON values start at 1.
	Limit      int               `json:"limit,omitempty"` // Go zero omits the field; explicit JSON values are 1–500.
}

// Normalize fills omitted pagination without changing asset or market selectors.
//
// Version:
//   - 2026-10-20: Added.
func (p Params) Normalize() Params {
	if p.Page == 0 {
		p.Page = DefaultPage
	}
	if p.Limit == 0 {
		p.Limit = DefaultLimit
	}
	return p
}

// Validate validates crypto asset selectors and bounded one-based pagination.
// Go zero page/limit select defaults; negative values remain invalid.
//
// Version:
//   - 2026-10-20: Added.
func (p Params) Validate() error {
	if err := (price.Params{AssetType: p.AssetType, Asset: p.Asset, MarketType: p.MarketType}).Validate(); err != nil {
		return fmt.Errorf("failed to validate markets parameters: %w", err)
	}
	p = p.Normalize()
	if p.Page < 1 {
		return invalid("page", "out_of_range")
	}
	if p.Limit < 1 || p.Limit > MaxLimit {
		return invalid("limit", "out_of_range")
	}
	return nil
}

// UnmarshalJSON decodes strict selectors and positive optional pagination atomically.
// Omitted page/limit use defaults; explicit null, zero and fractional values fail.
//
// Version:
//   - 2026-10-20: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type wire Params
	value := wire{Page: DefaultPage, Limit: DefaultLimit}
	if err := jsonobject.Decode(data, &value, "assetType", "asset", "marketType"); err != nil {
		return fmt.Errorf("failed to decode markets parameters: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to decode markets parameters: %w", err)
	}
	for _, key := range []string{"page", "limit"} {
		if raw, exists := fields[key]; exists && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return invalid(key, "null")
		}
	}
	if value.Page < 1 {
		return invalid("page", "out_of_range")
	}
	if value.Limit < 1 || value.Limit > MaxLimit {
		return invalid("limit", "out_of_range")
	}
	if err := Params(value).Validate(); err != nil {
		return fmt.Errorf("failed to decode markets parameters: %w", err)
	}
	*p = Params(value)
	return nil
}

func invalid(field, state string) error {
	return fmt.Errorf("failed to validate markets: %w: %s=%s", apperror.InvalidParameter(), field, state)
}
