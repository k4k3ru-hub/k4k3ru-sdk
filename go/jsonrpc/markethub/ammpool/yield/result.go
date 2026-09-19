// Package yield defines provider-reported AMM pool yield snapshots.
package yield

// Metric distinguishes unavailable data from a reported zero.
// APR and fee-rate values are decimal ratios; USD values are decimal dollar amounts.
type Metric struct {
	Value  *string `json:"value"`
	Status string  `json:"status"`
	Reason string  `json:"reason,omitempty"`
}
type APR struct {
	Fee     Metric      `json:"fee"`
	Reward  Metric      `json:"reward"`
	Total   Metric      `json:"total"`
	Rewards []RewardAPR `json:"rewards"`
	// complete, partial, or unknown. Empty legacy values mean unknown.
	RewardsStatus string `json:"rewardsStatus,omitempty"`
}
type RewardAPR struct {
	TokenID string `json:"tokenId"`
	APR     Metric `json:"apr"`
	// provider_reported, single_reward, or unknown.
	Attribution string `json:"attribution"`
}
type Period struct {
	Period    string `json:"period"`
	VolumeUSD Metric `json:"volumeUsd"`
	FeesUSD   Metric `json:"feesUsd"`
	APR       APR    `json:"apr"`
}
type Reward struct {
	TokenID     string `json:"tokenId"`
	DailyAmount Metric `json:"dailyAmount"`
	DailyUSD    Metric `json:"dailyUsd"`
	EndsAt      *int64 `json:"endsAt"`
	Status      string `json:"status"`
	Eligibility string `json:"eligibility"`
}
type Pool struct {
	Chain                string   `json:"chain"`
	Network              string   `json:"network"`
	Venue                string   `json:"venue"`
	PoolID               string   `json:"poolId"`
	Symbol               string   `json:"symbol"`
	BaseAssetID          string   `json:"baseAssetId"`
	QuoteAssetID         string   `json:"quoteAssetId"`
	Status               string   `json:"status"`
	Reason               string   `json:"reason,omitempty"`
	Source               string   `json:"source"`
	FetchedAt            *int64   `json:"fetchedAt"`
	SourceTimestamp      *int64   `json:"sourceTimestamp"`
	APRMethod            string   `json:"aprMethod"`
	APRDenominator       string   `json:"aprDenominator"`
	ProtocolFeeTreatment string   `json:"protocolFeeTreatment"`
	FeeRate              Metric   `json:"feeRate"`
	TVLUSD               Metric   `json:"tvlUsd"`
	Periods              []Period `json:"periods"`
	// APR is independent of Periods: its observation period is unspecified.
	APR           *APR     `json:"apr,omitempty"`
	Rewards       []Reward `json:"rewards"`
	RewardsStatus string   `json:"rewardsStatus"`
}
type Result struct {
	Filter     Params `json:"filter"`
	Epoch      string `json:"epoch"`
	Version    uint64 `json:"version"`
	Timestamp  int64  `json:"timestamp"`
	Pools      []Pool `json:"pools"`
	Truncated  bool   `json:"truncated"`
	NextCursor string `json:"nextCursor,omitempty"`
}
type GetResult struct {
	Pool   *Pool  `json:"pool"`
	Reason string `json:"reason,omitempty"`
}

// Params returns the normalized subscription filter.
//
// Version:
//   - 2026-09-19: Added.
func (r Result) Params() Params { return r.Filter.Normalize() }
