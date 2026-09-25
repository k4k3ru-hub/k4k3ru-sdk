package market

import (
	"encoding/json"
	"testing"
)

// TestBuildSymbolNormalizesAssets verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestBuildSymbolNormalizesAssets(t *testing.T) {
	symbol, err := BuildSymbol("btc", "usdc")
	if err != nil {
		t.Fatalf("BuildSymbol() error = %v", err)
	}
	if symbol != BTCUSDC {
		t.Fatalf("BuildSymbol() = %q, want %q", symbol, BTCUSDC)
	}
}

// TestSymbolJSONContract verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestSymbolJSONContract(t *testing.T) {
	encoded, err := json.Marshal(BTCUSDC)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(encoded) != `"BTC/USDC"` {
		t.Fatalf("json.Marshal() = %s, want %q", encoded, `"BTC/USDC"`)
	}
}

// TestSymbolIsValidAcceptsUSDTMarkets verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestSymbolIsValidAcceptsUSDTMarkets(t *testing.T) {
	tests := []Symbol{BTCUSDT, ETHUSDT, SOLUSDT, WBTCUSDT, WETHUSDT}
	for _, symbol := range tests {
		if !symbol.IsValid() {
			t.Errorf("Symbol(%q).IsValid() = false, want true", symbol)
		}
	}
}

// TestSymbolIsValidAcceptsWrappedEtherUSDC verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestSymbolIsValidAcceptsWrappedEtherUSDC(t *testing.T) {
	if !WETHUSDC.IsValid() {
		t.Errorf("Symbol(%q).IsValid() = false, want true", WETHUSDC)
	}
}

// TestSymbolIsValidAcceptsSUIUSDC verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestSymbolIsValidAcceptsSUIUSDC(t *testing.T) {
	if !SUIUSDC.IsValid() {
		t.Errorf("Symbol(%q).IsValid() = false, want true", SUIUSDC)
	}
}

// TestBuildSymbolNormalizesWrappedAssets verifies the copied finance behavior.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
func TestBuildSymbolNormalizesWrappedAssets(t *testing.T) {
	tests := []struct {
		baseAsset  string
		quoteAsset string
		want       Symbol
	}{
		{baseAsset: "wbtc", quoteAsset: "usdt", want: WBTCUSDT},
		{baseAsset: "weth", quoteAsset: "usdc", want: WETHUSDC},
		{baseAsset: "weth", quoteAsset: "usdt", want: WETHUSDT},
	}

	for _, test := range tests {
		t.Run(test.baseAsset, func(t *testing.T) {
			symbol, err := BuildSymbol(test.baseAsset, test.quoteAsset)
			if err != nil {
				t.Fatalf("BuildSymbol() error = %v", err)
			}
			if symbol != test.want {
				t.Fatalf("BuildSymbol() = %q, want %q", symbol, test.want)
			}
			if !symbol.IsValid() {
				t.Fatalf("Symbol(%q).IsValid() = false, want true", symbol)
			}
		})
	}
}
