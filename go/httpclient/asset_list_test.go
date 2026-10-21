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
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/asset"
)

func assetListResponse() asset.ListResult {
	return asset.ListResult{AssetType: "crypto", Assets: []asset.ListedAsset{{Asset: "sui", Symbol: "SUI", Name: "Sui", MarketTypes: []market.MarketType{"spot", "perpetual"}}}, Page: 1, Limit: 100, Total: 1, SentAt: 1792548000123456}
}

// TestAssetListComposition checks the existing composition root and immutable one-page transport.
//
// Version:
//   - 2026-10-21: Added.
func TestAssetListComposition(t *testing.T) {
	calls := 0
	p := asset.ListParams{AssetType: "crypto", MarketType: "spot", Search: "SuI"}
	before := p
	c, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var req jsonrpc.Request
		err := json.NewDecoder(r.Body).Decode(&req)
		closeErr := r.Body.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
		if req.Method != jsonrpc.MethodMarketHubAssetList || r.Method != http.MethodPost {
			t.Fatal("wrong method")
		}
		var got asset.ListParams
		if err := json.Unmarshal(req.Params, &got); err != nil || got != p.Normalize() {
			t.Fatal(got, err)
		}
		return testResponse(t, req, assetListResponse(), nil, 200), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 || c.MarketHub().Asset() == nil {
		t.Fatal("eager or missing composition")
	}
	res, err := c.MarketHub().Asset().List(t.Context(), p)
	if err != nil || res == nil || len(res.Assets) != 1 || calls != 1 || p != before {
		t.Fatal(res, err, calls)
	}
}

// TestAssetListFailures preserves causes and rejects malformed or out-of-scope responses.
//
// Version:
//   - 2026-10-21: Added.
func TestAssetListFailures(t *testing.T) {
	p := asset.ListParams{AssetType: "crypto", MarketType: "spot", Search: "sui"}
	calls := 0
	cause := apperror.OutOfTicks()
	c, err := newAssetClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		calls++
		return nil, cause
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.List(t.Context(), p); !errors.Is(err, cause) || calls != 1 {
		t.Fatal(err, calls)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.List(ctx, p); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := c.List(t.Context(), asset.ListParams{}); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal(err)
	}
	if _, err := c.List(nil, p); err == nil {
		t.Fatal("nil context")
	}
	if calls != 1 {
		t.Fatal("invalid request sent")
	}
	var absent *AssetClient
	if _, err := absent.List(t.Context(), p); err == nil {
		t.Fatal("nil client")
	}
	for _, mutate := range []func(*asset.ListResult){
		func(r *asset.ListResult) { r.Page = 2; r.Assets = []asset.ListedAsset{} },
		func(r *asset.ListResult) { r.Limit = 25 },
		func(r *asset.ListResult) {
			r.Assets[0].Asset = "btc"
			r.Assets[0].Symbol = "BTC"
			r.Assets[0].Name = "Bitcoin"
		},
		func(r *asset.ListResult) { r.Assets[0].MarketTypes = []market.MarketType{"perpetual"} },
	} {
		r := assetListResponse()
		mutate(&r)
		body, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		client, err := newAssetClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) { return body, nil }))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.List(t.Context(), p); err == nil {
			t.Fatal("out-of-scope result accepted")
		}
	}
	for _, raw := range []string{"null", "{", "{}"} {
		client, err := newAssetClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
			return []byte(raw), nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.List(t.Context(), p); err == nil {
			t.Fatal("invalid result accepted")
		}
	}
}
