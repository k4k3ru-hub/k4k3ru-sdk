package openinterest

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

func params() Params {
	return Params{AssetType: "crypto", Symbol: "SUI/USDC", Markets: []market.MarketTarget{{Venue: "hyperliquid", Network: "mainnet"}}}
}

func ptr[T any](v T) *T { return &v }

func result() Result {
	return Result{Symbol: "SUI/USDC", SentAt: 1792450800000001, Interests: []Interest{{
		Market: market.MarketRef{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: "SUI"},
		Status: StatusLive, Quantity: &market.Quantity{Amount: "12345", Decimals: 1},
		NotionalValue: ptr("1851.75"), NotionalCurrency: "USDC", Valuation: ValuationMark,
	}}}
}

// TestParamsContract verifies normalization, strict inputs and immutable failed decoding.
//
// Version:
//   - 2026-10-20: Added.
func TestParamsContract(t *testing.T) {
	p := params()
	p.Symbol, p.Markets[0].Venue, p.Markets[0].Network = " sui/usdc ", " Hyperliquid ", " Mainnet "
	p.Markets[0].VenueSymbol = " uSUI "
	n := p.Normalize()
	if n.Symbol != "SUI/USDC" || n.Markets[0].VenueSymbol != "uSUI" {
		t.Fatal("normalization lost selectors")
	}
	n.Markets[0].Network = "testnet"
	if p.Markets[0].Network != " Mainnet " || p.Validate() != nil {
		t.Fatal("normalization mutated caller")
	}
	good := `{"assetType":"crypto","symbol":"SUI/USDC","markets":[{"venue":"hyperliquid","network":"mainnet"}]}`
	for _, raw := range []string{
		`null`, `{}`, strings.Replace(good, `"assetType":"crypto",`, "", 1),
		strings.Replace(good, `"crypto"`, `null`, 1),
		strings.Replace(good, `"crypto"`, `"fx"`, 1),
		strings.Replace(good, `"SUI/USDC"`, `"SUI/SUI"`, 1),
		strings.Replace(good, `"SUI/USDC"`, `"SUI/USDC,BTC"`, 1),
		strings.Replace(good, `"markets":`, `"marketType":"perpetual","markets":`, 1),
		strings.Replace(good, `"symbol":`, `"symbol":"BTC/USDC","symbol":`, 1),
		strings.Replace(good, `,"network":"mainnet"`, "", 1),
		strings.Replace(good, `"network":"mainnet"`, `"network":null`, 1),
		strings.Replace(good, `[{"venue":"hyperliquid","network":"mainnet"}]`, `[]`, 1),
		strings.Replace(good, `[{"venue":"hyperliquid","network":"mainnet"}]`, `null`, 1),
	} {
		p := params()
		before := p.Normalize()
		if err := json.Unmarshal([]byte(raw), &p); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("bad input accepted or cause lost: %v", err)
		}
		if !reflect.DeepEqual(before, p) {
			t.Fatal("failed decoding mutated receiver")
		}
	}
	p = params()
	p.Markets = append(p.Markets, p.Markets[0])
	if !errors.Is(p.Validate(), apperror.InvalidParameter()) {
		t.Fatal("duplicate target accepted")
	}
	p.Markets = make([]market.MarketTarget, MaximumMarkets+1)
	if !errors.Is(p.Validate(), apperror.InvalidParameter()) {
		t.Fatal("unbounded targets accepted")
	}
	var decoded Params
	if err := json.Unmarshal([]byte(good), &decoded); err != nil || !reflect.DeepEqual(decoded, params()) {
		t.Fatalf("valid request failed: %v", err)
	}
}

// TestResultContract preserves zero, decimal units, valuation and missing observations.
//
// Version:
//   - 2026-10-20: Added.
func TestResultContract(t *testing.T) {
	for _, status := range []Status{StatusLive, StatusStale} {
		for _, valuation := range []Valuation{ValuationMark, ValuationIndex, ValuationOracle, ValuationVenueReported} {
			r := result()
			r.Interests[0].Status, r.Interests[0].Valuation = status, valuation
			r.Interests[0].Quantity, r.Interests[0].NotionalValue = &market.Quantity{Amount: "0", Decimals: 0}, ptr("0")
			if err := r.Validate(); err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var decoded Result
			if err := json.Unmarshal(data, &decoded); err != nil || !reflect.DeepEqual(r, decoded) {
				t.Fatalf("quantity or microseconds lost: %v", err)
			}
		}
	}
	r := result()
	r.Interests[0] = Interest{Market: r.Interests[0].Market, Status: StatusUnavailable}
	data, err := json.Marshal(r)
	if err != nil || r.Validate() != nil {
		t.Fatal("unavailable rejected", err)
	}
	for _, field := range []string{`"quantity":`, `"notionalValue":`, `"notionalCurrency":`, `"valuation":`} {
		if strings.Contains(string(data), field) {
			t.Fatal("missing observation published values")
		}
	}
	for name, change := range map[string]func(*Result){
		"timestamp":      func(r *Result) { r.SentAt = 0 },
		"empty":          func(r *Result) { r.Interests = nil },
		"duplicate":      func(r *Result) { r.Interests = append(r.Interests, r.Interests[0]) },
		"unresolved":     func(r *Result) { r.Interests[0].Market.VenueSymbol = "" },
		"status":         func(r *Result) { r.Interests[0].Status = "" },
		"quantity":       func(r *Result) { r.Interests[0].Quantity = nil },
		"negative":       func(r *Result) { r.Interests[0].Quantity.Amount = "-1" },
		"notional":       func(r *Result) { r.Interests[0].NotionalValue = nil },
		"nan":            func(r *Result) { r.Interests[0].NotionalValue = ptr("NaN") },
		"negative value": func(r *Result) { r.Interests[0].NotionalValue = ptr("-1") },
		"exponent":       func(r *Result) { r.Interests[0].NotionalValue = ptr("1e5") },
		"long":           func(r *Result) { r.Interests[0].NotionalValue = ptr(strings.Repeat("1", 513)) },
		"currency":       func(r *Result) { r.Interests[0].NotionalCurrency = "" },
		"currency space": func(r *Result) { r.Interests[0].NotionalCurrency = " USDC" },
		"valuation":      func(r *Result) { r.Interests[0].Valuation = "mid" },
		"unavailable":    func(r *Result) { r.Interests[0].Status = StatusUnavailable },
	} {
		t.Run(name, func(t *testing.T) {
			r := result()
			change(&r)
			if err := r.Validate(); err == nil {
				t.Fatal("contradictory result accepted")
			}
		})
	}
}
