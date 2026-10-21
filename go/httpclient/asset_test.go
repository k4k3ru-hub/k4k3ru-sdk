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

func assetParams() asset.Params {
	return asset.Params{AssetType: "crypto", Asset: "sui"}
}

func assetResponse() asset.Result {
	return asset.Result{AssetType: "crypto", Asset: "sui", Symbol: "SUI", Name: "Sui", MarketTypes: []market.MarketType{"spot", "perpetual"}, SentAt: 1792548000123456}
}

// TestAssetComposition verifies lazy composition, method routing and one immutable request.
//
// Version:
//   - 2026-10-21: Added.
func TestAssetComposition(t *testing.T) {
	calls := 0
	client, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var req jsonrpc.Request
		err := json.NewDecoder(r.Body).Decode(&req)
		closeErr := r.Body.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("request decode failed: %v %v", err, closeErr)
		}
		if req.Method != jsonrpc.MethodMarketHubAssetGet || string(req.Method) != "MarketHub.Asset.Get" || r.Method != http.MethodPost {
			t.Fatal("wrong method")
		}
		var p asset.Params
		if err := json.Unmarshal(req.Params, &p); err != nil || p != assetParams() {
			t.Fatalf("wrong selectors: %v", err)
		}
		return testResponse(t, req, assetResponse(), nil, 200), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	g := client.MarketHub()
	if calls != 0 || g.Asset() == nil || g.Price() == nil || g.Candles() == nil || g.Volume() == nil || g.Quote() == nil || g.Funding() == nil || g.OpenInterest() == nil {
		t.Fatal("incomplete or eager composition")
	}
	p := assetParams()
	before := p
	result, err := g.Asset().Get(t.Context(), p)
	if err != nil || result == nil || result.Symbol != "SUI" || calls != 1 || p != before {
		t.Fatalf("wrong response, retry or caller mutation: %v", err)
	}
}

// TestAssetFailures preserves error causes and sends no invalid or canceled requests.
//
// Version:
//   - 2026-10-21: Added.
func TestAssetFailures(t *testing.T) {
	calls := 0
	cause := apperror.OutOfTicks()
	c, err := newAssetClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		calls++
		return nil, cause
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(t.Context(), assetParams()); !errors.Is(err, cause) || calls != 1 {
		t.Fatal("error lost or retried")
	}
	bad := assetParams()
	bad.AssetType = ""
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.Get(t.Context(), bad); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("invalid input accepted")
	}
	if _, err := c.Get(ctx, assetParams()); !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("cancellation lost or request sent")
	}
	for _, raw := range []string{"null", "{", `{}`, `{"assetType":"crypto","asset":"btc","symbol":"BTC","name":"Bitcoin","marketTypes":["spot"],"sentAt":1792548000123456}`} {
		client, err := newAssetClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
			return []byte(raw), nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.Get(t.Context(), assetParams()); err == nil {
			t.Fatal("malformed result accepted")
		}
	}
	var absent *AssetClient
	if _, err := absent.Get(t.Context(), assetParams()); err == nil {
		t.Fatal("nil client accepted")
	}
	if _, err := c.Get(nil, assetParams()); err == nil {
		t.Fatal("nil context accepted")
	}
	var group *MarketHubClient
	if group.Asset() != nil {
		t.Fatal("nil group returned client")
	}
	if _, err := newAssetClient(nil); err == nil {
		t.Fatal("nil caller accepted")
	}
}
