package bbo

import (
	"sort"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// Normalize canonicalizes a BBO source filter without mutating its slices.
//
// Returns:
//   - Normalized source filter.
//
// Version:
//   - 2026-09-05: Added.
func (f SourceFilter) Normalize() SourceFilter {
	if f.VenueCategories != nil {
		f.VenueCategories = normalizeValues(f.VenueCategories)
	}
	if f.LiquidityModels != nil {
		f.LiquidityModels = normalizeValues(f.LiquidityModels)
	}
	if f.AMMPoolChains != nil {
		f.AMMPoolChains = normalizeValues(f.AMMPoolChains)
	}
	return f
}

// Validate validates a BBO source filter.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-05: Added.
func (f SourceFilter) Validate() error {
	f = f.Normalize()
	if f.VenueCategories != nil && len(f.VenueCategories) == 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate bbo source filter: %w: venue_categories=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if f.LiquidityModels != nil && len(f.LiquidityModels) == 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate bbo source filter: %w: liquidity_models=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if f.AMMPoolChains != nil && len(f.AMMPoolChains) == 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate bbo source filter: %w: amm_pool_chains=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	seenVenueCategories := make(map[VenueCategory]struct{}, len(f.VenueCategories))
	for _, value := range f.VenueCategories {
		if value != VenueCategoryCEX && value != VenueCategoryDEX {
			return k4k3ruSDKAppError.Tracef("failed to validate bbo source filter: %w: venue_category=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		if _, exists := seenVenueCategories[value]; exists {
			return k4k3ruSDKAppError.Tracef("failed to validate bbo source filter: %w: venue_category=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		seenVenueCategories[value] = struct{}{}
	}
	seenLiquidityModels := make(map[LiquidityModel]struct{}, len(f.LiquidityModels))
	for _, value := range f.LiquidityModels {
		if value != LiquidityModelOrderBook && value != LiquidityModelAMM {
			return k4k3ruSDKAppError.Tracef("failed to validate bbo source filter: %w: liquidity_model=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		if _, exists := seenLiquidityModels[value]; exists {
			return k4k3ruSDKAppError.Tracef("failed to validate bbo source filter: %w: liquidity_model=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		seenLiquidityModels[value] = struct{}{}
	}
	if f.AMMPoolChains != nil && (!contains(f.VenueCategories, VenueCategoryDEX) || !contains(f.LiquidityModels, LiquidityModelAMM)) {
		return k4k3ruSDKAppError.Tracef("failed to validate bbo source filter: %w: amm_pool_chains=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	seenChains := make(map[Chain]struct{}, len(f.AMMPoolChains))
	for _, value := range f.AMMPoolChains {
		switch value {
		case ChainEthereum, ChainBase, ChainBNB, ChainSolana, ChainSui:
		default:
			return k4k3ruSDKAppError.Tracef("failed to validate bbo source filter: %w: amm_pool_chain=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		if _, exists := seenChains[value]; exists {
			return k4k3ruSDKAppError.Tracef("failed to validate bbo source filter: %w: amm_pool_chain=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		seenChains[value] = struct{}{}
	}
	return nil
}

func normalizeValues[T ~string](values []T) []T {
	if values == nil {
		return nil
	}
	normalized := make([]T, len(values))
	for index, value := range values {
		normalized[index] = T(strings.ToLower(strings.TrimSpace(string(value))))
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return normalized
}

func contains[T comparable](values []T, target T) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
