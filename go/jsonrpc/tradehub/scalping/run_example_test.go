package scalping_test

import (
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping"
)

// ExampleRunParams constructs a request without sending an order or opening a stream.
//
// Version:
//   - 2026-10-01: Added.
func ExampleRunParams() {
	minimum, holding := "20", uint64(60000)
	request := scalping.RunParams{
		IdempotencyKey: "example-001",
		Params: &scalping.RunConfiguration{
			MarketType: market.MarketTypeSpot, Symbol: market.SUIUSDC, TestMode: true,
			Observation: scalping.Observation{Markets: []scalping.MarketSelector{{Venue: "cetus", Chain: "sui"}}},
			ExecutionRule: scalping.ExecutionRule{
				Markets: []scalping.ExecutionMarket{{
					MarketSelector: scalping.MarketSelector{Venue: "cetus", Chain: "sui"},
					AccountAddress: "YOUR_SUI_WALLET_ADDRESS",
				}},
				Entry: scalping.EntryRule{
					Side:            scalping.SideBuy,
					MaximumQuantity: market.Quantity{Amount: "100", Decimals: 1}, // At most 10 USDC.
					Condition:       scalping.Condition{PriceChangeBPS: &scalping.DecimalRange{Minimum: &minimum}},
				},
				Exit: scalping.ExitRule{MaximumHoldingMS: &holding},
			},
		},
	}.Normalize()
	if err := request.Validate(); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(jsonrpc.MethodTradeHubScalpingRun, request.Params.TestMode)
	fmt.Println(*request.Params.ExecutionRule.ExecutionTTLMS)
	resume, err := json.Marshal(scalping.RunParams{ExecutionID: "saved-execution-id"})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(resume))
	// Output:
	// TradeHub.Scalping.Run true
	// 30000
	// {"executionId":"saved-execution-id"}
}
