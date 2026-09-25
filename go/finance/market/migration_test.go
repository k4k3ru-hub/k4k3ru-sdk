package market

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestFinanceSymbolValidationPreservesDynamicSymbols verifies unlisted symbols and validation bounds.
//
// Version:
//   - 2026-09-25: Require IsValid and Validate to accept the same dynamic symbols.
func TestFinanceSymbolValidationPreservesDynamicSymbols(t *testing.T) {
	symbol, err := BuildSymbol("new", "usdc")
	if err != nil || symbol != "NEW/USDC" || !symbol.IsValid() || symbol.Validate() != nil {
		t.Fatalf("dynamic symbol contract changed: %q %v", symbol, err)
	}
	// Validate historically checks length, not pair syntax; API validators may
	// impose a stricter pair contract without restricting dynamic symbol listings.
	for _, test := range []struct {
		name   string
		symbol Symbol
		valid  bool
	}{
		{name: "known pair", symbol: SUIUSDC, valid: true},
		{name: "unlisted pair", symbol: "NEW/USDC", valid: true},
		{name: "non-pair", symbol: "NEW", valid: true},
		{name: "maximum length", symbol: Symbol(strings.Repeat("A", 16)), valid: true},
		{name: "empty"},
		{name: "too long", symbol: Symbol(strings.Repeat("A", 17))},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.symbol.IsValid(); got != test.valid {
				t.Fatalf("IsValid() = %t, want %t", got, test.valid)
			}
			if err := test.symbol.Validate(); (err == nil) != test.valid {
				t.Fatalf("Validate() = %v, want valid=%t", err, test.valid)
			}
		})
	}
}

// TestFinancePerpetualWireContract verifies all copied models use the canonical market type.
//
// Version:
//   - 2026-09-25: Added.
func TestFinancePerpetualWireContract(t *testing.T) {
	trade := validTrade()
	if trade.MarketType != MarketTypePerpetual || trade.Validate() != nil {
		t.Fatal("perpetual trade rejected")
	}
	encoded, err := json.Marshal(trade)
	if err != nil || !strings.Contains(string(encoded), `"mt":"perpetual"`) {
		t.Fatalf("wire market type is not perpetual: %s %v", encoded, err)
	}
	if err := MarketType("perp").Validate(); err == nil {
		t.Fatal("legacy spelling accepted by new finance contract")
	}
	trade.MarketType = "perp"
	if err := trade.Validate(); err == nil {
		t.Fatal("legacy trade spelling accepted")
	}
	if got := MarketType(" PERPETUAL ").Normalize(); got != MarketTypePerpetual {
		t.Fatalf("incorrect normalization: %q", got)
	}
}
