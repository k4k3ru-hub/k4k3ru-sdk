package scalping_test

import (
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping"
)

// ExampleParams builds a Spot request without signing or placing an order.
//
// Version:
//   - 2026-09-23: Added.
func ExampleParams() {
	// Replace placeholder asset and pool IDs with resolved provider metadata.
	market := rule.MarketRef{Venue: "cetus", Network: "mainnet", Chain: "sui", PoolID: "POOL_ID"}
	slippage, count := uint64(100), uint64(10)
	p := scalping.Params{
		MarketType: rule.MarketTypeSpot,
		BaseAsset:  rule.AssetRef{Chain: "sui", Network: "mainnet", AssetID: "0x2::sui::SUI"},
		QuoteAsset: rule.AssetRef{Chain: "sui", Network: "mainnet", AssetID: "USDC_COIN_TYPE"},
		Markets:    []rule.MarketRef{market},
		Conditions: scalping.Conditions{WindowMS: 30000, MaximumDataAgeMS: 2000, TradeCount: &scalping.CountRange{Minimum: &count}},
		ExecutionRule: rule.Rule{
			Open:  rule.OpenRule{Spot: &rule.SpotOpenRule{Amount: "1000000"}, MaximumSlippageBPS: &slippage, ExecutionTTLMS: 30000},
			Close: rule.CloseRule{TakeProfit: &rule.Trigger{Type: rule.TriggerTypePrice, Value: "2.1"}, Spot: &rule.SpotCloseRule{Markets: []rule.MarketRef{market}}, MaximumSlippageBPS: &slippage, ExecutionTTLMS: 30000},
		},
	}.Normalize()
	if err := p.Validate(); err != nil {
		fmt.Println(err)
		return
	}
	wire, err := json.Marshal(p)
	if err != nil {
		fmt.Println(err)
		return
	}
	var decoded scalping.Params
	if err := json.Unmarshal(wire, &decoded); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(jsonrpc.MethodTradeHubScalpingGet)
	fmt.Println(decoded.MarketType, decoded.ExecutionRule.Open.Spot.Amount)
	// Output:
	// TradeHub.Scalping.Get
	// spot 1000000
}
