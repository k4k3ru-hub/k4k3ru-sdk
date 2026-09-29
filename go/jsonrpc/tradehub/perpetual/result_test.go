package perpetual

import (
	"encoding/json"
	"testing"
)

// TestAccountBalanceEncoding keeps mode-specific fields optional and decimal strings exact.
//
// Version:
//   - 2026-09-29: Added.
func TestAccountBalanceEncoding(t *testing.T) {
	withdrawable := "100.000001"
	for _, mode := range []string{"disabled", "unifiedAccount", "portfolioMargin"} {
		r := AccountResult{AccountMode: mode}
		switch mode {
		case "disabled":
			r.TradingSupported = true
			r.Margin = &MarginSummary{AccountValue: "100.000001"}
			r.Withdrawable = &withdrawable
		case "unifiedAccount":
			r.TradingSupported = true
			r.UnifiedBalance = &UnifiedBalance{Coin: "USDC", Token: 0, Total: "995.364718", Hold: "78.420522"}
		}
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(b, &fields); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"margin", "withdrawable", "unifiedBalance"} {
			want := mode == "disabled" && key != "unifiedBalance" || mode == "unifiedAccount" && key == "unifiedBalance"
			if _, got := fields[key]; got != want {
				t.Fatalf("%s field %s present=%t", mode, key, got)
			}
		}
		var decoded AccountResult
		if err := json.Unmarshal(b, &decoded); err != nil {
			t.Fatal(err)
		}
		if mode == "unifiedAccount" && (*decoded.UnifiedBalance != *r.UnifiedBalance) {
			t.Fatal("unified precision changed")
		}
	}
}
