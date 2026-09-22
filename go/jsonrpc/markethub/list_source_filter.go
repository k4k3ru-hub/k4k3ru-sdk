package markethub

import (
	"fmt"
	appError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// ListSourceFilter selects catalog sources by liquidity model.
type ListSourceFilter struct {
	// LiquidityModels accepts order-book and amm. Omitted, null, or empty selects all models.
	LiquidityModels []string `json:"liquidityModels,omitempty"`
}

// Validate validates the optional catalog source filter.
//
// Version:
//   - 2026-09-22: Added.
func (f *ListSourceFilter) Validate() error {
	if f == nil || len(f.LiquidityModels) == 0 {
		return nil
	}
	seen := make(map[string]bool)
	for _, model := range f.LiquidityModels {
		if model != "order-book" && model != "amm" {
			return fmt.Errorf("failed to validate list source filter: %w: liquidity_model=invalid", appError.InvalidParameter())
		}
		if seen[model] {
			return fmt.Errorf("failed to validate list source filter: duplicate liquidity model: %w: liquidity_model=%q", appError.InvalidParameter(), model)
		}
		seen[model] = true
	}
	return nil
}

// MatchesLiquidityModel reports whether a model passes the catalog filter.
// Validate the filter before matching.
//
// Version:
//   - 2026-09-22: Added.
func (f *ListSourceFilter) MatchesLiquidityModel(model string) bool {
	if f == nil || len(f.LiquidityModels) == 0 {
		return true
	}
	for _, candidate := range f.LiquidityModels {
		if candidate == model {
			return true
		}
	}
	return false
}
