package httpclient

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price"
)

// TestPriceComposition verifies operation composition, exact selectors and one HTTP call.
//
// Version:
//   - 2026-10-19: Update required crypto asset selectors.
//   - 2026-10-18: Added.
func TestPriceComposition(t *testing.T) {
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
		if req.Method != jsonrpc.MethodMarketHubPriceGet || req.Auth != nil || string(req.Params) != `{"assetType":"crypto","asset":"sui","marketType":"perpetual"}` {
			t.Fatalf("wrong request: %+v", req)
		}
		return testResponse(t, req, price.Result{Asset: "sui", MarketType: "perpetual", Definition: "composite", Currency: "USD", Method: "volume_weighted_reference", Interval: "60s", Status: price.StatusUnavailable, SentAt: 1792300000123456}, nil, 200), nil
	})})
	if err != nil || client.MarketHub().Candles() == nil || client.MarketHub().Price() == nil || calls != 0 {
		t.Fatalf("composition failed: %v", err)
	}
	r, err := client.MarketHub().Price().Get(t.Context(), price.Params{AssetType: "crypto", Asset: "sui", MarketType: "perpetual"})
	if err != nil || calls != 1 || r.Price != nil || r.SentAt != 1792300000123456 {
		t.Fatalf("unexpected result: %+v %v", r, err)
	}
	if _, err := client.MarketHub().Price().Get(t.Context(), price.Params{AssetType: "crypto", Asset: "sui", MarketType: "perp"}); err == nil || calls != 1 {
		t.Fatal("invalid selectors reached transport")
	}
}
