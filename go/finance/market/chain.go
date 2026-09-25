package market

import (
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// Chain identifies the blockchain that hosts a market source.
type Chain string

const (
	ChainUnknown   Chain = ""
	ChainNone      Chain = "none"
	ChainEthereum  Chain = "ethereum"
	ChainBase      Chain = "base"
	ChainRobinhood Chain = "robinhood"
	ChainBNB       Chain = "bnb"
	ChainSolana    Chain = "solana"
	ChainSui       Chain = "sui"
)

// Normalize normalizes a chain identifier.
//
// Returns:
//   - Normalized chain.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-19: Added.
func (c Chain) Normalize() Chain {
	return Chain(strings.ToLower(strings.TrimSpace(string(c))))
}

// IsValid reports whether the chain is supported.
//
// Returns:
//   - True when the chain is supported.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-29: Supported Solana and Sui.
//   - 2026-09-06: Supported Robinhood Chain.
//   - 2026-08-19: Added.
func (c Chain) IsValid() bool {
	switch c.Normalize() {
	case ChainNone, ChainEthereum, ChainBase, ChainRobinhood, ChainBNB, ChainSolana, ChainSui:
		return true
	default:
		return false
	}
}

// Validate validates a chain identifier.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-06: Accept Robinhood Chain through chain validation.
//   - 2026-08-19: Added.
func (c Chain) Validate() error {
	c = c.Normalize()
	if c == ChainUnknown {
		return k4k3ruSDKAppError.Tracef("failed to validate chain: %w: chain=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if !c.IsValid() {
		return k4k3ruSDKAppError.Tracef("failed to validate chain: %w: chain=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	return nil
}

// MarketSourceKey identifies one venue and chain market-data source.
type MarketSourceKey struct {
	Venue Venue `json:"venue"`
	Chain Chain `json:"chain"`
}

// Normalize normalizes a market source key.
//
// Returns:
//   - Normalized market source key.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-19: Added.
func (k MarketSourceKey) Normalize() MarketSourceKey {
	k.Venue = Venue(strings.ToLower(strings.TrimSpace(string(k.Venue))))
	k.Chain = k.Chain.Normalize()
	return k
}

// Validate validates a market source key.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-19: Added.
func (k MarketSourceKey) Validate() error {
	k = k.Normalize()
	if err := k.Venue.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate market source key: %w", err)
	}
	if err := k.Chain.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate market source key: %w", err)
	}
	chainScopedVenue := k.Venue == Aerodrome || k.Venue == Bluefin || k.Venue == Cetus || k.Venue == Meteora || k.Venue == Momentum || k.Venue == Raydium || k.Venue == Turbos || k.Venue == UniswapV3 || k.Venue == UniswapV4
	if chainScopedVenue && k.Chain == ChainNone {
		return k4k3ruSDKAppError.Tracef("failed to validate market source key: %w: chain=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if !chainScopedVenue && k.Chain != ChainNone {
		return k4k3ruSDKAppError.Tracef("failed to validate market source key: %w: chain=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	return nil
}
