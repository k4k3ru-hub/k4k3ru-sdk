package arbitrage

import (
	"sort"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// Normalize canonicalizes an Arbitrage source filter without mutating its slices.
//
// Returns:
//   - Normalized source filter.
//
// Version:
//   - 2026-08-30: Added.
func (f SourceFilter) Normalize() SourceFilter {
	if f.Venues == nil {
		return f
	}
	f.Venues = append([]Venue(nil), f.Venues...)
	for i, venue := range f.Venues {
		f.Venues[i] = Venue(strings.ToLower(strings.TrimSpace(string(venue))))
	}
	sort.Slice(f.Venues, func(i, j int) bool { return f.Venues[i] < f.Venues[j] })
	return f
}

// Validate validates an Arbitrage source filter.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-31: Added Cetus support.
//   - 2026-08-31: Added Bluefin support.
//   - 2026-08-31: Added Meteora and Raydium support.
//   - 2026-08-30: Added Aerodrome support.
//   - 2026-08-30: Added.
func (f SourceFilter) Validate() error {
	f = f.Normalize()
	if len(f.Venues) < 2 {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage source filter: %w: venues=too_short actual_length=%d min_length=2", k4k3ruSDKAppError.InvalidParameter(), len(f.Venues))
	}
	seen := make(map[Venue]struct{}, len(f.Venues))
	for _, venue := range f.Venues {
		if venue != VenueAerodrome && venue != VenueBluefin && venue != VenueCetus && venue != VenueMeteora && venue != VenueRaydium && venue != VenueUniswapV3 && venue != VenueUniswapV4 {
			return k4k3ruSDKAppError.Tracef("failed to validate arbitrage source filter: %w: venue=invalid", k4k3ruSDKAppError.InvalidParameter())
		}
		if _, exists := seen[venue]; exists {
			return k4k3ruSDKAppError.Tracef("failed to validate arbitrage source filter: %w: venue=invalid duplicate=true", k4k3ruSDKAppError.InvalidParameter())
		}
		seen[venue] = struct{}{}
	}
	return nil
}
