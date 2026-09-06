package spread

import (
	k4k3ruSDKMarketHubSpread "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/spread"
	k4k3ruSDKTradeHubExecution "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
)

type PreparedRoute struct {
	Route     k4k3ruSDKMarketHubSpread.Route    `json:"route"`
	Execution k4k3ruSDKTradeHubExecution.Result `json:"execution"`
}

type Result struct {
	Opportunity    k4k3ruSDKMarketHubSpread.Result `json:"opportunity"`
	PreparedRoutes []PreparedRoute                 `json:"preparedRoutes"`
}
