package markets

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

func params() Params {
	return Params{AssetType: market.AssetClassCrypto, Asset: "sui", MarketType: market.MarketTypeSpot}
}

func result() Result {
	return Result{Asset: "sui", MarketType: market.MarketTypeSpot, Page: 1, Limit: DefaultLimit, Total: 3, SentAt: 1792508400000001, Markets: []Market{
		{Market: market.MarketRef{Venue: market.Binance, Network: "mainnet", VenueSymbol: "SUIUSDC"},
			Symbol: "SUI/USDC", LiquidityModel: market.LiquidityModelOrderBook, BaseAssetID: "SUI", QuoteAssetID: "USDC", CompositeEligible: true},
		{Market: market.MarketRef{Venue: market.Bluefin, Network: "mainnet", Chain: "sui", PoolID: "test-pool-1"},
			Symbol: "SUI/USDC", LiquidityModel: market.LiquidityModelAMM, BaseAssetID: "0x2::sui::SUI", QuoteAssetID: "test-usdc", CompositeEligible: true, CandlesSupported: true},
		{Market: market.MarketRef{Venue: market.Bluefin, Network: "mainnet", Chain: "sui", PoolID: "test-pool-2"},
			Symbol: "SUI/USDC", LiquidityModel: market.LiquidityModelAMM, BaseAssetID: "0x2::sui::SUI", QuoteAssetID: "test-usdc", CompositeEligible: true, CandlesSupported: true},
	}}
}

// TestParamsContract verifies defaults, strict wire validation and immutable failed decoding.
//
// Version:
//   - 2026-10-20: Added.
func TestParamsContract(t *testing.T) {
	p := params()
	before := p
	n := p.Normalize()
	if n.Page != 1 || n.Limit != 100 || p != before || p.Validate() != nil {
		t.Fatal("defaults, immutability or validation failed")
	}
	good := `{"assetType":"crypto","asset":"sui","marketType":"spot"}`
	var decoded Params
	if err := json.Unmarshal([]byte(good), &decoded); err != nil || decoded != n {
		t.Fatalf("defaults were not decoded: %v", err)
	}
	data, err := json.Marshal(p)
	if err != nil || strings.Contains(string(data), `"page"`) || strings.Contains(string(data), `"limit"`) {
		t.Fatal("Go zero did not omit pagination", err)
	}
	complete := `{"assetType":"crypto","asset":"sui","marketType":"perpetual","page":2,"limit":500}`
	if err := json.Unmarshal([]byte(complete), &decoded); err != nil || decoded.Page != 2 || decoded.Limit != 500 || decoded.MarketType != market.MarketTypePerpetual {
		t.Fatalf("explicit selectors rejected: %v", err)
	}
	for _, raw := range []string{
		`null`, `[]`, `{}`, good + `{}`,
		strings.Replace(good, `"assetType":"crypto",`, "", 1),
		strings.Replace(good, `"asset":"sui",`, "", 1),
		strings.Replace(good, `,"marketType":"spot"`, "", 1),
		strings.Replace(good, `"crypto"`, `null`, 1),
		strings.Replace(good, `"sui"`, `null`, 1),
		strings.Replace(good, `"spot"`, `null`, 1),
		strings.Replace(good, `"crypto"`, `"fx"`, 1),
		strings.Replace(good, `"sui"`, `"SUI"`, 1),
		strings.Replace(good, `"sui"`, `""`, 1),
		strings.Replace(good, `"sui"`, `"`+strings.Repeat("a", 65)+`"`, 1),
		strings.Replace(good, `"spot"`, `"perp"`, 1),
		strings.Replace(good, `"asset":`, `"asset":"btc","asset":`, 1),
		strings.Replace(good, `"asset":`, `"Asset":"btc","asset":`, 1),
		strings.Replace(good, `}`, `,"network":"mainnet"}`, 1),
		strings.Replace(good, `}`, `,"testMode":true}`, 1),
	} {
		target := params().Normalize()
		before := target
		if err := target.UnmarshalJSON([]byte(raw)); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid selector accepted or cause lost: %v", err)
		}
		if target != before {
			t.Fatal("failed decoding changed the receiver")
		}
	}
	for _, field := range []string{"page", "limit"} {
		for _, value := range []string{"null", "0", "-1", "1.5", `"1"`, "true", "99999999999999999999999"} {
			raw := strings.TrimSuffix(good, "}") + `,"` + field + `":` + value + `}`
			target := params().Normalize()
			before := target
			if err := json.Unmarshal([]byte(raw), &target); !errors.Is(err, apperror.InvalidParameter()) || target != before {
				t.Fatalf("invalid pagination accepted or receiver changed: field=%s value=%s err=%v", field, value, err)
			}
		}
	}
	for _, change := range []func(*Params){
		func(p *Params) { p.Page = -1 },
		func(p *Params) { p.Limit = -1 },
		func(p *Params) { p.Limit = MaxLimit + 1 },
	} {
		p := params()
		change(&p)
		if !errors.Is(p.Validate(), apperror.InvalidParameter()) {
			t.Fatal("invalid Go pagination accepted")
		}
	}
	var absent *Params
	if !errors.Is(absent.UnmarshalJSON([]byte(good)), apperror.InvalidParameter()) {
		t.Fatal("nil receiver accepted")
	}
}

// TestMarketResultContract preserves distinct pools and separates eligibility from capabilities.
//
// Version:
//   - 2026-10-20: Added.
func TestMarketResultContract(t *testing.T) {
	r := result()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Result
	if err := json.Unmarshal(data, &decoded); err != nil || !reflect.DeepEqual(r, decoded) {
		t.Fatalf("round trip changed identities or microseconds: %v", err)
	}
	if !strings.Contains(string(data), `"candlesSupported":false`) || strings.Contains(string(data), `"price":`) {
		t.Fatal("capability or metadata-only response changed")
	}
	r.Markets[1].CompositeEligible, r.Markets[1].CandlesSupported = false, false
	if err := r.Validate(); err != nil {
		t.Fatal("ineligible listed market rejected", err)
	}
	r = result()
	r.Asset = "sui-abcdef0123456789"
	if err := r.Validate(); err != nil {
		t.Fatal("scoped asset ID was inferred from the symbol", err)
	}
	r = Result{Asset: "sui", MarketType: market.MarketTypePerpetual, Page: 1, Limit: 100, Total: 1, SentAt: 1792508400000001, Markets: []Market{{
		Market: market.MarketRef{Venue: market.Bluefin, Network: "mainnet", Chain: "sui", VenueSymbol: "SUI-PERP"},
		Symbol: "SUI/USDC", LiquidityModel: market.LiquidityModelOrderBook, BaseAssetID: "SUI", QuoteAssetID: "USDC", CompositeEligible: true, CandlesSupported: true,
	}}}
	if err := r.Validate(); err != nil {
		t.Fatal("Bluefin's order-book perpetual was inferred from its AMM venue", err)
	}
}

// TestResultPagination checks partial final pages, empty pages and overflow-safe distant pages.
//
// Version:
//   - 2026-10-20: Added.
func TestResultPagination(t *testing.T) {
	r := result()
	r.Page, r.Limit, r.Markets = 2, 2, r.Markets[2:]
	if err := r.Validate(); err != nil {
		t.Fatal("partial final page rejected", err)
	}
	for _, page := range []int{3, math.MaxInt} {
		r.Page, r.Markets = page, []Market{}
		if err := r.Validate(); err != nil {
			t.Fatal("empty out-of-range page rejected", err)
		}
	}
	r.Total = 0
	if err := r.Validate(); err != nil {
		t.Fatal("zero matches rejected", err)
	}
	r = result()
	r.Page, r.Limit, r.Total, r.Markets = math.MaxInt, 1, math.MaxInt, r.Markets[:1]
	if err := r.Validate(); err != nil {
		t.Fatal("largest representable final page rejected", err)
	}
}

// TestInvalidResult rejects inconsistent pagination, ambiguous identities and test networks.
//
// Version:
//   - 2026-10-20: Added.
func TestInvalidResult(t *testing.T) {
	for name, change := range map[string]func(*Result){
		"page zero":       func(r *Result) { r.Page = 0 },
		"limit zero":      func(r *Result) { r.Limit = 0 },
		"limit large":     func(r *Result) { r.Limit = MaxLimit + 1 },
		"total negative":  func(r *Result) { r.Total = -1 },
		"incomplete page": func(r *Result) { r.Total = 4 },
		"extra rows":      func(r *Result) { r.Total = 2 },
		"null rows":       func(r *Result) { r.Markets, r.Total = nil, 0 },
		"timestamp":       func(r *Result) { r.SentAt = 0 },
		"network":         func(r *Result) { r.Markets[0].Market.Network = "testnet" },
		"network empty":   func(r *Result) { r.Markets[0].Market.Network = "" },
		"scope":           func(r *Result) { r.Markets[0].Market.Venue = " BINANCE " },
		"empty identity":  func(r *Result) { r.Markets[0].Market.VenueSymbol = "" },
		"dual identity":   func(r *Result) { r.Markets[0].Market.PoolID = "another" },
		"pool chain":      func(r *Result) { r.Markets[1].Market.Chain = "" },
		"wrong model":     func(r *Result) { r.Markets[1].LiquidityModel = market.LiquidityModelOrderBook },
		"unknown model":   func(r *Result) { r.Markets[0].LiquidityModel = "unknown" },
		"duplicate":       func(r *Result) { r.Markets[2] = r.Markets[1] },
		"reordered":       func(r *Result) { r.Markets[1], r.Markets[2] = r.Markets[2], r.Markets[1] },
		"empty token":     func(r *Result) { r.Markets[0].BaseAssetID = "" },
		"long token":      func(r *Result) { r.Markets[0].BaseAssetID = strings.Repeat("a", 513) },
		"token controls":  func(r *Result) { r.Markets[0].QuoteAssetID = "USDC\n" },
		"symbol":          func(r *Result) { r.Markets[0].Symbol = "SUI" },
		"symbol case":     func(r *Result) { r.Markets[0].Symbol = "sui/USDC" },
		"symbol same":     func(r *Result) { r.Markets[0].Symbol = "SUI/SUI" },
	} {
		t.Run(name, func(t *testing.T) {
			r := result()
			change(&r)
			if err := r.Validate(); err == nil {
				t.Fatal("inconsistent result accepted")
			}
		})
	}
}
