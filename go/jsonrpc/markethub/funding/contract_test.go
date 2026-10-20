package funding

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
	return Result{Symbol: "SUI/USDC", SentAt: 1792450800000001, Rates: []Rate{{
		Market: market.MarketRef{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: "SUI"},
		Status: StatusLive, Kind: KindCurrentEstimate, Rate: ptr("0.000012"), IntervalMinutes: ptr(int32(60)), FundingAt: ptr(int64(1792454400000001)),
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

// TestResultContract preserves signed ratios and omits missing rates instead of inventing zero.
//
// Version:
//   - 2026-10-20: Added.
func TestResultContract(t *testing.T) {
	for _, value := range []string{"0", "-0.000012", "0.000012"} {
		for _, status := range []Status{StatusLive, StatusStale} {
			r := result()
			r.Rates[0].Rate, r.Rates[0].Status = &value, status
			if err := r.Validate(); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var decoded Result
			if err := json.Unmarshal(encoded, &decoded); err != nil || !reflect.DeepEqual(decoded, r) {
				t.Fatalf("ratio or microseconds lost: %v", err)
			}
		}
	}
	r := result()
	r.Rates[0].FundingAt = nil
	if err := r.Validate(); err != nil {
		t.Fatal("unknown target time rejected", err)
	}
	r.Rates[0] = Rate{Market: r.Rates[0].Market, Status: StatusUnavailable}
	data, err := json.Marshal(r)
	if err != nil || r.Validate() != nil {
		t.Fatal("unavailable result rejected", err)
	}
	for _, field := range []string{`"rate":`, `"kind":`, `"intervalMinutes":`, `"fundingAt":`} {
		if strings.Contains(string(data), field) {
			t.Fatal("unavailable observation published values")
		}
	}
	for name, change := range map[string]func(*Result){
		"send time":          func(r *Result) { r.SentAt = 0 },
		"empty":              func(r *Result) { r.Rates = nil },
		"duplicate":          func(r *Result) { r.Rates = append(r.Rates, r.Rates[0]) },
		"unresolved":         func(r *Result) { r.Rates[0].Market.VenueSymbol = "" },
		"status":             func(r *Result) { r.Rates[0].Status = "estimated" },
		"settled":            func(r *Result) { r.Rates[0].Kind = "settled" },
		"missing rate":       func(r *Result) { r.Rates[0].Rate = nil },
		"nan":                func(r *Result) { r.Rates[0].Rate = ptr("NaN") },
		"exponent":           func(r *Result) { r.Rates[0].Rate = ptr("1e-5") },
		"percent":            func(r *Result) { r.Rates[0].Rate = ptr("0.01%") },
		"long":               func(r *Result) { r.Rates[0].Rate = ptr(strings.Repeat("1", 513)) },
		"interval":           func(r *Result) { r.Rates[0].IntervalMinutes = ptr(int32(0)) },
		"no interval":        func(r *Result) { r.Rates[0].IntervalMinutes = nil },
		"funding time":       func(r *Result) { r.Rates[0].FundingAt = ptr(int64(0)) },
		"unavailable values": func(r *Result) { r.Rates[0].Status = StatusUnavailable },
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
