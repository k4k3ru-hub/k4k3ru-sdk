package market

import "fmt"

type Venue string

const (
	Unknown          Venue = ""
	Arcus            Venue = "arcus"
	Binance          Venue = "binance"
	Hyperliquid      Venue = "hyperliquid"
	DYDX             Venue = "dydx"
	Lighter          Venue = "lighter"
	LighterRobinhood Venue = "lighter-robinhood"
	BTSE             Venue = "btse"
	Bybit            Venue = "bybit"
	Coinbase         Venue = "coinbase"
	OKX              Venue = "okx"
	Aerodrome        Venue = "aerodrome"
	Bluefin          Venue = "bluefin"
	Cetus            Venue = "cetus"
	Turbos           Venue = "turbos"
	Meteora          Venue = "meteora"
	Momentum         Venue = "momentum"
	Raydium          Venue = "raydium"
	UniswapV4        Venue = "uniswap-v4"
	UniswapV3        Venue = "uniswap-v3"
)

// IsValid reports whether the venue is supported.
//
// Returns:
//   - True when the venue is supported.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-07: Supported Arcus.
//   - 2026-09-06: Supported both Lighter deployments.
//   - 2026-09-01: Supported Momentum.
//   - 2026-08-31: Supported Bluefin.
//   - 2026-08-31: Supported Cetus.
//   - 2026-09-01: Supported Turbos.
//   - 2026-08-30: Supported Aerodrome.
//   - 2026-08-24: Supported Coinbase Exchange.
//   - 2026-08-24: Supported OKX.
//   - 2026-08-23: Supported Bybit.
//   - 2026-08-19: Supported Uniswap v4.
//   - 2026-08-15: Moved to the market package and renamed from Name.
//   - 2026-08-16: Supported BTSE.
func (v Venue) IsValid() bool {
	switch v {
	case Arcus, Binance,
		Hyperliquid,
		DYDX,
		Lighter,
		LighterRobinhood,
		BTSE,
		Bybit,
		Coinbase,
		OKX,
		Aerodrome,
		Bluefin,
		Cetus,
		Turbos,
		Meteora,
		Momentum,
		Raydium,
		UniswapV4,
		UniswapV3:
		return true
	default:
		return false
	}
}

// String returns the venue as a string.
//
// Returns:
//   - Venue string.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-15: Moved to the market package and renamed from Name.
func (v Venue) String() string {
	return string(v)
}

// Validate validates the venue.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-15: Moved to the market package, renamed from Name, and fixed error returns.
func (v Venue) Validate() error {
	if v == "" {
		return fmt.Errorf("failed to validate venue: venue=empty")
	}
	if len(v) > 64 {
		return fmt.Errorf("failed to validate venue: venue=too_long actual_length=%d max_length=%d", len(v), 64)
	}
	if !v.IsValid() {
		return fmt.Errorf("failed to validate venue: venue=invalid")
	}
	return nil
}
