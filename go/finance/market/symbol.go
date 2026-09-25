package market

import (
	"fmt"
	"strings"
)

type Symbol string

const (
	BTCUSD   Symbol = "BTC/USD"
	BTCUSDC  Symbol = "BTC/USDC"
	BTCUSDT  Symbol = "BTC/USDT"
	ETHUSD   Symbol = "ETH/USD"
	ETHUSDC  Symbol = "ETH/USDC"
	ETHUSDT  Symbol = "ETH/USDT"
	HYPEUSDC Symbol = "HYPE/USDC"
	PONSUSDG Symbol = "PONS/USDG"
	SOLUSDC  Symbol = "SOL/USDC"
	SOLUSDT  Symbol = "SOL/USDT"
	SUIUSDC  Symbol = "SUI/USDC"
	WBTCUSDT Symbol = "WBTC/USDT"
	WETHUSDC Symbol = "WETH/USDC"
	WETHUSDT Symbol = "WETH/USDT"
)

// String returns the symbol as a string.
//
// Returns:
//   - Symbol string.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-15: Moved to the market package and renamed from Canonical.
func (s Symbol) String() string {
	return string(s)
}

// IsEmpty reports whether the symbol is empty.
//
// Returns:
//   - True when the symbol is empty.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-15: Moved to the market package and renamed from Canonical.
func (s Symbol) IsEmpty() bool {
	return s == ""
}

// IsValid reports whether Validate accepts the symbol.
// It does not check a known-symbol list or market availability.
//
// Returns:
//   - True when Validate returns nil.
//
// Version:
//   - 2026-09-25: Delegate to Validate instead of restricting symbols to constants.
//   - 2026-09-06: Added PONS quoted in USDG.
//   - 2026-08-31: Added SUI quoted in USDC.
//   - 2026-08-30: Added WETH quoted in USDC.
//   - 2026-08-21: Added WBTC and WETH markets quoted in USDT.
//   - 2026-08-17: Added BTC, ETH, and SOL markets quoted in USDT.
//   - 2026-08-15: Moved to the market package and renamed from Canonical.
func (s Symbol) IsValid() bool {
	return s.Validate() == nil
}

// Validate validates a nonempty symbol of at most 16 bytes.
// Pair syntax, normalization and market availability are API-specific concerns.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-15: Moved to the market package and renamed from Canonical.
func (s Symbol) Validate() error {
	if s == "" {
		return fmt.Errorf("failed to validate symbol: symbol=empty")
	}
	if len(s) > 16 {
		return fmt.Errorf("failed to validate symbol: symbol=too_long actual_length=%d max_length=%d", len(s), 16)
	}
	return nil
}

// BuildSymbol builds a canonical market symbol.
//
// Parameters:
//   - baseAsset: Base asset.
//   - quoteAsset: Quote asset.
//
// Returns:
//   - Canonical market symbol.
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-15: Moved to the market package and renamed from Build.
func BuildSymbol(baseAsset, quoteAsset string) (Symbol, error) {
	if baseAsset == "" {
		return "", fmt.Errorf("failed to build symbol: base_asset=empty")
	}
	if quoteAsset == "" {
		return "", fmt.Errorf("failed to build symbol: quote_asset=empty")
	}

	symbol := Symbol(strings.ToUpper(baseAsset) + "/" + strings.ToUpper(quoteAsset))
	if err := symbol.Validate(); err != nil {
		return "", fmt.Errorf("failed to build symbol: %w: base_asset=%q quote_asset=%q", err, baseAsset, quoteAsset)
	}

	return symbol, nil
}
