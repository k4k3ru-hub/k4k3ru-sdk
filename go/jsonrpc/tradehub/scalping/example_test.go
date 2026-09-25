package scalping_test

import (
	"encoding/json"
	"fmt"

	sdkMarket "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping"
)

// ExampleSubscribeParams builds a Spot request without signing or placing an order.
//
// Version:
//   - 2026-09-25: Use SDK finance market types and canonical perpetual values.
//   - 2026-09-24: Use the default observation window constant for Go requests.
//   - 2026-09-24: Encode an idempotent subscription start.
//   - 2026-09-23: Added.
func ExampleSubscribeParams() {
	// Replace placeholder asset and pool IDs with resolved provider metadata.
	market := rule.MarketRef{Venue: "cetus", Network: "mainnet", Chain: "sui", PoolID: "POOL_ID"}
	slippage, count := uint64(100), uint64(10)
	p := scalping.Params{
		MarketType: sdkMarket.MarketTypeSpot,
		BaseAsset:  rule.AssetRef{Chain: "sui", Network: "mainnet", AssetID: "0x2::sui::SUI"},
		QuoteAsset: rule.AssetRef{Chain: "sui", Network: "mainnet", AssetID: "USDC_COIN_TYPE"},
		Markets:    []rule.MarketRef{market},
		Conditions: scalping.Conditions{WindowMS: scalping.DefaultWindowMS, MaximumDataAgeMS: 2000, TradeCount: &scalping.CountRange{Minimum: &count}},
		ExecutionRule: rule.Rule{
			Open:  rule.OpenRule{Spot: &rule.SpotOpenRule{Amount: "1000000"}, MaximumSlippageBPS: &slippage, ExecutionTTLMS: 30000},
			Close: rule.CloseRule{TakeProfit: &rule.Trigger{Type: rule.TriggerTypePrice, Value: "2.1"}, Spot: &rule.SpotCloseRule{Markets: []rule.MarketRef{market}}, MaximumSlippageBPS: &slippage, ExecutionTTLMS: 30000},
		},
	}.Normalize()
	if err := p.Validate(); err != nil {
		fmt.Println(err)
		return
	}
	wire, err := json.Marshal(scalping.SubscribeParams{IdempotencyKey: "start-one", Params: &p})
	if err != nil {
		fmt.Println(err)
		return
	}
	var decoded scalping.SubscribeParams
	if err := json.Unmarshal(wire, &decoded); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(jsonrpc.MethodTradeHubScalpingSubscribe)
	fmt.Println(decoded.Params.MarketType, decoded.Params.ExecutionRule.Open.Spot.Amount)
	fmt.Println(decoded.Params.Conditions.WindowMS)
	// Output:
	// TradeHub.Scalping.Subscribe
	// spot 1000000
	// 60000
}
