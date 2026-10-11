package candles

type Coverage struct {
	Known                     bool `json:"known"`
	ExpectedSourceMinutes     int  `json:"expectedSourceMinutes"`
	CoveredSourceMinutes      int  `json:"coveredSourceMinutes"`
	ContributingSourceMinutes int  `json:"contributingSourceMinutes"`
	MissingMinutes            int  `json:"missingMinutes"`
}

type Candle struct {
	Time        int64    `json:"time"` // UTC interval start in Unix seconds.
	Open        *float64 `json:"open"`
	High        *float64 `json:"high"`
	Low         *float64 `json:"low"`
	Close       *float64 `json:"close"`
	Volume      float64  `json:"volume"`
	QuoteVolume float64  `json:"quoteVolume"`
	TradeCount  uint64   `json:"tradeCount"`
	Quality     uint64   `json:"quality"`
	Usable      bool     `json:"usable"`
	Closed      bool     `json:"closed"`
	Revision    string   `json:"revision"`
	PublishedAt int64    `json:"publishedAt"` // Unix microseconds; zero means unknown.
	Coverage    Coverage `json:"coverage"`
}

// Result covers [From, To). To is the next request's From until the originally
// requested end is reached. Pages do not share an immutable publication snapshot.
type Result struct {
	Definition     string   `json:"definition"`
	Asset          string   `json:"asset"`
	MarketType     string   `json:"marketType"`
	Interval       string   `json:"interval"`
	From           int64    `json:"from"` // Unix seconds.
	To             int64    `json:"to"`   // Processed range end in Unix seconds.
	AsOf           int64    `json:"asOf"` // Request evaluation time in Unix seconds.
	Candles        []Candle `json:"candles"`
	MissingMinutes int      `json:"missingMinutes"`
}
