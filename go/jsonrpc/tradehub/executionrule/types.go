package executionrule

type MarketType string
type PositionSide string
type MarginMode string
type TriggerType string

const (
	MarketTypeSpot      MarketType = "spot"
	MarketTypePerpetual MarketType = "perpetual"

	PositionSideLong  PositionSide = "long"
	PositionSideShort PositionSide = "short"

	MarginModeCross    MarginMode = "cross"
	MarginModeIsolated MarginMode = "isolated"

	TriggerTypePrice     TriggerType = "price"
	TriggerTypeReturnBPS TriggerType = "return_bps"
)

// AssetRef identifies the reference asset whose smallest units define amounts.
type AssetRef struct {
	Chain   string `json:"chain,omitempty"`
	Venue   string `json:"venue,omitempty"`
	Network string `json:"network"`
	AssetID string `json:"assetId"`
}

// MarketRef identifies a pool or a venue-native instrument.
type MarketRef struct {
	Venue       string `json:"venue"`
	Network     string `json:"network"`
	Chain       string `json:"chain,omitempty"`
	PoolID      string `json:"poolId,omitempty"`
	VenueSymbol string `json:"venueSymbol,omitempty"`
}

type Rule struct {
	Open  OpenRule  `json:"open"`
	Close CloseRule `json:"close"`
}

type OpenRule struct {
	Spot               *SpotOpenRule `json:"spot,omitempty"`
	Perp               *PerpOpenRule `json:"perp,omitempty"`
	LimitPrice         *string       `json:"limitPrice,omitempty"`
	MaximumSlippageBPS *uint64       `json:"maximumSlippageBps"`
	ExecutionTTLMS     uint64        `json:"executionTtlMs"`
}

type SpotOpenRule struct {
	Amount string `json:"amount"`
}

type PerpOpenRule struct {
	Side       PositionSide `json:"side"`
	Quantity   string       `json:"quantity"`
	Leverage   uint32       `json:"leverage"`
	MarginMode MarginMode   `json:"marginMode"`
}

type CloseRule struct {
	TakeProfit         *Trigger       `json:"takeProfit,omitempty"`
	StopLoss           *Trigger       `json:"stopLoss,omitempty"`
	MaximumHoldingMS   *uint64        `json:"maximumHoldingMs,omitempty"`
	Spot               *SpotCloseRule `json:"spot,omitempty"`
	MaximumSlippageBPS *uint64        `json:"maximumSlippageBps"`
	ExecutionTTLMS     uint64         `json:"executionTtlMs"`
}

type SpotCloseRule struct {
	Markets []MarketRef `json:"markets"`
}

type Trigger struct {
	Type  TriggerType `json:"type"`
	Value string      `json:"value"`
}
