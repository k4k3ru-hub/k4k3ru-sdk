package asset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
)

const (
	DefaultListPage  = 1
	DefaultListLimit = 100
	MaxListLimit     = 500
	MaxSearchLength  = 64
)

type ListParams struct {
	AssetType  market.AssetClass `json:"assetType"`
	MarketType market.MarketType `json:"marketType,omitempty"`
	Search     string            `json:"search,omitempty"`
	Page       int               `json:"page,omitempty"`
	Limit      int               `json:"limit,omitempty"`
}

// Normalize fills omitted pagination without changing filters or caller state.
//
// Version:
//   - 2026-10-21: Added.
func (p ListParams) Normalize() ListParams {
	if p.Page == 0 {
		p.Page = DefaultListPage
	}
	if p.Limit == 0 {
		p.Limit = DefaultListLimit
	}
	return p
}

// Validate validates crypto asset discovery, optional filters and pagination.
// Go zero pagination selects defaults; explicit JSON zero is rejected.
//
// Version:
//   - 2026-10-21: Added.
func (p ListParams) Validate() error {
	if p.AssetType == market.AssetClassUnknown {
		return invalid("asset_type", "empty")
	}
	if p.AssetType != market.AssetClassCrypto {
		return invalid("asset_type", "invalid")
	}
	if p.MarketType != "" && p.MarketType != market.MarketTypeSpot && p.MarketType != market.MarketTypePerpetual {
		return invalid("market_type", "invalid")
	}
	if !utf8.ValidString(p.Search) || strings.IndexFunc(p.Search, unicode.IsControl) >= 0 {
		return invalid("search", "invalid")
	}
	if utf8.RuneCountInString(p.Search) > MaxSearchLength {
		return invalid("search", "too_long")
	}
	p = p.Normalize()
	if p.Page < 1 {
		return invalid("page", "out_of_range")
	}
	if p.Limit < 1 || p.Limit > MaxListLimit {
		return invalid("limit", "out_of_range")
	}
	return nil
}

// Matches checks case-insensitive substring and registered-market filters.
// It retains full coverage in the entry; callers validate parameters separately.
//
// Version:
//   - 2026-10-21: Added.
func (p ListParams) Matches(entry ListedAsset) bool {
	if p.MarketType != "" && !slices.Contains(entry.MarketTypes, p.MarketType) {
		return false
	}
	search := strings.ToLower(p.Search)
	return strings.Contains(strings.ToLower(entry.Asset), search) ||
		strings.Contains(strings.ToLower(entry.Symbol), search) ||
		strings.Contains(strings.ToLower(entry.Name), search)
}

// UnmarshalJSON decodes strict filters and positive pagination atomically.
// Omitted marketType selects both types; explicit empty or null types are invalid.
//
// Version:
//   - 2026-10-21: Added.
func (p *ListParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type wire ListParams
	value := wire{Page: DefaultListPage, Limit: DefaultListLimit}
	if err := jsonobject.Decode(data, &value, "assetType"); err != nil {
		return fmt.Errorf("failed to decode asset list parameters: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to decode asset list parameters: %w", err)
	}
	for name, raw := range fields {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			field := strings.ToLower(name)
			if field == "assettype" {
				field = "asset_type"
			} else if field == "markettype" {
				field = "market_type"
			}
			return invalid(field, "null")
		}
		if strings.EqualFold(name, "marketType") && value.MarketType == "" {
			return invalid("market_type", "empty")
		}
	}
	if value.Page < 1 {
		return invalid("page", "out_of_range")
	}
	if value.Limit < 1 {
		return invalid("limit", "out_of_range")
	}
	if err := ListParams(value).Validate(); err != nil {
		return fmt.Errorf("failed to decode asset list parameters: %w", err)
	}
	*p = ListParams(value)
	return nil
}
