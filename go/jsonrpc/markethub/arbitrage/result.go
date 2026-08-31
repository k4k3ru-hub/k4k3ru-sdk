package arbitrage

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

type RouteResult struct {
	Direction   RouteDirection `json:"direction"`
	Legs        []LegResult    `json:"legs"`
	GrossProfit string         `json:"grossProfit"`
}

type Result struct {
	ArbitrageType      ArbitrageType `json:"arbitrageType"`
	Chain              Chain         `json:"chain"`
	Network            Network       `json:"network"`
	Symbol             Symbol        `json:"symbol"`
	InputAsset         string        `json:"inputAsset"`
	AmountIn           string        `json:"amountIn"`
	MinimumGrossProfit string        `json:"minimumGrossProfit"`
	SourceFilter       *SourceFilter `json:"sourceFilter,omitempty"`
	BlockNumber        uint64        `json:"blockNumber,omitempty"`
	BlockHash          string        `json:"blockHash,omitempty"`
	BlockTimestamp     uint64        `json:"blockTimestamp,omitempty"`
	EvaluatedAt        int64         `json:"evaluatedAt"`
	Routes             []RouteResult `json:"routes"`
}
