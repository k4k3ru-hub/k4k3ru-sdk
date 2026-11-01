package overview

import (
	"encoding/json"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/funding"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/markets"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/openinterest"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/change"
	"strings"
	"testing"
	"time"
)

func testPointer[T any](v T) *T { return &v }
func fixture() Result {
	at := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC).UnixMicro()
	observed := at - 1_000_000
	expires := observed + 86400000000
	times := PointTimes{ObservedAt: &observed, FreshUntil: testPointer(observed + 60_000_000), ExpiresAt: &expires}
	p := Price{Definition: "composite", Currency: "USD", Method: "volume_weighted_reference", Interval: "60s", Price: testPointer("3.12"), Status: price.StatusLive, VenueCount: 1, PointTimes: times}
	c := PriceChanges{Definition: "composite", Currency: "USD", Method: "volume_weighted_reference", Price: p.Price, PointTimes: times, Changes: make([]change.Change, 6)}
	for i, period := range periods {
		c.Changes[i] = change.Change{Period: change.Period(period), Status: change.StatusUnavailable}
	}
	return Result{AssetType: market.AssetClassCrypto, Asset: "sui", Symbol: "SUI", Name: "Sui", MarketTypes: []market.MarketType{market.MarketTypeSpot}, Snapshot: Snapshot{ID: "publication-1", GeneratedAt: at}, SentAt: at + 1, Groups: []Group{{MarketType: market.MarketTypeSpot, Price: Section[Price]{State: StateReady, CalculatedAt: &at, ExpiresAt: &expires, Data: &p}, PriceChanges: Section[PriceChanges]{State: StateReady, CalculatedAt: &at, ExpiresAt: &expires, Data: &c}, PriceRanges: Section[PriceRanges]{State: StateUnavailable, Reason: testPointer("not_ready")}, Volume: Section[Volume]{State: StateUnavailable, Reason: testPointer("not_ready")}, Markets: Section[Markets]{State: StateUnavailable, Reason: testPointer("not_ready")}, Funding: Section[[]Funding]{State: StateNotApplicable}, OpenInterest: Section[[]OpenInterest]{State: StateNotApplicable}}}}
}

// TestOverviewSelectors verifies strict canonical selectors and atomic failure decoding.
//
// Version:
//   - 2026-11-01: Added.
func TestOverviewSelectors(t *testing.T) {
	original := Params{AssetType: "crypto", Asset: "sui"}
	for _, raw := range []string{`null`, `{}`, `{"assetType":"crypto","asset":null}`, `{"assetType":"crypto","asset":"SUI"}`, `{"assetType":"crypto","asset":"sui","asset":"btc"}`, `{"assetType":"crypto","asset":"sui","refresh":true}`} {
		p := original
		if err := json.Unmarshal([]byte(raw), &p); err == nil || p != original {
			t.Fatalf("accepted or partially decoded invalid selectors: %s %+v %v", raw, p, err)
		}
	}
}

// TestOverviewResponseScopeAndState verifies bounded strict decoding, immutable scope and paired-current consistency.
//
// Version:
//   - 2026-11-01: Added.
func TestOverviewResponseScopeAndState(t *testing.T) {
	valid := fixture()
	raw, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	value, err := DecodeResult(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := value.ValidateFor(Params{AssetType: "crypto", Asset: "btc"}); err == nil {
		t.Fatal("wrong asset accepted")
	}
	for _, mutate := range []func(*Result){func(r *Result) { r.Groups[0].MarketType = "perpetual" }, func(r *Result) { r.Snapshot.GeneratedAt = r.SentAt + 1 }, func(r *Result) { r.Groups[0].PriceChanges.Data.Price = testPointer("4") }, func(r *Result) { r.Groups[0].Price.Data.ObservedAt = nil }, func(r *Result) { r.Groups[0].PriceChanges.Data.Changes[1].Period = "5m" }, func(r *Result) { r.Groups[0].Price.State = StateStale }, func(r *Result) { r.Groups[0].Price.ExpiresAt = testPointer(r.Snapshot.GeneratedAt + 86400000001) }} {
		var r Result
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatal(err)
		}
		mutate(&r)
		if err := r.Validate(); err == nil {
			t.Fatal("invalid response accepted")
		}
	}
	for _, bad := range []string{`null`, `[]`, strings.Replace(string(raw), `"sentAt":`, `"unknown":true,"sentAt":`, 1), strings.Replace(string(raw), `"price":"3.12"`, `"price":"3.12","price":"4"`, 1), strings.Repeat(" ", MaximumResponseBytes+1)} {
		if _, err := DecodeResult([]byte(bad)); err == nil {
			t.Fatal("ambiguous or oversized response accepted")
		}
	}
}

// TestElapsedFundingDeadline allows retained stale estimates whose settlement target predates receipt.
//
// Version:
//   - 2026-11-01: Added.
func TestElapsedFundingDeadline(t *testing.T) {
	times := PointTimes{ObservedAt: testPointer(int64(100)), FreshUntil: testPointer(int64(99)), ExpiresAt: testPointer(int64(1000))}
	if err := validatePoint(times, true, 101); err != nil {
		t.Fatal(err)
	}
}

func unavailableCEXFixture() Result {
	r := fixture()
	r.MarketTypes = []market.MarketType{"spot", "perpetual"}
	spot := r.Groups[0]
	perp := spot
	perp.MarketType = "perpetual"
	r.Groups = []Group{spot, perp}
	at, expires := r.Snapshot.GeneratedAt, *spot.Price.ExpiresAt
	ref := market.MarketRef{Venue: "binance", Network: "mainnet", VenueSymbol: "SUIUSDC"}
	for i := range r.Groups {
		r.Groups[i].Markets = Section[Markets]{State: StateReady, CalculatedAt: &at, ExpiresAt: &expires, Data: &Markets{Total: 1, Eligible: 1, Items: []Market{{Market: markets.Market{Market: ref, Symbol: "SUI/USDC", LiquidityModel: "order-book", BaseAssetID: "SUI", QuoteAssetID: "USDC", CompositeEligible: true}, Price: MarketPrice{Currency: "USD", Method: "last_trade", Status: price.StatusUnavailable}}}}}
	}
	r.Groups[1].Funding = Section[[]Funding]{State: StateReady, CalculatedAt: &at, ExpiresAt: &expires, Data: testPointer([]Funding{{Rate: funding.Rate{Market: ref, Status: funding.StatusUnavailable}, Symbol: "SUI/USDC"}})}
	r.Groups[1].OpenInterest = Section[[]OpenInterest]{State: StateReady, CalculatedAt: &at, ExpiresAt: &expires, Data: testPointer([]OpenInterest{{Interest: openinterest.Interest{Market: ref, Status: openinterest.StatusUnavailable}, Symbol: "SUI/USDC"}})}
	return r
}

// TestRegisteredCEXMetadataWithoutNumericalPrices preserves nullable registered rows without widening Price.Get.
//
// Version:
//   - 2026-11-01: Added.
func TestRegisteredCEXMetadataWithoutNumericalPrices(t *testing.T) {
	r := unavailableCEXFixture()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeResult(raw); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Result){
		func(r *Result) {
			p := &r.Groups[0].Markets.Data.Items[0].Price
			p.Price = testPointer("1")
			p.Status = price.StatusLive
			p.PointTimes = r.Groups[0].Price.Data.PointTimes
		},
		func(r *Result) { r.Groups[0].Markets.Data.Items[0].Price.Currency = "EUR" },
		func(r *Result) { r.Groups[0].Markets.Data.Items[0].Price.Method = "pool_price" },
		func(r *Result) {
			r.Groups[0].Markets.Data.Items[0].Price.ObservedAt = testPointer(r.Snapshot.GeneratedAt)
		},
		func(r *Result) { r.Groups[0].Markets.Data.Items[0].Market.Market.Network = "testnet" },
		func(r *Result) { r.Groups[0].Markets.Data.Items[0].Market.Market.Venue = "" },
		func(r *Result) { (*r.Groups[1].Funding.Data)[0].Rate.Rate = testPointer("0") },
		func(r *Result) {
			(*r.Groups[1].OpenInterest.Data)[0].Quantity = &market.Quantity{Amount: "1", Decimals: 0}
		},
	} {
		bad := unavailableCEXFixture()
		mutate(&bad)
		if err := bad.Validate(); err == nil {
			t.Fatal("malformed or fabricated CEX numerical data accepted")
		}
	}
	m := r.Groups[0].Markets.Data.Items[0]
	if err := (price.Result{Asset: "sui", MarketType: "spot", Definition: "composite", Currency: "USD", Method: "volume_weighted_reference", Interval: "60s", Status: price.StatusUnavailable, SentAt: r.SentAt, Markets: []price.MarketPrice{{Market: m.Market.Market, Symbol: m.Symbol, Currency: "USD", Method: "last_trade", Status: price.StatusUnavailable}}}).Validate(); err == nil {
		t.Fatal("existing Price.Get CEX scope changed")
	}
}
