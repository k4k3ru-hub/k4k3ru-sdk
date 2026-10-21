package asset

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

// TestParamsStrictDecoding rejects ambiguous selectors and preserves a reused destination on failure.
//
// Version:
//   - 2026-10-21: Added.
func TestParamsStrictDecoding(t *testing.T) {
	good := Params{AssetType: "crypto", Asset: "sui"}
	data, err := json.Marshal(good)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Params
	if err := json.Unmarshal(data, &decoded); err != nil || decoded != good {
		t.Fatal(decoded, err)
	}
	for _, raw := range []string{
		`null`, `{}`, `[]`,
		`{"asset":"sui"}`, `{"assetType":"crypto"}`,
		`{"assetType":null,"asset":"sui"}`, `{"assetType":"crypto","asset":null}`,
		`{"assetType":"fx","asset":"sui"}`, `{"assetType":"","asset":"sui"}`,
		`{"assetType":"crypto","asset":""}`, `{"assetType":"crypto","asset":"SUI"}`,
		`{"assetType":"crypto","asset":" sui "}`, `{"assetType":"crypto","asset":"sui/usdc"}`,
		`{"assetType":"crypto","asset":"sui","marketType":"spot"}`,
		`{"assetType":"crypto","asset":"sui","asset":"btc"}`,
		`{"assetType":"crypto","asset":"sui","Asset":"btc"}`,
		`{"assetType":"crypto","asset":1}`,
		`{"assetType":"crypto","asset":"sui"}{}`,
		`{"assetType":"crypto","asset":"` + strings.Repeat("a", 65) + `"}`,
	} {
		p := good
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			t.Fatalf("accepted %s", raw)
		}
		if p != good {
			t.Fatal("failed decoding mutated destination")
		}
	}
	for _, p := range []Params{{Asset: "sui"}, {AssetType: "crypto"}, {AssetType: "crypto", Asset: " SUI "}} {
		if !errors.Is(p.Validate(), apperror.InvalidParameter()) {
			t.Fatal("validation cause lost")
		}
	}
	for _, slug := range []string{"sui-token-reference", strings.Repeat("a", 64)} {
		if err := (Params{AssetType: "crypto", Asset: slug}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
	var missing *Params
	if err := missing.UnmarshalJSON(data); err == nil {
		t.Fatal("nil receiver accepted")
	}
}

// TestResultContract validates metadata coverage and exact microsecond round trips.
//
// Version:
//   - 2026-10-21: Added.
func TestResultContract(t *testing.T) {
	valid := Result{AssetType: "crypto", Asset: "sui", Symbol: "SUI", Name: "Sui", MarketTypes: []market.MarketType{"spot", "perpetual"}, SentAt: 1792548000123456}
	data, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	var got Result
	if err := json.Unmarshal(data, &got); err != nil || !reflect.DeepEqual(valid, got) {
		t.Fatal(got, err)
	}
	for _, kinds := range [][]market.MarketType{{"spot"}, {"perpetual"}, {"spot", "perpetual"}} {
		r := valid
		r.MarketTypes = kinds
		if err := r.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for name, mutate := range map[string]func(*Result){
		"asset type":     func(r *Result) { r.AssetType = "" },
		"asset":          func(r *Result) { r.Asset = "SUI" },
		"symbol":         func(r *Result) { r.Symbol = "" },
		"name":           func(r *Result) { r.Name = "" },
		"control":        func(r *Result) { r.Name = "Sui\n" },
		"nil coverage":   func(r *Result) { r.MarketTypes = nil },
		"empty coverage": func(r *Result) { r.MarketTypes = []market.MarketType{} },
		"duplicate":      func(r *Result) { r.MarketTypes = []market.MarketType{"spot", "spot"} },
		"reversed":       func(r *Result) { r.MarketTypes = []market.MarketType{"perpetual", "spot"} },
		"invalid type":   func(r *Result) { r.MarketTypes = []market.MarketType{"perp"} },
		"too many":       func(r *Result) { r.MarketTypes = []market.MarketType{"spot", "perpetual", "spot"} },
		"unsent":         func(r *Result) { r.SentAt = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			r := valid
			mutate(&r)
			if r.Validate() == nil {
				t.Fatal("invalid result accepted")
			}
		})
	}
}
