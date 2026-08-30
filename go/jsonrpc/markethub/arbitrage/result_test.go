package arbitrage

import (
	"encoding/json"
	"testing"

	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

func TestResultJSON(t *testing.T) {
	t.Parallel()

	result := Result{
		ArbitrageType:      ArbitrageTypeAtomic,
		Chain:              k4k3ruOnchainCore.ChainEthereum,
		Network:            k4k3ruOnchainCore.NetworkMainnet,
		Symbol:             "WBTC/USDT",
		InputAsset:         "WBTC",
		AmountIn:           "0.1",
		MinimumGrossProfit: "0.00001",
		SourceFilter:       &SourceFilter{Venues: []Venue{VenueUniswapV3, VenueUniswapV4}},
		BlockNumber:        123,
		BlockHash:          "0xabc",
		BlockTimestamp:     456,
		EvaluatedAt:        789,
		Routes: []RouteResult{{
			Direction:   RouteDirectionUniswapV3ToV4,
			Legs:        []LegResult{{Venue: VenueUniswapV3, PoolID: "pool", TokenIn: "WBTC", TokenOut: "USDT", AmountIn: "0.1", AmountOut: "100", GasEstimate: "1"}},
			GrossProfit: "0.01", TotalGasEstimate: "2", MeetsProfitBuffer: true,
		}},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"arbitrageType":"atomic","chain":"ethereum","network":"mainnet","symbol":"WBTC/USDT","inputAsset":"WBTC","amountIn":"0.1","minimumGrossProfit":"0.00001","sourceFilter":{"venues":["uniswap-v3","uniswap-v4"]},"blockNumber":123,"blockHash":"0xabc","blockTimestamp":456,"evaluatedAt":789,"routes":[{"direction":"uniswap-v3-to-v4","legs":[{"venue":"uniswap-v3","poolId":"pool","tokenIn":"WBTC","tokenOut":"USDT","amountIn":"0.1","amountOut":"100","gasEstimate":"1"}],"grossProfit":"0.01","totalGasEstimate":"2","meetsProfitBuffer":true}]}`
	if string(data) != want {
		t.Fatalf("Marshal() = %s, want %s", data, want)
	}
}
