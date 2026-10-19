package httpclient

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/candles"
)

// TestCandlesMarketClient checks resolved-market identity and developing bars through the operation client.
//
// Version:
//   - 2026-10-19: Added.
func TestCandlesMarketClient(t *testing.T) {
	p := testParams()
	p.Asset, p.To, p.IncludeCurrent = "sui", p.From+60, true
	p.Market = &market.MarketTarget{Venue: "cetus", Chain: "sui", Network: "mainnet", PoolID: "0xpool"}
	original := *p.Market
	r := candles.Result{Definition: "venue", Method: "trade", Asset: p.Asset, MarketType: p.MarketType, Currency: "USD", Symbol: "SUI/USDC", Market: &market.MarketRef{Venue: "cetus", Chain: "sui", Network: "mainnet", PoolID: "0xpool"}, Interval: p.Interval, From: p.From, To: p.To, SentAt: 1791417600123456, Candles: []candles.Candle{{Time: p.From, Closed: false, Revision: "developing"}}}
	calls := 0
	client, err := newCandlesClient(callerFunc(func(_ context.Context, method jsonrpc.Method, params json.RawMessage) (json.RawMessage, error) {
		calls++
		var decoded candles.Params
		if err := json.Unmarshal(params, &decoded); err != nil {
			t.Fatal(err)
		}
		if method != jsonrpc.MethodMarketHubCandlesGet || !decoded.IncludeCurrent || decoded.Market == nil || *decoded.Market != original {
			t.Fatalf("selectors changed: %+v", decoded)
		}
		return json.Marshal(r)
	}))
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.Get(t.Context(), p)
	if err != nil || calls != 1 || *p.Market != original || got.SentAt != r.SentAt || got.Candles[0].Closed {
		t.Fatalf("invalid operation result: %+v %v", got, err)
	}
	for name, change := range map[string]func(*candles.Result){
		"wrong_pool":       func(v *candles.Result) { v.Market.PoolID = "0xother" },
		"wrong_network":    func(v *candles.Result) { v.Market.Network = "testnet" },
		"wrong_definition": func(v *candles.Result) { v.Definition = "composite" },
		"missing_market":   func(v *candles.Result) { v.Market = nil },
		"native_currency":  func(v *candles.Result) { v.Currency = "SUI" },
		"missing_symbol":   func(v *candles.Result) { v.Symbol = "" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := r
			ref := *r.Market
			bad.Market = &ref
			change(&bad)
			if err := validateCandlePage(&bad, p); err == nil {
				t.Fatal("incorrect series accepted")
			}
		})
	}
	p.IncludeCurrent = false
	if err := validateCandlePage(&r, p); err == nil {
		t.Fatal("unrequested developing candle accepted")
	}
}
