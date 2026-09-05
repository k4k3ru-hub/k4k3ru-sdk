package spread

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResultUsesCompactFields(t *testing.T) {
	encoded, err := json.Marshal(Result{AssetClass: AssetClassCrypto, Symbol: "BTC/USDC", BaseAsset: "BTC", QuoteAsset: "USDC", Quantity: "0.1", EligibleRoutes: []Route{}})
	if err != nil {
		t.Fatal(err)
	}
	value := string(encoded)
	for _, key := range []string{`"ac"`, `"s"`, `"ba"`, `"qa"`, `"q"`, `"er"`} {
		if !strings.Contains(value, key) {
			t.Fatalf("Marshal() = %s, want %s", value, key)
		}
	}
}
