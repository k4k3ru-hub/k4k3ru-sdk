package asset

import (
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type ListedAsset struct {
	Asset       string              `json:"asset"`
	Symbol      string              `json:"symbol"`
	Name        string              `json:"name"`
	MarketTypes []market.MarketType `json:"marketTypes"`
}

type ListResult struct {
	AssetType market.AssetClass `json:"assetType"`
	Assets    []ListedAsset     `json:"assets"`
	Page      int               `json:"page"`
	Limit     int               `json:"limit"`
	Total     int               `json:"total"`
	SentAt    int64             `json:"sentAt"`
}

// Validate validates metadata, symbol/slug ordering, unique identities and page bounds.
//
// Version:
//   - 2026-10-21: Added.
func (r ListResult) Validate() error {
	if err := (ListParams{AssetType: r.AssetType, Page: r.Page, Limit: r.Limit}).Validate(); err != nil {
		return fmt.Errorf("failed to validate asset list result: %w", err)
	}
	if r.Page < 1 || r.Limit < 1 || r.Total < 0 {
		return invalid("pagination", "out_of_range")
	}
	if r.SentAt <= 0 {
		return invalid("sent_at", "out_of_range")
	}
	if r.Assets == nil {
		return invalid("assets", "null")
	}
	expected := 0
	// Check the quotient first so distant pages cannot overflow multiplication.
	if r.Total > 0 && r.Page-1 <= (r.Total-1)/r.Limit {
		expected = min(r.Limit, r.Total-(r.Page-1)*r.Limit)
	}
	if len(r.Assets) != expected {
		return invalid("page_size", "invalid")
	}
	seen := make(map[string]bool, len(r.Assets))
	for i, entry := range r.Assets {
		check := Result{AssetType: r.AssetType, Asset: entry.Asset, Symbol: entry.Symbol, Name: entry.Name, MarketTypes: entry.MarketTypes, SentAt: r.SentAt}
		if err := check.Validate(); err != nil {
			return fmt.Errorf("failed to validate asset list result: %w: asset_index=%d", err, i)
		}
		if seen[entry.Asset] {
			return invalid("duplicate_asset", "invalid")
		}
		seen[entry.Asset] = true
		if i > 0 {
			previous := r.Assets[i-1]
			if previous.Symbol > entry.Symbol || previous.Symbol == entry.Symbol && previous.Asset >= entry.Asset {
				return invalid("asset_order", "invalid")
			}
		}
	}
	return nil
}

// ValidateFor validates a page and its requested asset type, pagination and filters.
//
// Version:
//   - 2026-10-21: Added.
func (r ListResult) ValidateFor(params ListParams) error {
	if err := params.Validate(); err != nil {
		return fmt.Errorf("failed to validate asset list response: %w", err)
	}
	if err := r.Validate(); err != nil {
		return fmt.Errorf("failed to validate asset list response: %w", err)
	}
	p := params.Normalize()
	if r.AssetType != p.AssetType || r.Page != p.Page || r.Limit != p.Limit {
		return invalid("response_scope", "invalid")
	}
	for _, entry := range r.Assets {
		if !p.Matches(entry) {
			return invalid("response_filter", "invalid")
		}
	}
	return nil
}
