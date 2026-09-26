package scalping

import (
	market "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observations "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
)

type EvaluationStatus string

const (
	EvaluationStatusMatched     EvaluationStatus = "matched"
	EvaluationStatusNotMatched  EvaluationStatus = "not_matched"
	EvaluationStatusUnavailable EvaluationStatus = "unavailable"
)

type AssetMetadata struct {
	Reference rule.AssetRef `json:"reference"`
	Symbol    string        `json:"symbol"`
	Decimals  uint8         `json:"decimals"`
}

type Result struct {
	EvaluationID string                `json:"evaluationId"`
	MarketType   market.MarketType     `json:"marketType"`
	Symbol       market.Symbol         `json:"symbol"`
	BaseAsset    rule.AssetRef         `json:"baseAsset"`
	QuoteAsset   rule.AssetRef         `json:"quoteAsset"`
	EvaluatedAt  int64                 `json:"evaluatedAt"`
	Metrics      *observations.Metrics `json:"metrics,omitempty"`
	Markets      []MarketEvaluation    `json:"markets"`
}

type MarketEvaluation struct {
	Price     observations.MarketPrice `json:"price"`
	Status    EvaluationStatus         `json:"status"`
	Candidate *Candidate               `json:"candidate,omitempty"`
	Reasons   []string                 `json:"reasons,omitempty"`
}

type Candidate struct {
	CandidateID string `json:"candidateId"`
	Revision    uint64 `json:"revision"`
	ExpiresAt   int64  `json:"expiresAt"`
}
