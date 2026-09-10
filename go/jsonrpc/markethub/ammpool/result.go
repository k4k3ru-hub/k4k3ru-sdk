package ammpool

// Result is a complete latest-state response, not a Swap history event.
// Timestamps are Unix microseconds; decimal prices and protocol fields are strings.
type Result struct {
	Symbol             string  `json:"symbol"`
	MaxAgeSeconds      uint32  `json:"maxAgeSeconds"`
	Epoch              string  `json:"epoch"`
	Version            uint64  `json:"version"`
	Timestamp          int64   `json:"timestamp"`
	CompositeMid       *string `json:"compositeMid"`
	Available          bool    `json:"available"`
	Reason             string  `json:"reason,omitempty"`
	IncludedVenueCount int     `json:"includedVenueCount"`
	IncludedPoolCount  int     `json:"includedPoolCount"`
	Pools              []Pool  `json:"pools"`
}

type Pool struct {
	Venue           string      `json:"venue"`
	Chain           string      `json:"chain"`
	Network         string      `json:"network"`
	PoolID          string      `json:"poolId"`
	Symbol          string      `json:"symbol"`
	BaseAssetID     string      `json:"baseAssetId"`
	QuoteAssetID    string      `json:"quoteAssetId"`
	BaseDecimals    uint8       `json:"baseDecimals"`
	QuoteDecimals   uint8       `json:"quoteDecimals"`
	Protocol        string      `json:"protocol"`
	Generation      uint64      `json:"generation"`
	Version         uint64      `json:"version"`
	Price           *Price      `json:"price"`
	State           *State      `json:"state"`
	Retained        *Retained   `json:"retained"`
	Coverage        Coverage    `json:"coverage"`
	QuoteStatus     QuoteStatus `json:"quoteStatus"`
	Synchronized    bool        `json:"synchronized"`
	Included        bool        `json:"included"`
	ExclusionReason string      `json:"exclusionReason,omitempty"`
}

type Revision struct {
	Sequence    uint64   `json:"sequence"`
	Digest      string   `json:"digest"`
	DigestScope string   `json:"digestScope,omitempty"`
	Offset      []uint64 `json:"offset"`
}
type Price struct {
	Mid               string   `json:"mid"`
	Source            string   `json:"source"`
	ReceivedTimestamp int64    `json:"receivedTimestamp"`
	Revision          Revision `json:"revision"`
}
type Component struct {
	Fields            map[string]string `json:"fields"`
	ReceivedTimestamp int64             `json:"receivedTimestamp"`
	Revision          Revision          `json:"revision"`
}
type State struct {
	Type       string               `json:"type"`
	Components map[string]Component `json:"components"`
}
type Position struct {
	Kind     string  `json:"kind"`
	Sequence uint64  `json:"sequence"`
	Digest   string  `json:"digest"`
	Index    *uint64 `json:"index,omitempty"`
}
type Retained struct {
	Baseline          Position          `json:"baseline"`
	Position          Position          `json:"position"`
	ReceivedTimestamp int64             `json:"receivedTimestamp"`
	Fields            map[string]string `json:"fields"`
}
type Segment struct {
	Lower int64 `json:"lower"`
	Upper int64 `json:"upper"`
}
type Coverage struct {
	Unit     string    `json:"unit"`
	Status   string    `json:"status"`
	Segments []Segment `json:"segments"`
}
type QuoteStatus struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
}

// Params returns the subscription selectors echoed by this result.
//
// Version:
//   - 2026-09-10: Added.
func (r Result) Params() Params { return Params{Symbol: r.Symbol, MaxAgeSeconds: r.MaxAgeSeconds} }
