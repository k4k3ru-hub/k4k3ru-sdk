package spread

import (
	"sort"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// Normalize canonicalizes a Spread source filter.
//
// Version:
//   - 2026-09-05: Added.
func (f SourceFilter) Normalize() SourceFilter {
	f.VenueCategories = normalizeValues(f.VenueCategories)
	f.LiquidityModels = normalizeValues(f.LiquidityModels)
	f.AMMPoolChains = normalizeValues(f.AMMPoolChains)
	return f
}

// Validate validates a Spread source filter.
//
// Version:
//   - 2026-09-05: Added.
func (f SourceFilter) Validate() error {
	f = f.Normalize()
	if f.VenueCategories != nil && len(f.VenueCategories) == 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate spread source filter: %w: venue_categories=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if f.LiquidityModels != nil && len(f.LiquidityModels) == 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate spread source filter: %w: liquidity_models=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if f.AMMPoolChains != nil && len(f.AMMPoolChains) == 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate spread source filter: %w: amm_pool_chains=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := validateUnique(f.VenueCategories, func(v VenueCategory) bool { return v == VenueCategoryCEX || v == VenueCategoryDEX }); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate spread source filter: %w: venue_category=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := validateUnique(f.LiquidityModels, func(v LiquidityModel) bool { return v == LiquidityModelOrderBook || v == LiquidityModelAMM }); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate spread source filter: %w: liquidity_model=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if f.AMMPoolChains != nil {
		if !contains(f.VenueCategories, VenueCategoryDEX) || !contains(f.LiquidityModels, LiquidityModelAMM) {
			return k4k3ruSDKAppError.Tracef("failed to validate spread source filter: %w: amm_pool_chains=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		if err := validateUnique(f.AMMPoolChains, func(v Chain) bool {
			return v == ChainEthereum || v == ChainBase || v == ChainBNB || v == ChainSolana || v == ChainSui
		}); err != nil {
			return k4k3ruSDKAppError.Tracef("failed to validate spread source filter: %w: amm_pool_chain=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
	}
	return nil
}

func normalizeValues[T ~string](values []T) []T {
	if values == nil {
		return nil
	}
	out := append([]T(nil), values...)
	for i := range out {
		out[i] = T(strings.ToLower(strings.TrimSpace(string(out[i]))))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
func validateUnique[T comparable](values []T, valid func(T) bool) error {
	seen := make(map[T]struct{}, len(values))
	for _, value := range values {
		if !valid(value) {
			return k4k3ruSDKAppError.InvalidParameter()
		}
		if _, ok := seen[value]; ok {
			return k4k3ruSDKAppError.InvalidParameter()
		}
		seen[value] = struct{}{}
	}
	return nil
}
func contains[T comparable](values []T, target T) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
