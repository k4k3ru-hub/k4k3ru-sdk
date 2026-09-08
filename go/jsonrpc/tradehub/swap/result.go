package swap

import (
	k4k3ruSDKMarketHubArbitrage "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/arbitrage"
	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

type Result struct {
	Chain                  k4k3ruOnchainCore.Chain                     `json:"chain"`
	Network                k4k3ruOnchainCore.Network                   `json:"network"`
	Venue                  k4k3ruSDKMarketHubArbitrage.Venue           `json:"venue"`
	PoolID                 string                                      `json:"poolId"`
	TokenInAssetID         string                                      `json:"tokenInAssetId"`
	TokenOutAssetID        string                                      `json:"tokenOutAssetId"`
	Kind                   Kind                                        `json:"kind"`
	AmountIn               string                                      `json:"amountIn"`
	AmountOut              string                                      `json:"amountOut"`
	RecommendedAmountLimit string                                      `json:"recommendedAmountLimit"`
	MaximumSlippageBPS     uint64                                      `json:"maximumSlippageBps"`
	GasEstimate            string                                      `json:"gasEstimate,omitempty"`
	StateReference         *k4k3ruSDKMarketHubArbitrage.StateReference `json:"stateReference,omitempty"`
	QuotedAt               int64                                       `json:"quotedAt"`
}
