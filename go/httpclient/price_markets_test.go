package httpclient

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price"
)

// TestPriceMarketsHTTP verifies one composed HTTP call and rejects incomplete market responses.
//
// Version:
//   - 2026-10-21: Added.
func TestPriceMarketsHTTP(t *testing.T) {
	q := price.Params{AssetType: "crypto", Asset: "sui", MarketType: "perpetual", Markets: []market.MarketRef{{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: "SUI"}}}
	calls := 0
	mode := "valid"
	client, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var req jsonrpc.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if err := r.Body.Close(); err != nil {
			t.Fatal(err)
		}
		var got price.Params
		if err := json.Unmarshal(req.Params, &got); err != nil {
			t.Fatal(err)
		}
		if req.Method != jsonrpc.MethodMarketHubPriceGet || len(got.Markets) != 1 || got.Markets[0] != q.Markets[0] {
			t.Fatal("wrong batch", req)
		}
		result := price.Result{Asset: "sui", MarketType: "perpetual", Definition: "composite", Currency: "USD", Method: "volume_weighted_reference", Interval: "60s", Status: price.StatusUnavailable, SentAt: 1792508400000001,
			Markets: []price.MarketPrice{{Market: q.Markets[0], Symbol: "SUI/USDC", Currency: "USD", Method: "last_trade", Status: price.StatusUnavailable}}}
		if mode == "missing" {
			result.Markets = nil
		}
		return testResponse(t, req, result, nil, 200), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.MarketHub().Price().Get(t.Context(), q); err != nil || calls != 1 {
		t.Fatal("batch failed", err, calls)
	}
	mode = "missing"
	if _, err := client.MarketHub().Price().Get(t.Context(), q); err == nil || calls != 2 {
		t.Fatal("missing market accepted or retried", err, calls)
	}
}
