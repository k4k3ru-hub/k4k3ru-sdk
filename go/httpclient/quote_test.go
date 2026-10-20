package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/quote"
)

func quoteParams() quote.Params {
	return quote.Params{AssetType: "crypto", MarketType: "spot", Symbol: "SUI/USDC", Side: quote.SideBuy,
		Quantity: market.Quantity{Amount: "100", Decimals: 0},
		Markets:  []market.MarketTarget{{Venue: market.Cetus, Chain: "sui", Network: "mainnet"}},
	}
}

func quoteResponse(p quote.Params) quote.Result {
	price := "2"
	r := quote.Result{MarketType: p.MarketType, Symbol: p.Symbol, Side: p.Side, Quantity: p.Quantity,
		SentAt: time.Date(2026, 10, 20, 0, 0, 0, 123000, time.UTC).UnixMicro()}
	for _, target := range p.Markets {
		ref := market.MarketRef(target)
		if ref.PoolID == "" && ref.VenueSymbol == "" {
			if ref.Chain == "sui" {
				ref.PoolID = "0x1"
			} else {
				ref.VenueSymbol = "SUI"
			}
		}
		q := quote.Quote{Market: ref, Status: quote.StatusVWAP, NetPrice: &price}
		if p.MarketType == "spot" {
			q.NetPayQuantity = &market.Quantity{Amount: "100000000", Decimals: 6}
			q.NetReceiveQuantity = &market.Quantity{Amount: "50000000000", Decimals: 9}
		}
		r.Quotes = append(r.Quotes, q)
	}
	return r
}

// TestQuoteComposition verifies lazy construction and immutable requests through the shared transport.
//
// Version:
//   - 2026-10-20: Added.
func TestQuoteComposition(t *testing.T) {
	calls := 0
	client, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var req jsonrpc.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if err := r.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if req.Method != jsonrpc.MethodMarketHubQuoteGet || string(req.Method) != "MarketHub.Quote.Get" || r.Method != http.MethodPost {
			t.Fatal("wrong method")
		}
		var p quote.Params
		if err := json.Unmarshal(req.Params, &p); err != nil {
			t.Fatal(err)
		}
		if p.AssetType != "crypto" || p.Symbol != "SUI/USDC" || p.Side != quote.SideBuy || p.Markets[0].Network != "mainnet" {
			t.Fatal("lost selectors")
		}
		return testResponse(t, req, quoteResponse(p), nil, 200), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	g := client.MarketHub()
	if calls != 0 || g == nil || g.Quote() == nil || g.Candles() == nil || g.Price() == nil || g.Price().Change() == nil || g.Price().Range() == nil || g.Volume() == nil {
		t.Fatal("incomplete or eager composition")
	}
	p := quoteParams()
	p.MarketType, p.Symbol, p.Side, p.Markets[0].Network = " SPOT ", " sui/usdc ", " Buy ", " MAINNET "
	before := p
	before.Markets = append([]market.MarketTarget(nil), p.Markets...)
	r, err := g.Quote().Get(t.Context(), p)
	if err != nil || r == nil || calls != 1 {
		t.Fatalf("request failed: %v", err)
	}
	if !reflect.DeepEqual(p, before) {
		t.Fatal("caller parameters mutated")
	}
	p = quoteParams()
	p.MarketType = "perpetual"
	p.Markets = []market.MarketTarget{{Venue: market.Hyperliquid, Network: "mainnet"}}
	r, err = g.Quote().Get(t.Context(), p)
	if err != nil || r == nil || r.Quotes[0].NetReceiveQuantity != nil || calls != 2 {
		t.Fatalf("perpetual request failed: %v", err)
	}
}

// TestQuoteResponseBinding rejects estimates for other selectors or market scopes.
//
// Version:
//   - 2026-10-20: Added.
func TestQuoteResponseBinding(t *testing.T) {
	p := quoteParams()
	p.Markets[0].PoolID = "0x1"
	p.Markets = append(p.Markets, market.MarketTarget{Venue: market.Turbos, Chain: "sui", Network: "mainnet"})
	for name, change := range map[string]func(*quote.Result){
		"symbol":        func(r *quote.Result) { r.Symbol = "BTC/USDC" },
		"side":          func(r *quote.Result) { r.Side = quote.SideSell },
		"quantity":      func(r *quote.Result) { r.Quantity.Amount = "101" },
		"scale":         func(r *quote.Result) { r.Quantity.Decimals = 1 },
		"network":       func(r *quote.Result) { r.Quotes[0].Market.Network = "testnet" },
		"chain":         func(r *quote.Result) { r.Quotes[0].Market.Chain = "ethereum" },
		"pool":          func(r *quote.Result) { r.Quotes[0].Market.PoolID = "0x2" },
		"venue":         func(r *quote.Result) { r.Quotes[0].Market.Venue = market.Turbos },
		"missing scope": func(r *quote.Result) { r.Quotes = r.Quotes[:1] },
	} {
		t.Run(name, func(t *testing.T) {
			r := quoteResponse(p)
			change(&r)
			c, err := newQuoteClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(r)
			}))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.Get(t.Context(), p); err == nil {
				t.Fatal("mismatched response accepted")
			}
		})
	}
	// One scope may resolve several pools. Equivalent scales and padded Sui IDs
	// remain the same request; neither comparison uses floating-point arithmetic.
	r := quoteResponse(p)
	r.Quantity = market.Quantity{Amount: "0001000", Decimals: 1}
	r.Quotes[0].Market.PoolID = "0x" + strings.Repeat("0", 63) + "1"
	extra := r.Quotes[1]
	extra.Market.PoolID = "0x2"
	r.Quotes = append(r.Quotes, extra)
	c, err := newQuoteClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(r)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(t.Context(), p); err != nil {
		t.Fatalf("valid expanded scope rejected: %v", err)
	}
	if quoteMarketMatches(market.MarketTarget{Venue: market.Hyperliquid, Network: "mainnet", VenueSymbol: "SUI"}, market.MarketRef{Venue: market.Hyperliquid, Network: "mainnet", VenueSymbol: "BTC"}) {
		t.Fatal("wrong venue symbol matched")
	}
	large := market.Quantity{Amount: "9007199254740993", Decimals: 9}
	if equalQuoteQuantity(large, market.Quantity{Amount: "9007199254740992", Decimals: 9}) {
		t.Fatal("quantity comparison lost precision")
	}
}

// TestQuoteFailures verifies preflight checks, malformed responses and inspectable non-retried errors.
//
// Version:
//   - 2026-10-20: Added.
func TestQuoteFailures(t *testing.T) {
	p := quoteParams()
	for _, raw := range []string{"null", "{", `{"quotes":[]}`} {
		c, err := newQuoteClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
			return []byte(raw), nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Get(t.Context(), p); err == nil {
			t.Fatal("malformed response accepted")
		}
	}
	calls := 0
	cause := apperror.OutOfTicks()
	c, err := newQuoteClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		calls++
		return nil, cause
	}))
	if err != nil {
		t.Fatal(err)
	}
	invalid := p
	invalid.Quantity.Amount = "0"
	if _, err := c.Get(t.Context(), invalid); !errors.Is(err, apperror.InvalidParameter()) || calls != 0 {
		t.Fatal("invalid input reached transport")
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.Get(canceled, p); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("canceled input reached transport")
	}
	if _, err := c.Get(nil, p); err == nil || calls != 0 {
		t.Fatal("nil context accepted")
	}
	if _, err := c.Get(t.Context(), p); !errors.Is(err, cause) || calls != 1 {
		t.Fatal("error lost or request retried")
	}
	var absent *QuoteClient
	if _, err := absent.Get(t.Context(), p); err == nil {
		t.Fatal("nil client accepted")
	}
	var group *MarketHubClient
	if group.Quote() != nil {
		t.Fatal("nil group returned a client")
	}
	if _, err := newQuoteClient(nil); err == nil {
		t.Fatal("nil caller accepted")
	}
}
