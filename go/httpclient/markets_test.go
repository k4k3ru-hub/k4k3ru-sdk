package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/markets"
)

func marketsParams() markets.Params {
	return markets.Params{AssetType: "crypto", Asset: "sui", MarketType: "perpetual"}
}

func marketsResponse() markets.Result {
	return markets.Result{Asset: "sui", MarketType: "perpetual", Page: 1, Limit: 100, Total: 1, SentAt: 1792508400000001, Markets: []markets.Market{{
		Market: market.MarketRef{Venue: market.Hyperliquid, Network: "mainnet", VenueSymbol: "SUI"},
		Symbol: "SUI/USDC", LiquidityModel: market.LiquidityModelOrderBook, BaseAssetID: "SUI", QuoteAssetID: "USDC", CompositeEligible: true, CandlesSupported: true,
	}}}
}

// TestMarketsComposition verifies lazy composition, method routing and one immutable request.
//
// Version:
//   - 2026-10-20: Added.
func TestMarketsComposition(t *testing.T) {
	calls := 0
	client, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var req jsonrpc.Request
		err := json.NewDecoder(r.Body).Decode(&req)
		closeErr := r.Body.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("request decode failed: %v %v", err, closeErr)
		}
		if req.Method != jsonrpc.MethodMarketHubMarketsList || string(req.Method) != "MarketHub.Markets.List" || r.Method != http.MethodPost {
			t.Fatal("wrong method")
		}
		var p markets.Params
		if err := json.Unmarshal(req.Params, &p); err != nil || p != marketsParams().Normalize() {
			t.Fatalf("wrong selectors: %v", err)
		}
		return testResponse(t, req, marketsResponse(), nil, 200), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	g := client.MarketHub()
	if calls != 0 || g.Markets() == nil || g.Price() == nil || g.Candles() == nil || g.Volume() == nil || g.Quote() == nil || g.Funding() == nil || g.OpenInterest() == nil {
		t.Fatal("incomplete or eager composition")
	}
	p := marketsParams()
	before := p
	result, err := g.Markets().List(t.Context(), p)
	if err != nil || result == nil || result.Markets[0].Market.VenueSymbol != "SUI" || calls != 1 || p != before {
		t.Fatalf("wrong response, retry or caller mutation: %v", err)
	}
}

// TestMarketsResponseBinding rejects valid catalog pages belonging to another request.
//
// Version:
//   - 2026-10-20: Added.
func TestMarketsResponseBinding(t *testing.T) {
	for name, change := range map[string]func(*markets.Result){
		"asset":       func(r *markets.Result) { r.Asset = "btc" },
		"market type": func(r *markets.Result) { r.MarketType = "spot" },
		"page":        func(r *markets.Result) { r.Page, r.Total = 2, 101 },
		"limit":       func(r *markets.Result) { r.Limit = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			result := marketsResponse()
			change(&result)
			if err := result.Validate(); err != nil {
				t.Fatal("fixture must be a valid page", err)
			}
			c, err := newMarketsClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(result)
			}))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.List(t.Context(), marketsParams()); err == nil {
				t.Fatal("other request's page accepted")
			}
		})
	}
}

// TestMarketsEmptyPage preserves totals without fetching another page.
//
// Version:
//   - 2026-10-20: Added.
func TestMarketsEmptyPage(t *testing.T) {
	calls := 0
	c, err := newMarketsClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		calls++
		result := marketsResponse()
		result.Page, result.Markets = 2, []markets.Market{}
		return json.Marshal(result)
	}))
	if err != nil {
		t.Fatal(err)
	}
	p := marketsParams()
	p.Page = 2
	r, err := c.List(t.Context(), p)
	if err != nil || r == nil || r.Total != 1 || r.Markets == nil || len(r.Markets) != 0 || calls != 1 {
		t.Fatalf("empty page rejected, total lost or another page fetched: %v", err)
	}
}

// TestMarketsFailures preserves error causes and sends no invalid or canceled requests.
//
// Version:
//   - 2026-10-20: Added.
func TestMarketsFailures(t *testing.T) {
	calls := 0
	cause := apperror.OutOfTicks()
	c, err := newMarketsClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		calls++
		return nil, cause
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.List(t.Context(), marketsParams()); !errors.Is(err, cause) || calls != 1 {
		t.Fatal("error lost or retried")
	}
	bad := marketsParams()
	bad.AssetType = ""
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.List(t.Context(), bad); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("invalid input accepted")
	}
	if _, err := c.List(ctx, marketsParams()); !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("cancellation lost or request sent")
	}
	for _, raw := range []string{"null", "{", `{"asset":"sui","marketType":"perpetual","page":1,"limit":100,"total":0,"sentAt":1792508400000001}`} {
		client, err := newMarketsClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
			return []byte(raw), nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.List(t.Context(), marketsParams()); err == nil {
			t.Fatal("malformed result accepted")
		}
	}
	var absent *MarketsClient
	if _, err := absent.List(t.Context(), marketsParams()); err == nil {
		t.Fatal("nil client accepted")
	}
	if _, err := c.List(nil, marketsParams()); err == nil {
		t.Fatal("nil context accepted")
	}
	var group *MarketHubClient
	if group.Markets() != nil {
		t.Fatal("nil group returned client")
	}
	if _, err := newMarketsClient(nil); err == nil {
		t.Fatal("nil caller accepted")
	}
}
