package arbitrage

import k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"

type ArbitrageType string

const (
	ArbitrageTypeUnknown ArbitrageType = ""
	ArbitrageTypeAtomic  ArbitrageType = "atomic"
)

type Chain = k4k3ruOnchainCore.Chain
type Network = k4k3ruOnchainCore.Network

type Symbol string

type Venue string

const (
	VenueUnknown   Venue = ""
	VenueAerodrome Venue = "aerodrome"
	VenueBluefin   Venue = "bluefin"
	VenueCetus     Venue = "cetus"
	VenueMeteora   Venue = "meteora"
	VenueRaydium   Venue = "raydium"
	VenueUniswapV3 Venue = "uniswap-v3"
	VenueUniswapV4 Venue = "uniswap-v4"
)

type SourceFilter struct {
	Venues []Venue `json:"venues,omitempty"`
}
