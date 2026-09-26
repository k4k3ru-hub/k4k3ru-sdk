package scalping_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
)

// TestScalpingResultWireContract verifies flat snapshots, omission and stable status names.
//
// Version:
//   - 2026-09-26: Added.
func TestScalpingResultWireContract(t *testing.T) {
	volatility := "140.718928426497526047"
	zero := uint64(0)
	value := scalping.Result{
		EvaluatedAt: 1790380800000,
		Metrics:     &scalping.Metrics{TradeCount: &zero, RealizedVolatilityBPS: &volatility},
		Buy:         []scalping.MarketPrice{{Market: market.MarketRef{Venue: market.Hyperliquid, Network: "mainnet", VenueSymbol: "SUI"}, Status: scalping.PriceStatusUnavailable}},
		Sell:        []scalping.MarketPrice{},
	}
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{`"tradeCount":0`, `"realizedVolatilityBps":"140.718928426497526047"`, `"status":"unavailable"`, `"sell":[]`} {
		if !strings.Contains(string(raw), part) {
			t.Fatalf("missing contract field: %s", part)
		}
	}
	for _, part := range []string{`"ohlc"`, `"price"`, `"issues"`, `"groups"`, `"trend"`} {
		if strings.Contains(string(raw), part) {
			t.Fatalf("unexpected field: %s", part)
		}
	}
	var decoded scalping.Result
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Metrics.RealizedVolatilityBPS == nil || *decoded.Metrics.RealizedVolatilityBPS != volatility {
		t.Fatal("volatility lost")
	}
}
