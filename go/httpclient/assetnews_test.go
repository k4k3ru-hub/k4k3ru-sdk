package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/assetnews"
)

// TestAssetNewsComposition verifies lazy composition, one request, and preserved caller parameters.
//
// Version:
//   - 2026-11-02: Added.
func TestAssetNewsComposition(t *testing.T) {
	calls := 0
	p := assetnews.Params{AssetType: "crypto", Asset: "sui"}
	before := p
	c, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var req jsonrpc.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if err := r.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if req.Method != jsonrpc.MethodMarketHubAssetNewsList || req.Auth != nil {
			t.Fatal("wrong method or unexpected auth")
		}
		result := assetnews.Result{AssetType: "crypto", Asset: "sui", Scope: "asset_context", Coverage: "latest_source_windows", Snapshot: "fixture", Page: 1, Limit: 20, Items: []assetnews.Item{}, Sources: []assetnews.Source{}, SentAt: 1}
		return testResponse(t, req, result, nil, 200), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 || c.MarketHub().AssetNews() == nil {
		t.Fatal("eager or missing composition")
	}
	if _, err := c.MarketHub().AssetNews().List(t.Context(), p); err != nil || calls != 1 || p != before {
		t.Fatal(err, calls)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.MarketHub().AssetNews().List(ctx, p); !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatal("canceled request sent")
	}
}
