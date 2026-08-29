package aggregation

import (
	"sort"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// Normalize canonicalizes an aggregation source filter without mutating its slices.
//
// Returns:
//   - Normalized source filter.
//
// Version:
//   - 2026-08-29: Added.
func (f AggregationSourceFilter) Normalize() AggregationSourceFilter {
	if f.VenueCategories != nil {
		f.VenueCategories = normalizeStrings(f.VenueCategories)
	}
	if f.LiquidityModels != nil {
		f.LiquidityModels = normalizeStrings(f.LiquidityModels)
	}
	if f.AMMPoolChains != nil {
		f.AMMPoolChains = normalizeStrings(f.AMMPoolChains)
	}
	return f
}

// Validate validates an aggregation source filter.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-29: Added.
func (f AggregationSourceFilter) Validate() error {
	f = f.Normalize()
	if f.VenueCategories != nil && len(f.VenueCategories) == 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate aggregation source filter: %w: venue_categories=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if f.LiquidityModels != nil && len(f.LiquidityModels) == 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate aggregation source filter: %w: liquidity_models=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if f.AMMPoolChains != nil && len(f.AMMPoolChains) == 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate aggregation source filter: %w: amm_pool_chains=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := validateVenueCategories(f.VenueCategories); err != nil {
		return err
	}
	if err := validateLiquidityModels(f.LiquidityModels); err != nil {
		return err
	}
	if f.AMMPoolChains != nil {
		if !contains(f.VenueCategories, VenueCategoryDEX) || !contains(f.LiquidityModels, LiquidityModelAMM) {
			return k4k3ruSDKAppError.Tracef("failed to validate aggregation source filter: %w: amm_pool_chains=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		seen := make(map[Chain]struct{}, len(f.AMMPoolChains))
		for _, chain := range f.AMMPoolChains {
			if !isValidAMMPoolChain(chain) {
				return k4k3ruSDKAppError.Tracef("failed to validate aggregation source filter: %w: amm_pool_chain=invalid", k4k3ruSDKAppError.InvalidParameter())
			}
			if _, ok := seen[chain]; ok {
				return k4k3ruSDKAppError.Tracef("failed to validate aggregation source filter: %w: amm_pool_chain=invalid", k4k3ruSDKAppError.InvalidParameter())
			}
			seen[chain] = struct{}{}
		}
	}
	return nil
}

func normalizeStrings[T ~string](values []T) []T {
	normalized := make([]T, len(values))
	for i, value := range values {
		normalized[i] = T(strings.ToLower(strings.TrimSpace(string(value))))
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	return normalized
}

func validateVenueCategories(values []VenueCategory) error {
	seen := make(map[VenueCategory]struct{}, len(values))
	for _, value := range values {
		if value != VenueCategoryCEX && value != VenueCategoryDEX {
			return k4k3ruSDKAppError.Tracef("failed to validate aggregation source filter: %w: venue_category=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		if _, ok := seen[value]; ok {
			return k4k3ruSDKAppError.Tracef("failed to validate aggregation source filter: %w: venue_category=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateLiquidityModels(values []LiquidityModel) error {
	seen := make(map[LiquidityModel]struct{}, len(values))
	for _, value := range values {
		if value != LiquidityModelOrderBook && value != LiquidityModelAMM {
			return k4k3ruSDKAppError.Tracef("failed to validate aggregation source filter: %w: liquidity_model=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		if _, ok := seen[value]; ok {
			return k4k3ruSDKAppError.Tracef("failed to validate aggregation source filter: %w: liquidity_model=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		seen[value] = struct{}{}
	}
	return nil
}

func isValidAMMPoolChain(chain Chain) bool {
	switch chain {
	case ChainEthereum, ChainBase, ChainBNB, ChainSolana, ChainSui:
		return true
	default:
		return false
	}
}

func contains[T comparable](values []T, target T) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
