package market

import (
	"fmt"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

type MarketData struct {
	AssetClass AssetClass `json:"ac"`
	MarketType MarketType `json:"mt"`

	Symbol      Symbol `json:"s"`
	Venue       Venue  `json:"v"`
	VenueSymbol string `json:"vs,omitempty"`

	BestBidPrice string `json:"bbp,omitempty"`
	BestBidSize  string `json:"bbs,omitempty"`
	BestAskPrice string `json:"bap,omitempty"`
	BestAskSize  string `json:"bas,omitempty"`

	LastPrice string `json:"lp,omitempty"`
	LastSize  string `json:"ls,omitempty"`

	MarkPrice  string `json:"mp,omitempty"`
	IndexPrice string `json:"ip,omitempty"`

	OpenPrice string `json:"op,omitempty"`
	HighPrice string `json:"hp,omitempty"`
	LowPrice  string `json:"lop,omitempty"`
	Volume    string `json:"vol,omitempty"`

	Bids []MarketDataLevel `json:"b,omitempty"`
	Asks []MarketDataLevel `json:"a,omitempty"`

	Timestamp int64 `json:"ts"`

	Sequence string `json:"seq,omitempty"`
	Checksum string `json:"cs,omitempty"`
	Snapshot bool   `json:"snapshot,omitempty"`
}

type MarketDataLevel struct {
	Price    string `json:"p"`
	Quantity string `json:"q"`
}

// Key builds the market-data subscription key.
//
// Returns:
//   - Subscription key.
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-15: Moved from the subscription package.
func (d MarketData) Key() (string, error) {
	marketType := strings.ToUpper(string(d.MarketType))
	venue := strings.ToUpper(strings.TrimSpace(string(d.Venue)))
	symbol := strings.ToUpper(strings.TrimSpace(string(d.Symbol)))

	if marketType == "" {
		return "", fmt.Errorf("failed to build subscription key from market data: %w: market_type=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if venue == "" {
		return "", fmt.Errorf("failed to build subscription key from market data: %w: venue=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if symbol == "" {
		return "", fmt.Errorf("failed to build subscription key from market data: %w: symbol=empty", k4k3ruSDKAppError.InvalidParameter())
	}

	return fmt.Sprintf("%s:%s:%s", venue, symbol, marketType), nil
}

// Validate validates market data.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-15: Moved from the subscription package.
func (d MarketData) Validate() error {
	if err := d.Symbol.Validate(); err != nil {
		return err
	}
	if err := d.Venue.Validate(); err != nil {
		return err
	}
	if d.Timestamp <= 0 {
		return fmt.Errorf("failed to validate market data: timestamp=out_of_range")
	}
	return nil
}
