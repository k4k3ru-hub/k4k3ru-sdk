package arbitrage

import "strings"

type LegResult struct {
	Venue     Venue  `json:"venue"`
	PoolID    string `json:"poolId"`
	TokenIn   string `json:"tokenIn"`
	TokenOut  string `json:"tokenOut"`
	AmountIn  string `json:"amountIn"`
	AmountOut string `json:"amountOut"`
}

type RouteDirection string

const (
	RouteDirectionUniswapV3ToV4 RouteDirection = "uniswap-v3-to-v4"
	RouteDirectionUniswapV4ToV3 RouteDirection = "uniswap-v4-to-v3"
)

// RouteDirectionForVenues builds the direction for an ordered venue pair.
//
// Parameters:
//   - first: first route venue.
//   - second: second route venue.
//
// Returns:
//   - Route direction; empty when either venue is unknown.
//
// Version:
//   - 2026-08-31: Added.
func RouteDirectionForVenues(first, second Venue) RouteDirection {
	firstValue := strings.ToLower(strings.TrimSpace(string(first)))
	secondValue := strings.ToLower(strings.TrimSpace(string(second)))
	if firstValue == "" || secondValue == "" {
		return ""
	}
	if firstValue == string(VenueUniswapV3) && secondValue == string(VenueUniswapV4) {
		return RouteDirectionUniswapV3ToV4
	}
	if firstValue == string(VenueUniswapV4) && secondValue == string(VenueUniswapV3) {
		return RouteDirectionUniswapV4ToV3
	}
	return RouteDirection(firstValue + "-to-" + secondValue)
}

type StateReferenceKind string

const (
	StateReferenceKindUnknown    StateReferenceKind = ""
	StateReferenceKindEVMBlock   StateReferenceKind = "evm-block"
	StateReferenceKindSolanaSlot StateReferenceKind = "solana-slot"
)

type StateReference struct {
	Kind      StateReferenceKind `json:"kind"`
	Number    uint64             `json:"number"`
	Hash      string             `json:"hash,omitempty"`
	Timestamp uint64             `json:"timestamp,omitempty"`
}

type RouteResult struct {
	Direction   RouteDirection `json:"direction"`
	Legs        []LegResult    `json:"legs"`
	GrossProfit string         `json:"grossProfit"`
}

type Result struct {
	ArbitrageType      ArbitrageType  `json:"arbitrageType"`
	Chain              Chain          `json:"chain"`
	Network            Network        `json:"network"`
	Symbol             Symbol         `json:"symbol"`
	InputAsset         string         `json:"inputAsset"`
	AmountIn           string         `json:"amountIn"`
	MinimumGrossProfit string         `json:"minimumGrossProfit"`
	SourceFilter       *SourceFilter  `json:"sourceFilter,omitempty"`
	StateReference     StateReference `json:"stateReference"`
	BlockNumber        uint64         `json:"blockNumber,omitempty"`
	BlockHash          string         `json:"blockHash,omitempty"`
	BlockTimestamp     uint64         `json:"blockTimestamp,omitempty"`
	EvaluatedAt        int64          `json:"evaluatedAt"`
	Routes             []RouteResult  `json:"routes"`
}
