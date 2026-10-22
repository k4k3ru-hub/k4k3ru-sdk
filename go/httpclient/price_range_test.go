package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/candles"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/change"
	pricerange "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/range"
)

func rangeResponse(p pricerange.Params) pricerange.Result {
	to := time.Date(2026, 10, 19, 12, 10, 30, 123000, time.UTC)
	r := pricerange.Result{Asset: p.Asset, MarketType: p.MarketType, Definition: "composite", Currency: "USD", Method: "volume_weighted_reference", SentAt: to.UnixMicro() + 1000}
	for _, period := range p.Normalize().Periods {
		r.Ranges = append(r.Ranges, pricerange.Range{Period: period, From: to.Add(-period.Duration()).Truncate(time.Minute).UnixMicro(), To: to.UnixMicro(), CoveragePercentage: "0", CoverageStatus: pricerange.CoverageStatusUnavailable})
	}
	return r
}

// TestPriceRangeComposition verifies constructor wiring and one request for each whole snapshot.
//
// Version:
//   - 2026-10-19: Added.
func TestPriceRangeComposition(t *testing.T) {
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
		if req.Method != jsonrpc.MethodMarketHubPriceRangeGet || r.Method != http.MethodPost {
			t.Fatal("wrong method")
		}
		if string(req.Method) != "MarketHub.Price.Range.Get" {
			t.Fatal("wrong wire method")
		}
		var p pricerange.Params
		if err := json.Unmarshal(req.Params, &p); err != nil {
			t.Fatal(err)
		}
		if p.AssetType != market.AssetClassCrypto || p.Asset != "sui" || p.MarketType != "spot" {
			t.Fatal("lost selectors")
		}
		if calls == 1 && len(p.Periods) != 6 {
			t.Fatal("omitted periods were not normalized")
		}
		if calls == 2 && !slices.Equal(p.Periods, []pricerange.Period{pricerange.Period24h, pricerange.Period5m}) {
			t.Fatal("explicit periods reordered")
		}
		return testResponse(t, req, rangeResponse(p), nil, 200), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 || client.MarketHub().Price().Range() == nil || client.MarketHub().Price().Change() == nil || client.MarketHub().Candles() == nil {
		t.Fatal("incomplete or eager composition")
	}
	p := pricerange.Params{AssetType: market.AssetClassCrypto, Asset: "sui", MarketType: market.MarketTypeSpot}
	r, err := client.MarketHub().Price().Range().Get(t.Context(), p)
	if err != nil || r == nil || len(r.Ranges) != 6 || len(p.Periods) != 0 || calls != 1 {
		t.Fatalf("invalid default snapshot: %v", err)
	}
	p.Periods = []pricerange.Period{pricerange.Period24h, pricerange.Period5m}
	if _, err := client.MarketHub().Price().Range().Get(t.Context(), p); err != nil || calls != 2 {
		t.Fatalf("invalid selected snapshot: %v", err)
	}
}

// TestAssetTypeRejectedBeforeHTTP prevents billable requests with missing or unsupported asset types.
//
// Version:
//   - 2026-10-19: Added.
func TestAssetTypeRejectedBeforeHTTP(t *testing.T) {
	calls := 0
	client, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected transport") })})
	if err != nil {
		t.Fatal(err)
	}
	for _, assetType := range []market.AssetClass{market.AssetClassUnknown, market.AssetClassFX, market.AssetClassStock, "Crypto"} {
		_, a := client.MarketHub().Price().Get(t.Context(), price.Params{AssetType: assetType, Asset: "sui", MarketType: "spot"})
		_, b := client.MarketHub().Price().Change().Get(t.Context(), change.Params{AssetType: assetType, Asset: "sui", MarketType: "spot"})
		_, c := client.MarketHub().Price().Range().Get(t.Context(), pricerange.Params{AssetType: assetType, Asset: "sui", MarketType: "spot"})
		_, d := client.MarketHub().Candles().Get(t.Context(), candles.Params{AssetType: assetType, Asset: "sui", MarketType: "spot", From: 1792326600, To: 1792326660, Interval: "1m"})
		for _, err := range []error{a, b, c, d} {
			if !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("invalid selector cause lost: %v", err)
			}
		}
	}
	if calls != 0 {
		t.Fatal("invalid asset type reached transport")
	}
}

// TestPriceRangeErrors preserves wrapped failures and rejects malformed or mismatched snapshots.
//
// Version:
//   - 2026-10-19: Added.
func TestPriceRangeErrors(t *testing.T) {
	p := pricerange.Params{AssetType: "crypto", Asset: "sui", MarketType: "spot", Periods: []pricerange.Period{pricerange.Period5m}}
	for name, mutate := range map[string]func(*pricerange.Result){
		"wrong asset":    func(r *pricerange.Result) { r.Asset = "btc" },
		"wrong market":   func(r *pricerange.Result) { r.MarketType = "perpetual" },
		"missing period": func(r *pricerange.Result) { r.Ranges = nil },
		"wrong period": func(r *pricerange.Result) {
			r.Ranges[0].Period = pricerange.Period15m
			r.Ranges[0].From -= 10 * time.Minute.Microseconds()
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := rangeResponse(p)
			mutate(&r)
			client, err := newPriceRangeClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(r)
			}))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Get(t.Context(), p); err == nil {
				t.Fatal("invalid snapshot accepted")
			}
		})
	}
	for _, raw := range []string{"null", "{", `{"ranges":null}`} {
		c, err := newPriceRangeClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
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
	c, err := newPriceRangeClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		calls++
		return nil, cause
	}))
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.Get(canceled, p); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("canceled request reached transport")
	}
	if _, err := c.Get(nil, p); err == nil || calls != 0 {
		t.Fatal("nil context accepted")
	}
	if _, err := c.Get(t.Context(), p); !errors.Is(err, cause) || calls != 1 {
		t.Fatal("transport error lost or retried")
	}
	var absent *PriceRangeClient
	if _, err := absent.Get(t.Context(), p); err == nil {
		t.Fatal("nil client accepted")
	}
	var group *PriceClient
	if group.Range() != nil {
		t.Fatal("nil group returned a client")
	}
	if _, err := newPriceRangeClient(nil); err == nil {
		t.Fatal("nil caller accepted")
	}
	if _, err := newPriceClient(c.caller, &PriceChangeClient{caller: c.caller}, nil); err == nil {
		t.Fatal("missing range client accepted")
	}
}
