// subscription_data.go
package market

import (
	"fmt"
	"strings"
)

type SubscriptionData struct {
	EventType EventType

	AssetClass AssetClass
	MarketType MarketType

	Symbol Symbol
	Venue  Venue

	// Required only when EventType is Candle.
	Interval string
}

// Key builds the finance data subscription key.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-06-17: Added.
func (d SubscriptionData) Key() (string, error) {
	parts := make([]string, 0, 6)

	if d.Venue != "" {
		v := strings.ToUpper(strings.TrimSpace(string(d.Venue)))
		parts = append(parts, "v="+v)
	}
	if d.Symbol != "" {
		v := strings.ToUpper(strings.TrimSpace(string(d.Symbol)))
		parts = append(parts, "s="+v)
	}
	if d.MarketType != "" {
		v := strings.ToUpper(strings.TrimSpace(string(d.MarketType)))
		parts = append(parts, "mt="+v)
	}
	if d.AssetClass != "" {
		v := strings.ToUpper(strings.TrimSpace(string(d.AssetClass)))
		parts = append(parts, "ac="+v)
	}
	if d.EventType != "" {
		v := strings.ToUpper(strings.TrimSpace(string(d.EventType)))
		parts = append(parts, "e="+v)
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("failed to build subscription key: subscription_data=empty")
	}

	return strings.Join(parts, ":"), nil
}
