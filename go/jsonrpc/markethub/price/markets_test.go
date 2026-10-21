package price

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

// TestMarketPriceSelectors verifies strict identities, bounded batches and omission compatibility.
//
// Version:
//   - 2026-10-21: Added.
func TestMarketPriceSelectors(t *testing.T) {
	base := `{"assetType":"crypto","asset":"sui","marketType":"perpetual"`
	ref := `{"venue":"hyperliquid","network":"mainnet","venueSymbol":"SUI"}`
	for _, body := range []string{base + `}`, base + `,"markets":[` + ref + `]}`} {
		var p Params
		if err := json.Unmarshal([]byte(body), &p); err != nil {
			t.Fatal(err)
		}
	}
	for _, suffix := range []string{`,"markets":null}`, `,"markets":[]}`, `,"markets":[null]}`, `,"markets":[` + ref + `,` + ref + `]}`, `,"markets":[{"venue":"hyperliquid","network":"mainnet"}]}`, `,"markets":[` + strings.Replace(ref, "mainnet", "testnet", 1) + `]}`, `,"markets":[` + strings.Replace(ref, "hyperliquid", "binance", 1) + `]}`, `,"unknown":1}`, `,"markets":[` + ref + `],"Markets":[]}`} {
		var p Params
		if err := json.Unmarshal([]byte(base+suffix), &p); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("accepted invalid request: %s %v", suffix, err)
		}
	}
	p := Params{AssetType: "crypto", Asset: "sui", MarketType: "perpetual"}
	for i := 0; i < MaximumMarkets; i++ {
		p.Markets = append(p.Markets, market.MarketRef{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: fmt.Sprint(i)})
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Markets = append(p.Markets, market.MarketRef{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: "65"})
	if p.Validate() == nil {
		t.Fatal("unbounded markets")
	}
	p.Markets = p.Markets[:1]
	normalized := p.Normalize()
	normalized.Markets[0].VenueSymbol = "changed"
	if p.Markets[0].VenueSymbol == "changed" {
		t.Fatal("normalize mutated caller")
	}
}

// TestMarketPriceResultBinding rejects dropped, substituted and malformed market results.
//
// Version:
//   - 2026-10-21: Added.
func TestMarketPriceResultBinding(t *testing.T) {
	ref := market.MarketRef{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: "SUI"}
	q := Params{AssetType: "crypto", Asset: "sui", MarketType: "perpetual", Markets: []market.MarketRef{ref}}
	value := "1.234"
	r := Result{Asset: "sui", MarketType: "perpetual", Definition: "composite", Currency: "USD", Method: "volume_weighted_latest", Interval: "60s", Status: StatusUnavailable, SentAt: 1792508400000001,
		Markets: []MarketPrice{{Market: ref, Symbol: "SUI/USDC", Currency: "USD", Method: "last_trade", Price: &value, Status: StatusLive}}}
	if err := r.ValidateFor(q); err != nil {
		t.Fatal(err)
	}
	r.Markets[0].Status = StatusStale
	if err := r.ValidateFor(q); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Result){
		func(r *Result) { r.Markets = nil }, func(r *Result) { r.Markets[0].Market.VenueSymbol = "BTC" },
		func(r *Result) { r.Markets[0].Status = StatusUnavailable }, func(r *Result) { r.Markets[0].Method = "mid" },
		func(r *Result) { r.Markets[0].Price = nil }, func(r *Result) { r.Markets = append(r.Markets, r.Markets[0]) },
	} {
		bad := r
		bad.Markets = append([]MarketPrice(nil), r.Markets...)
		mutate(&bad)
		if bad.ValidateFor(q) == nil {
			t.Fatalf("accepted invalid result: %+v", bad)
		}
	}
	r.Markets[0].Price = nil
	r.Markets[0].Status = StatusUnavailable
	if err := r.ValidateFor(q); err != nil {
		t.Fatal(err)
	}
	q.Markets = nil
	if r.ValidateFor(q) == nil {
		t.Fatal("unrequested markets")
	}
	r.Markets = nil
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"markets"`) {
		t.Fatal("omission changed legacy response")
	}
}
