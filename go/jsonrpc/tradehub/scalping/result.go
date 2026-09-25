package scalping

import (
	market "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
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
	EvaluationID string             `json:"evaluationId"`
	MarketType   market.MarketType  `json:"marketType"`
	BaseAsset    AssetMetadata      `json:"baseAsset"`
	QuoteAsset   AssetMetadata      `json:"quoteAsset"`
	EvaluatedAt  int64              `json:"evaluatedAt"`
	Markets      []MarketEvaluation `json:"markets"`
}

type MarketEvaluation struct {
	Market    rule.MarketRef   `json:"market"`
	Status    EvaluationStatus `json:"status"`
	Metrics   *Metrics         `json:"metrics,omitempty"`
	Candidate *Candidate       `json:"candidate,omitempty"`
	Reasons   []string         `json:"reasons,omitempty"`
}

type Candidate struct {
	CandidateID string `json:"candidateId"`
	Revision    uint64 `json:"revision"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type Metrics struct {
	WindowStart       int64   `json:"windowStart"`
	WindowEnd         int64   `json:"windowEnd"`
	LastObservedAt    int64   `json:"lastObservedAt"`
	PriceChangeBPS    *string `json:"priceChangeBps,omitempty"`
	QuoteVolume       *string `json:"quoteVolume,omitempty"`
	TradeCount        *uint64 `json:"tradeCount,omitempty"`
	BuyVolumeRatioBPS *string `json:"buyVolumeRatioBps,omitempty"`
}
