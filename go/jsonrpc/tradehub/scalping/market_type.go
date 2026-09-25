package scalping

import (
	market "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

func validateMarketType(value market.MarketType) error {
	if value != market.MarketTypeSpot && value != market.MarketTypePerpetual {
		return v.Invalid("validate scalping market type", "market_type", "invalid")
	}
	return nil
}
