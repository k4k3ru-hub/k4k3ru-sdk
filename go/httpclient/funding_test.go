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
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/funding"
)

func fundingParams() funding.Params {
	return funding.Params{AssetType: "crypto", Symbol: "SUI/USDC", Markets: []market.MarketTarget{{Venue: "hyperliquid", Network: "mainnet"}}}
}

func fundingResponse() funding.Result {
	rate, interval := "-0.000012", int32(60)
	return funding.Result{Symbol: "SUI/USDC", SentAt: 1792450800000001, Rates: []funding.Rate{{
		Market: market.MarketRef{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: "SUI"},
		Status: funding.StatusLive, Kind: funding.KindCurrentEstimate, Rate: &rate, IntervalMinutes: &interval,
	}}}
}

// TestFundingComposition verifies the composed transport and independent request normalization.
//
// Version:
//   - 2026-10-20: Added.
func TestFundingComposition(t *testing.T) {
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
		if req.Method != jsonrpc.MethodMarketHubFundingGet || string(req.Method) != "MarketHub.Funding.Get" || r.Method != http.MethodPost {
			t.Fatal("wrong method")
		}
		var p funding.Params
		if err := json.Unmarshal(req.Params, &p); err != nil || !reflect.DeepEqual(p, fundingParams()) {
			t.Fatalf("wrong selectors: %v", err)
		}
		return testResponse(t, req, fundingResponse(), nil, 200), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	g := client.MarketHub()
	if calls != 0 || g.Funding() == nil || g.Quote() == nil || g.Price() == nil || g.Volume() == nil || g.Candles() == nil {
		t.Fatal("incomplete or eager composition")
	}
	p := fundingParams()
	p.Symbol, p.Markets[0].Network = " sui/usdc ", " Mainnet "
	before := p
	before.Markets = append([]market.MarketTarget(nil), p.Markets...)
	r, err := g.Funding().Get(t.Context(), p)
	if err != nil || r == nil || *r.Rates[0].Rate != "-0.000012" || calls != 1 || !reflect.DeepEqual(before, p) {
		t.Fatalf("invalid result, retry or mutation: %v", err)
	}
}

// TestFundingResponseBinding rejects other instruments and missing requested scopes.
//
// Version:
//   - 2026-10-20: Added.
func TestFundingResponseBinding(t *testing.T) {
	for name, change := range map[string]func(*funding.Result){
		"symbol":     func(r *funding.Result) { r.Symbol = "BTC/USDC" },
		"venue":      func(r *funding.Result) { r.Rates[0].Market.Venue = "bluefin" },
		"network":    func(r *funding.Result) { r.Rates[0].Market.Network = "testnet" },
		"instrument": func(r *funding.Result) { r.Rates[0].Market.VenueSymbol = "BTC" },
		"chain":      func(r *funding.Result) { r.Rates[0].Market.Chain = "ethereum" },
		"empty":      func(r *funding.Result) { r.Rates = nil },
	} {
		t.Run(name, func(t *testing.T) {
			result := fundingResponse()
			result.Rates[0].Market.Chain = "sui"
			change(&result)
			c, err := newFundingClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(result)
			}))
			if err != nil {
				t.Fatal(err)
			}
			p := fundingParams()
			p.Markets[0].VenueSymbol = "SUI"
			p.Markets[0].Chain = "sui"
			if _, err := c.Get(t.Context(), p); err == nil {
				t.Fatal("mismatched response accepted")
			}
		})
	}
	result := fundingResponse()
	c, err := newFundingClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		return json.Marshal(result)
	}))
	if err != nil {
		t.Fatal(err)
	}
	p := fundingParams()
	p.Markets = append(p.Markets, market.MarketTarget{Venue: "bluefin", Network: "mainnet"})
	if _, err := c.Get(t.Context(), p); err == nil {
		t.Fatal("missing requested scope accepted")
	}
}

// TestFundingFailures preserves causes and avoids requests for invalid or canceled input.
//
// Version:
//   - 2026-10-20: Added.
func TestFundingFailures(t *testing.T) {
	calls := 0
	cause := apperror.OutOfTicks()
	c, err := newFundingClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		calls++
		return nil, cause
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(t.Context(), fundingParams()); !errors.Is(err, cause) || calls != 1 {
		t.Fatal("error lost or retried")
	}
	bad := fundingParams()
	bad.AssetType = ""
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.Get(t.Context(), bad); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("invalid input accepted")
	}
	if _, err := c.Get(ctx, fundingParams()); !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("cancellation lost or request sent")
	}
	for _, raw := range []string{"null", "{", `{"rates":[]}`} {
		client, err := newFundingClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
			return []byte(raw), nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.Get(t.Context(), fundingParams()); err == nil {
			t.Fatal("malformed result accepted")
		}
	}
	var absent *FundingClient
	if _, err := absent.Get(t.Context(), fundingParams()); err == nil {
		t.Fatal("nil client accepted")
	}
	if _, err := c.Get(nil, fundingParams()); err == nil {
		t.Fatal("nil context accepted")
	}
	var group *MarketHubClient
	if group.Funding() != nil {
		t.Fatal("nil group returned client")
	}
	if _, err := newFundingClient(nil); err == nil {
		t.Fatal("nil caller accepted")
	}
}
