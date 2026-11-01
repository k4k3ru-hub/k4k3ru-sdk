package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/asset/overview"
	"net/http"
	"testing"
)

// TestAssetOverviewComposition verifies the asset group composes the new operation and preserves the old client.
//
// Version:
//   - 2026-11-01: Added.
func TestAssetOverviewComposition(t *testing.T) {
	root, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("constructor invoked HTTP")
		return nil, errors.New("unexpected constructor HTTP")
	})})
	if err != nil || root.MarketHub() == nil || root.MarketHub().Asset() == nil || root.MarketHub().Asset().Overview() == nil {
		t.Fatal("root constructor did not compose Overview", err)
	}

	calls := 0
	c, err := newAssetClient(callerFunc(func(_ context.Context, m jsonrpc.Method, p json.RawMessage) (json.RawMessage, error) {
		calls++
		if m != jsonrpc.MethodMarketHubAssetOverviewGet {
			t.Fatalf("wrong method: %s", m)
		}
		if string(p) != `{"assetType":"crypto","asset":"sui"}` {
			t.Fatalf("changed parameters: %s", p)
		}
		return json.RawMessage(`null`), nil
	}))
	if err != nil || c.Overview() == nil {
		t.Fatal("missing composed overview", err)
	}
	p := overview.Params{AssetType: "crypto", Asset: "sui"}
	if _, err := c.Overview().Get(t.Context(), p); err == nil || calls != 1 || p.Asset != "sui" {
		t.Fatal("null response or request mutation")
	}
	if _, err := c.Overview().Get(t.Context(), overview.Params{AssetType: "crypto", Asset: "SUI"}); err == nil || calls != 1 {
		t.Fatal("invalid selector reached transport")
	}
	if (*AssetClient)(nil).Overview() != nil {
		t.Fatal("nil group returned operation")
	}
}
