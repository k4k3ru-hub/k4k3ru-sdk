package execution

import (
	k4k3ruSDKMarketHubArbitrage "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/arbitrage"
	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

type Simulation struct {
	Success            bool                                        `json:"success"`
	GrossProfit        string                                      `json:"grossProfit"`
	ExecutionCost      string                                      `json:"executionCost"`
	ExecutionCostAsset string                                      `json:"executionCostAsset"`
	NetProfit          string                                      `json:"netProfit"`
	ProfitAsset        string                                      `json:"profitAsset"`
	SlippageBPS        uint64                                      `json:"slippageBps"`
	StateReference     *k4k3ruSDKMarketHubArbitrage.StateReference `json:"stateReference,omitempty"`
}

type EVMUnsignedTransaction struct {
	ChainID            string `json:"chainId"`
	Nonce              uint64 `json:"nonce"`
	To                 string `json:"to"`
	Value              string `json:"value"`
	Data               string `json:"data"`
	GasLimit           uint64 `json:"gasLimit"`
	MaximumFeePerGas   string `json:"maximumFeePerGas"`
	MaximumPriorityFee string `json:"maximumPriorityFeePerGas"`
}

type SigningPayload struct {
	ChainFamily         ChainFamily             `json:"chainFamily"`
	Encoding            string                  `json:"encoding,omitempty"`
	TransactionBytes    string                  `json:"transactionBytes,omitempty"`
	UnsignedTransaction *EVMUnsignedTransaction `json:"unsignedTransaction,omitempty"`
	Digest              string                  `json:"digest"`
}

type Manifest struct {
	Chain            k4k3ruOnchainCore.Chain             `json:"chain"`
	Network          k4k3ruOnchainCore.Network           `json:"network"`
	Signer           string                              `json:"signer"`
	InputAsset       string                              `json:"inputAsset"`
	MaximumAmountIn  string                              `json:"maximumAmountIn"`
	MinimumAmountOut string                              `json:"minimumAmountOut"`
	Venues           []k4k3ruSDKMarketHubArbitrage.Venue `json:"venues"`
}

type SubmitParams struct {
	ExecutionID   string `json:"executionId"`
	PayloadDigest string `json:"payloadDigest"`
}

type Result struct {
	ExecutionID     string          `json:"executionId,omitempty"`
	Intent          Intent          `json:"intent"`
	Status          Status          `json:"status"`
	RejectionReason RejectionReason `json:"rejectionReason,omitempty"`
	SubmissionMode  SubmissionMode  `json:"submissionMode"`
	Simulation      Simulation      `json:"simulation"`
	SigningPayload  *SigningPayload `json:"signingPayload,omitempty"`
	Manifest        *Manifest       `json:"manifest,omitempty"`
	SubmitParams    *SubmitParams   `json:"submitParams,omitempty"`
	PreparedAt      int64           `json:"preparedAt"`
	ExpiresAt       int64           `json:"expiresAt,omitempty"`
}
