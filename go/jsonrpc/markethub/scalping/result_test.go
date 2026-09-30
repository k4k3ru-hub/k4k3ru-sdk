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
//   - 2026-10-01: Omit unknown price-window evaluation times.
//   - 2026-09-28: Verify fee-inclusive result fields.
//   - 2026-09-26: Omit unknown fees and last trade time.
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
	for _, part := range []string{`"ohlc"`, `"netPrice"`, `"issues"`, `"groups"`, `"trend"`, `"fees"`, `"lastTradeAt"`, `"priceEvaluatedAt"`} {
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

// TestScalpingFeeWireContract preserves charged-token identity, units and known zero fees.
//
// Version:
//   - 2026-09-28: Verify fee-inclusive result fields.
//   - 2026-09-26: Added.
func TestScalpingFeeWireContract(t *testing.T) {
	const raw = `{
		"evaluatedAt":1790380800000,
		"buy":[{"market":{"venue":"cetus","chain":"sui","network":"testnet","poolId":"0x1"},"status":"vwap","netPrice":"2.5","netReceiveQuantity":{"amount":"40000000000","decimals":9},
			"observedAt":1790380799000,"lastTradeAt":1790380620000,
			"fees":{"swap":{"token":{"assetId":"0x3::usdc::USDC","symbol":"USDC"},"quantity":{"amount":"3000","decimals":6}}}}],
		"sell":[{"market":{"venue":"hyperliquid","network":"mainnet","venueSymbol":"@1"},"status":"vwap","netPrice":"2.4","netReceiveQuantity":{"amount":"2400000","decimals":6},
			"fees":{"taker":{"token":{"assetId":"USDC","symbol":"USDC"},"quantity":{"amount":"0","decimals":0}}}}]
	}`
	var value scalping.Result
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatal(err)
	}
	buy, sell := value.Buy[0], value.Sell[0]
	if buy.NetReceiveQuantity == nil || buy.NetReceiveQuantity.Decimals != 9 || buy.NetReceiveQuantity.Amount != "40000000000" || sell.NetReceiveQuantity == nil || sell.NetReceiveQuantity.Decimals != 6 || sell.NetReceiveQuantity.Amount != "2400000" {
		t.Fatal("directional receive quantities lost")
	}
	if buy.Fees.Swap.Token.AssetID != "0x3::usdc::USDC" || buy.Fees.Swap.Token.Symbol != "USDC" || buy.Fees.Swap.Quantity.Amount != "3000" || buy.Fees.Swap.Quantity.Decimals != 6 {
		t.Fatal("charged asset or units lost")
	}
	if buy.Fees.Taker != nil || sell.Fees.Swap != nil || sell.Fees.Taker.Quantity.Amount != "0" || sell.Fees.Taker.Quantity.Decimals != 0 {
		t.Fatal("unknown fee confused with known zero")
	}
	if buy.LastTradeAt == nil || *buy.LastTradeAt != 1790380620000 || *buy.LastTradeAt == *buy.ObservedAt || sell.LastTradeAt != nil {
		t.Fatal("trade and input timestamps conflated")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{`"baseQuantity"`, `"quoteQuantity"`, `"token0"`, `"token1"`, `"issues"`} {
		if strings.Contains(string(encoded), part) {
			t.Fatalf("unexpected fee representation: %s", part)
		}
	}
	var roundTrip scalping.Result
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if *roundTrip.Buy[0].Fees.Swap != *buy.Fees.Swap || *roundTrip.Sell[0].Fees.Taker != *sell.Fees.Taker {
		t.Fatal("fee changed during round trip")
	}
	for _, bad := range []string{
		strings.Replace(raw, `"amount":"3000","decimals":6`, `"amount":"3000"`, 1),
		strings.Replace(raw, `"amount":"3000"`, `"amount":"-1"`, 1),
	} {
		if err := json.Unmarshal([]byte(bad), &roundTrip); err == nil {
			t.Fatal("invalid fee quantity accepted")
		}
	}
}

// TestPriceEvaluationTimeWire separates common price coverage from calculation time.
//
// Version:
//   - 2026-10-01: Added.
func TestPriceEvaluationTimeWire(t *testing.T) {
	const raw = `{"evaluatedAt":1790380800000,"priceEvaluatedAt":1790380799750,"buy":[],"sell":[]}`
	var result scalping.Result
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatal(err)
	}
	if result.PriceEvaluatedAt == nil || *result.PriceEvaluatedAt != 1790380799750 || result.EvaluatedAt != 1790380800000 {
		t.Fatal("evaluation times conflated")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"priceEvaluatedAt":1790380799750`) {
		t.Fatal("price timestamp lost")
	}
}
