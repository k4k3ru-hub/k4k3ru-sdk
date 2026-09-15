package launch

type Finding struct {
	Status string `json:"status"`
	Value  string `json:"value,omitempty"`
	Reason string `json:"reason,omitempty"`
}
type Token struct {
	Address        string  `json:"address"`
	Name           Finding `json:"name"`
	Symbol         Finding `json:"symbol"`
	Decimals       Finding `json:"decimals"`
	CreatedAt      *int64  `json:"createdAt"`
	CreationBlock  *uint64 `json:"creationBlock"`
	Creation       Finding `json:"creation"`
	Owner          Finding `json:"owner"`
	Implementation Finding `json:"implementation"`
	Admin          Finding `json:"admin"`
	Beacon         Finding `json:"beacon"`
}
type Launch struct {
	Chain             string   `json:"chain"`
	Network           string   `json:"network"`
	Venue             string   `json:"venue"`
	PoolID            string   `json:"poolId"`
	Emitter           string   `json:"emitter"`
	Event             string   `json:"event"`
	BlockNumber       uint64   `json:"blockNumber"`
	BlockHash         string   `json:"blockHash"`
	TransactionHash   string   `json:"transactionHash"`
	LogIndex          uint     `json:"logIndex"`
	LaunchedAt        int64    `json:"launchedAt"`
	DiscoveredAt      int64    `json:"discoveredAt"`
	AssessedAt        *int64   `json:"assessedAt"`
	AssessmentStatus  string   `json:"assessmentStatus"`
	EvaluatorVersion  string   `json:"evaluatorVersion"`
	Token0            Token    `json:"token0"`
	Token1            Token    `json:"token1"`
	Hooks             Finding  `json:"hooks"`
	UnsupportedChecks []string `json:"unsupportedChecks"`
}
type Coverage struct {
	Chain        string `json:"chain"`
	Network      string `json:"network"`
	Venue        string `json:"venue"`
	Emitter      string `json:"emitter"`
	ObservedFrom int64  `json:"observedFrom"`
	ThroughBlock uint64 `json:"throughBlock"`
	Status       string `json:"status"`
	Reason       string `json:"reason,omitempty"`
}

// Result is a complete replacement snapshot; absence removes a previously visible launch.
type Result struct {
	Filter     Params     `json:"filter"`
	Epoch      string     `json:"epoch"`
	Version    uint64     `json:"version"`
	Timestamp  int64      `json:"timestamp"`
	Launches   []Launch   `json:"launches"`
	Coverage   []Coverage `json:"coverage"`
	Truncated  bool       `json:"truncated"`
	NextCursor string     `json:"nextCursor,omitempty"`
}
type GetResult struct {
	Epoch  string  `json:"epoch"`
	Launch *Launch `json:"launch"`
	Reason string  `json:"reason,omitempty"`
}

// Params returns the filter identifying this snapshot's subscription.
//
// Version:
//   - 2026-09-15: Added.
func (r Result) Params() Params { return r.Filter.Normalize() }
