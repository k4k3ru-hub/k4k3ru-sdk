package quote

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

func str(value string) *string { return &value }

func validResult() Result {
	p := validParams()
	return Result{MarketType: p.MarketType, Symbol: p.Symbol, Side: p.Side, Quantity: p.Quantity,
		SentAt: time.Date(2026, 10, 20, 0, 0, 0, 123000, time.UTC).UnixMicro(),
		Quotes: []Quote{{Market: market.MarketRef{Venue: market.Hyperliquid, Network: "mainnet", VenueSymbol: "SUI"},
			Status: StatusVWAP, NetPrice: str("2"),
			NetPayQuantity:     &market.Quantity{Amount: "100000000", Decimals: 6},
			NetReceiveQuantity: &market.Quantity{Amount: "50000000000", Decimals: 9},
		}},
	}
}

// TestResultCapabilities separates full estimates, references and unavailable markets.
//
// Version:
//   - 2026-10-20: Added.
func TestResultCapabilities(t *testing.T) {
	for _, typ := range []market.MarketType{"spot", "perpetual"} {
		for _, side := range []Side{SideBuy, SideSell} {
			for _, status := range []Status{StatusVWAP, StatusFallbackReference, StatusUnavailable} {
				r := validResult()
				r.MarketType, r.Side, r.Quotes[0].Status = typ, side, status
				q := &r.Quotes[0]
				if typ == "perpetual" || status != StatusVWAP {
					q.NetPayQuantity, q.NetReceiveQuantity = nil, nil
				}
				if status == StatusUnavailable {
					q.NetPrice = nil
				}
				if err := r.Validate(); err != nil {
					t.Fatalf("%s %s %s: %v", typ, side, status, err)
				}
				data, err := json.Marshal(r)
				if err != nil {
					t.Fatal(err)
				}
				if (typ == "perpetual" || status != StatusVWAP) && (strings.Contains(string(data), "netPayQuantity") || strings.Contains(string(data), "netReceiveQuantity")) {
					t.Fatal("inapplicable quantities were published")
				}
				var decoded Result
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Fatal(err)
				}
				if err := decoded.Validate(); err != nil || decoded.SentAt != r.SentAt {
					t.Fatalf("invalid result round trip: %v", err)
				}
			}
		}
	}
	r := validResult()
	r.Quotes[0].Fees = &Fees{Taker: &Fee{Token: FeeToken{AssetID: "USDC", Symbol: "USDC"}, Quantity: market.Quantity{Amount: "0", Decimals: 6}}}
	if err := r.Validate(); err != nil {
		t.Fatalf("known zero fee rejected: %v", err)
	}
	r.Quotes[0].Fees = nil
	if err := r.Validate(); err != nil {
		t.Fatalf("unknown fee breakdown rejected: %v", err)
	}
}

// TestResultRejectsContradictions prevents incomplete quantities from masquerading as executable estimates.
//
// Version:
//   - 2026-10-20: Added.
func TestResultRejectsContradictions(t *testing.T) {
	for name, change := range map[string]func(*Result){
		"zero timestamp": func(r *Result) { r.SentAt = 0 },
		"no quotes":      func(r *Result) { r.Quotes = nil },
		"duplicate":      func(r *Result) { r.Quotes = append(r.Quotes, r.Quotes[0]) },
		"unresolved market": func(r *Result) {
			r.Quotes[0].Market.VenueSymbol = ""
		},
		"unknown status":        func(r *Result) { r.Quotes[0].Status = "reference" },
		"missing price":         func(r *Result) { r.Quotes[0].NetPrice = nil },
		"nan price":             func(r *Result) { r.Quotes[0].NetPrice = str("NaN") },
		"exponent price":        func(r *Result) { r.Quotes[0].NetPrice = str("1e2") },
		"zero price":            func(r *Result) { r.Quotes[0].NetPrice = str("0.0") },
		"negative price":        func(r *Result) { r.Quotes[0].NetPrice = str("-2") },
		"too long price":        func(r *Result) { r.Quotes[0].NetPrice = str(strings.Repeat("1", 513)) },
		"missing receipt":       func(r *Result) { r.Quotes[0].NetReceiveQuantity = nil },
		"zero receipt":          func(r *Result) { r.Quotes[0].NetReceiveQuantity.Amount = "0" },
		"perpetual with tokens": func(r *Result) { r.MarketType = "perpetual" },
		"unavailable with price": func(r *Result) {
			r.Quotes[0].Status = StatusUnavailable
			r.Quotes[0].NetPayQuantity, r.Quotes[0].NetReceiveQuantity = nil, nil
		},
		"reference with quantities": func(r *Result) { r.Quotes[0].Status = StatusFallbackReference },
		"reference with fees": func(r *Result) {
			r.Quotes[0].Status = StatusFallbackReference
			r.Quotes[0].NetPayQuantity, r.Quotes[0].NetReceiveQuantity = nil, nil
			r.Quotes[0].Fees = &Fees{Swap: &Fee{}}
		},
		"empty fees": func(r *Result) { r.Quotes[0].Fees = &Fees{} },
		"no fee token": func(r *Result) {
			r.Quotes[0].Fees = &Fees{Swap: &Fee{Quantity: market.Quantity{Amount: "0", Decimals: 6}}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := validResult()
			change(&r)
			if err := r.Validate(); err == nil {
				t.Fatal("contradictory response accepted")
			}
		})
	}
}
