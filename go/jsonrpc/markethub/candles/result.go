package candles

import "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"

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
	Closed      bool     `json:"closed"` // Interval ended, including the seal delay; later corrections remain possible.
	Revision    string   `json:"revision"`
	PublishedAt int64    `json:"publishedAt"` // Unix microseconds; zero means unknown.
	Coverage    Coverage `json:"coverage"`
}

// Result covers [From, To). To is the next request's From until the originally
// requested end is reached. Pages do not share an immutable publication snapshot.
type Result struct {
	Definition     string            `json:"definition"`
	Method         string            `json:"method"` // Calculation identity; histories with different methods must not be spliced.
	Asset          string            `json:"asset"`
	MarketType     string            `json:"marketType"`
	Market         *market.MarketRef `json:"market,omitempty"` // Concrete market for definition=venue.
	Symbol         market.Symbol     `json:"symbol,omitempty"` // Canonical pair for definition=venue.
	Currency       string            `json:"currency"`         // OHLC and quote volume currency: USD.
	Interval       string            `json:"interval"`
	From           int64             `json:"from"`                  // Unix seconds.
	To             int64             `json:"to"`                    // Processed range end in Unix seconds.
	HistoryFrom    *int64            `json:"historyFrom,omitempty"` // Earliest retained minute in this series, Unix seconds; absent while unknown or empty. Not limited to this page; may be unusable.
	AsOf           int64             `json:"asOf"`                  // Request evaluation time in Unix seconds.
	SentAt         int64             `json:"sentAt"`                // Public Gateway send time in Unix microseconds.
	Candles        []Candle          `json:"candles"`
	MissingMinutes int               `json:"missingMinutes"`
}
