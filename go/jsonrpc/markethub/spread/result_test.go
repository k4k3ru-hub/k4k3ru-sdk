package spread

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestResultUsesCompactFields(t *testing.T) {
	encoded, err := json.Marshal(Result{AssetClass: AssetClassCrypto, Symbol: "BTC/USDC", BaseAsset: "BTC", QuoteAsset: "USDC", Quantity: "0.1", EligibleRoutes: []Route{{Buy: Leg{Venue: "cetus", MarketType: MarketTypeSpot, Side: SideBuy, BaseAsset: "SUI", BaseAssetID: "0x2::sui::SUI", QuoteAsset: "USDC", QuoteAssetID: "0xusdc", PoolID: "0xpool", Chain: ChainSui, Network: NetworkMainnet}}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"bai":"0x2::sui::SUI"`, `"qai":"0xusdc"`, `"pid":"0xpool"`, `"c":"sui"`, `"n":"mainnet"`} {
		if !strings.Contains(string(encoded), field) {
			t.Fatalf("Marshal() = %s, missing %s", encoded, field)
		}
	}
	value := string(encoded)
	for _, key := range []string{`"ac"`, `"s"`, `"ba"`, `"qa"`, `"q"`, `"er"`} {
		if !strings.Contains(value, key) {
			t.Fatalf("Marshal() = %s, want %s", value, key)
		}
	}
}
