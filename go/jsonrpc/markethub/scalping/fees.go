package scalping

import "github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"

// Fees groups estimated charges for the enclosing buy or sell price.
// Unknown charges are omitted. Swap includes both LP and protocol shares;
// Taker describes immediate OrderBook execution when account fees are known.
// These amounts are excluded from gross price, receiveQuantity, ranking and spread.
type Fees struct {
	Swap  *Fee `json:"swap,omitempty"`
	Taker *Fee `json:"taker,omitempty"`
}

type Fee struct {
	Token    FeeToken        `json:"token"`
	Quantity market.Quantity `json:"quantity"`
}

// FeeToken identifies the charged asset in the enclosing MarketRef's chain/network
// namespace, or venue/network namespace for venue assets. Symbol is descriptive;
// identity comparisons must use AssetID and that scope, not Symbol alone.
type FeeToken struct {
	AssetID string `json:"assetId"`
	Symbol  string `json:"symbol"`
}
