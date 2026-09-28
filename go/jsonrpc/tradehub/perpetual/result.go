package perpetual

import "encoding/json"

type Contract struct {
	Asset        uint32 `json:"asset"`
	Coin         string `json:"coin"`
	SizeDecimals int    `json:"sizeDecimals"`
	MaxLeverage  uint32 `json:"maxLeverage"`
	OnlyIsolated bool   `json:"onlyIsolated"`
}
type Prepared struct {
	PreparationID string          `json:"preparationId"`
	Intent        PrepareParams   `json:"intent"`
	Contract      Contract        `json:"contract"`
	Action        json.RawMessage `json:"action"`
	ActionHash    string          `json:"actionHash"`
	Digest        string          `json:"digest"`
	PreparedAt    uint64          `json:"preparedAt"`
	ExpiresAfter  uint64          `json:"expiresAfter"`
}
type PrepareResult struct {
	Prepared      Prepared `json:"prepared"`
	PreparedToken string   `json:"preparedToken"`
}
type MarginSummary struct {
	AccountValue          string `json:"accountValue"`
	TotalMarginUsed       string `json:"totalMarginUsed"`
	TotalNotionalPosition string `json:"totalNotionalPosition"`
	TotalRawUSD           string `json:"totalRawUsd"`
}
type Position struct {
	Quantity         string  `json:"quantity"`
	EntryPrice       *string `json:"entryPrice,omitempty"`
	LiquidationPrice *string `json:"liquidationPrice,omitempty"`
	Leverage         uint32  `json:"leverage"`
	MarginMode       string  `json:"marginMode"`
	MarginUsed       string  `json:"marginUsed"`
	UnrealizedPnL    string  `json:"unrealizedPnl"`
}
type AccountResult struct {
	Scope
	AccountMode      string         `json:"accountMode"`
	TradingSupported bool           `json:"tradingSupported"`
	Margin           *MarginSummary `json:"margin,omitempty"`
	Withdrawable     *string        `json:"withdrawable,omitempty"`
	Position         *Position      `json:"position,omitempty"`
	ObservedAt       uint64         `json:"observedAt"`
}
type SubmitResult struct {
	PreparationID          string `json:"preparationId"`
	Kind                   string `json:"kind"`
	Status                 string `json:"status"`
	OrderID                string `json:"orderId,omitempty"`
	ClientOrderID          string `json:"clientOrderId,omitempty"`
	RequestedSize          string `json:"requestedSize,omitempty"`
	FilledSize             string `json:"filledSize,omitempty"`
	AveragePrice           string `json:"averagePrice,omitempty"`
	ReconciliationRequired bool   `json:"reconciliationRequired"`
}
type Fill struct {
	TradeID   string `json:"tradeId"`
	Time      uint64 `json:"time"`
	Quantity  string `json:"quantity"`
	Price     string `json:"price"`
	Fee       string `json:"fee"`
	FeeToken  string `json:"feeToken"`
	ClosedPnL string `json:"closedPnl"`
}
type Fee struct {
	Token  string `json:"token"`
	Amount string `json:"amount"`
}
type OrderResult struct {
	Scope
	Found                bool   `json:"found"`
	OrderID              string `json:"orderId,omitempty"`
	ClientOrderID        string `json:"clientOrderId,omitempty"`
	Status               string `json:"status"`
	Side                 string `json:"side,omitempty"`
	ReduceOnly           bool   `json:"reduceOnly"`
	LimitPrice           string `json:"limitPrice,omitempty"`
	OriginalSize         string `json:"originalSize,omitempty"`
	RemainingSize        string `json:"remainingSize,omitempty"`
	ObservedFilledSize   string `json:"observedFilledSize"`
	ObservedAveragePrice string `json:"observedAveragePrice,omitempty"`
	Fees                 []Fee  `json:"fees"`
	Fills                []Fill `json:"fills"`
	FillsComplete        bool   `json:"fillsComplete"`
	FillsLimited         bool   `json:"fillsLimited"`
	ObservedAt           uint64 `json:"observedAt"`
}
