package newpair

import "encoding/json"

type Finding struct {
	Status string `json:"status"`
	Value  string `json:"value,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type Token struct {
	ID       string             `json:"id"`
	Kind     string             `json:"kind"`
	Name     Finding            `json:"name"`
	Symbol   Finding            `json:"symbol"`
	Decimals Finding            `json:"decimals"`
	Security map[string]Finding `json:"security"`
}

// Position retains chain-specific block, slot, checkpoint and transaction coordinates.
// Numeric coordinates are decimal strings to avoid JSON integer precision loss.
type Position struct {
	Kind          string          `json:"kind"`
	Number        string          `json:"number"`
	ID            string          `json:"id"`
	TransactionID string          `json:"transactionId,omitempty"`
	EventIndex    string          `json:"eventIndex,omitempty"`
	Details       json.RawMessage `json:"details,omitempty"`
}

type Pair struct {
	LPPrincipal   *LPPrincipal `json:"lpPrincipal"`
	Fees          *Fees        `json:"fees"`
	Activity      *Activity    `json:"activity,omitempty"`
	ChainFamily   string       `json:"chainFamily"`
	Chain         string       `json:"chain"`
	Network       string       `json:"network"`
	Venue         string       `json:"venue"`
	PoolID        string       `json:"poolId"`
	Protocol      string       `json:"protocol"`
	Source        string       `json:"source"`
	Creation      Position     `json:"creation"`
	PoolCreatedAt int64        `json:"poolCreatedAt"`
	// SwapObservedAt records an observed swap, not necessarily the first historical swap.
	SwapObservedAt       *int64    `json:"swapObservedAt"`
	SwapObservedPosition *Position `json:"swapObservedPosition"`
	ConfirmedAt          *int64    `json:"confirmedAt"`
	// LPStateStatus is process-local: syncing, synced or unavailable. Missing means unverified.
	LPStateStatus   string    `json:"lpStateStatus"`
	LPStatePosition *Position `json:"lpStatePosition"`
	// LiquidityEvaluatedAt is the time of the last adopted USD calculation, not a block timestamp.
	LiquidityEvaluatedAt *int64             `json:"liquidityEvaluatedAt"`
	IsListed             bool               `json:"isListed"`
	ExclusionReason      string             `json:"exclusionReason,omitempty"`
	LastObservedAt       int64              `json:"lastObservedAt"`
	LiquidityUSD         Finding            `json:"liquidityUsd"`
	LiquidityMethod      string             `json:"liquidityMethod,omitempty"`
	Token0               Token              `json:"token0"`
	Token1               Token              `json:"token1"`
	Security             map[string]Finding `json:"security"`
	AssessmentStatus     string             `json:"assessmentStatus"`
	AssessedAt           *int64             `json:"assessedAt"`
	EvaluatorVersion     string             `json:"evaluatorVersion"`
	ProtocolState        json.RawMessage    `json:"protocolState,omitempty"`
	UnsupportedChecks    []string           `json:"unsupportedChecks"`
}

type Coverage struct {
	ChainFamily  string   `json:"chainFamily"`
	Chain        string   `json:"chain"`
	Network      string   `json:"network"`
	Venue        string   `json:"venue"`
	Source       string   `json:"source"`
	ObservedFrom int64    `json:"observedFrom"`
	Position     Position `json:"position"`
	Status       string   `json:"status"`
	Reason       string   `json:"reason,omitempty"`
}

type Result struct {
	Filter        Params     `json:"filter"`
	Epoch         string     `json:"epoch"`
	Version       uint64     `json:"version"`
	Timestamp     int64      `json:"timestamp"`
	Pairs         []Pair     `json:"pairs"`
	ExcludedPairs []Pair     `json:"excludedPairs"`
	Coverage      []Coverage `json:"coverage"`
	Truncated     bool       `json:"truncated"`
	NextCursor    string     `json:"nextCursor,omitempty"`
}

type GetResult struct {
	Epoch  string `json:"epoch"`
	Pair   *Pair  `json:"pair"`
	Reason string `json:"reason,omitempty"`
}

// Params returns the normalized subscription filter.
//
// Version:
//   - 2026-09-16: Added.
func (r Result) Params() Params { return r.Filter.Normalize() }
