package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/openinterest"
)

func openinterestParams() openinterest.Params {
	return openinterest.Params{AssetType: "crypto", Symbol: "SUI/USDC", Markets: []market.MarketTarget{{Venue: "hyperliquid", Network: "mainnet"}}}
}

func openinterestResponse() openinterest.Result {
	notional := "1851.75"
	return openinterest.Result{Symbol: "SUI/USDC", SentAt: 1792450800000001, Interests: []openinterest.Interest{{
		Market: market.MarketRef{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: "SUI"},
		Status: openinterest.StatusLive, Quantity: &market.Quantity{Amount: "12345", Decimals: 1}, NotionalValue: &notional, NotionalCurrency: "USDC", Valuation: openinterest.ValuationMark,
	}}}
}

// TestOpenInterestComposition verifies the composed transport and independent request normalization.
//
// Version:
//   - 2026-10-20: Added.
func TestOpenInterestComposition(t *testing.T) {
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
		if req.Method != jsonrpc.MethodMarketHubOpenInterestGet || string(req.Method) != "MarketHub.OpenInterest.Get" || r.Method != http.MethodPost {
			t.Fatal("wrong method")
		}
		var p openinterest.Params
		if err := json.Unmarshal(req.Params, &p); err != nil || !reflect.DeepEqual(p, openinterestParams()) {
			t.Fatalf("wrong selectors: %v", err)
		}
		return testResponse(t, req, openinterestResponse(), nil, 200), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	g := client.MarketHub()
	if calls != 0 || g.OpenInterest() == nil || g.Quote() == nil || g.Price() == nil || g.Volume() == nil || g.Candles() == nil {
		t.Fatal("incomplete or eager composition")
	}
	p := openinterestParams()
	p.Symbol, p.Markets[0].Network = " sui/usdc ", " Mainnet "
	before := p
	before.Markets = append([]market.MarketTarget(nil), p.Markets...)
	r, err := g.OpenInterest().Get(t.Context(), p)
	if err != nil || r == nil || *r.Interests[0].NotionalValue != "1851.75" || calls != 1 || !reflect.DeepEqual(before, p) {
		t.Fatalf("invalid result, retry or mutation: %v", err)
	}
}

// TestOpenInterestResponseBinding rejects other instruments and missing requested scopes.
//
// Version:
//   - 2026-10-20: Added.
func TestOpenInterestResponseBinding(t *testing.T) {
	for name, change := range map[string]func(*openinterest.Result){
		"symbol":     func(r *openinterest.Result) { r.Symbol = "BTC/USDC" },
		"venue":      func(r *openinterest.Result) { r.Interests[0].Market.Venue = "bluefin" },
		"network":    func(r *openinterest.Result) { r.Interests[0].Market.Network = "testnet" },
		"instrument": func(r *openinterest.Result) { r.Interests[0].Market.VenueSymbol = "BTC" },
		"chain":      func(r *openinterest.Result) { r.Interests[0].Market.Chain = "ethereum" },
		"empty":      func(r *openinterest.Result) { r.Interests = nil },
	} {
		t.Run(name, func(t *testing.T) {
			result := openinterestResponse()
			result.Interests[0].Market.Chain = "sui"
			change(&result)
			c, err := newOpenInterestClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(result)
			}))
			if err != nil {
				t.Fatal(err)
			}
			p := openinterestParams()
			p.Markets[0].VenueSymbol = "SUI"
			p.Markets[0].Chain = "sui"
			if _, err := c.Get(t.Context(), p); err == nil {
				t.Fatal("mismatched response accepted")
			}
		})
	}
	result := openinterestResponse()
	c, err := newOpenInterestClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(result)
	}))
	if err != nil {
		t.Fatal(err)
	}
	p := openinterestParams()
	p.Markets = append(p.Markets, market.MarketTarget{Venue: "bluefin", Network: "mainnet"})
	if _, err := c.Get(t.Context(), p); err == nil {
		t.Fatal("missing requested scope accepted")
	}
}

// TestOpenInterestFailures preserves causes and avoids requests for invalid or canceled input.
//
// Version:
//   - 2026-10-20: Added.
func TestOpenInterestFailures(t *testing.T) {
	calls := 0
	cause := apperror.OutOfTicks()
	c, err := newOpenInterestClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		calls++
		return nil, cause
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(t.Context(), openinterestParams()); !errors.Is(err, cause) || calls != 1 {
		t.Fatal("error lost or retried")
	}
	bad := openinterestParams()
	bad.AssetType = ""
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.Get(t.Context(), bad); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("invalid input accepted")
	}
	if _, err := c.Get(ctx, openinterestParams()); !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("cancellation lost or request sent")
	}
	for _, raw := range []string{"null", "{", `{"interests":[]}`} {
		client, err := newOpenInterestClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
			return []byte(raw), nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.Get(t.Context(), openinterestParams()); err == nil {
			t.Fatal("malformed result accepted")
		}
	}
	var absent *OpenInterestClient
	if _, err := absent.Get(t.Context(), openinterestParams()); err == nil {
		t.Fatal("nil client accepted")
	}
	if _, err := c.Get(nil, openinterestParams()); err == nil {
		t.Fatal("nil context accepted")
	}
	var group *MarketHubClient
	if group.OpenInterest() != nil {
		t.Fatal("nil group returned client")
	}
	if _, err := newOpenInterestClient(nil); err == nil {
		t.Fatal("nil caller accepted")
	}
}
