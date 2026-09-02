package arbitrage

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

func TestResultJSON(t *testing.T) {
	t.Parallel()

	result := Result{
		EvaluationID:        "eval_test",
		ArbitrageType:       ArbitrageTypeAtomic,
		Chain:               k4k3ruOnchainCore.ChainEthereum,
		Network:             k4k3ruOnchainCore.NetworkMainnet,
		Symbol:              "WBTC/USDT",
		InputAsset:          "WBTC",
		AmountIn:            "0.1",
		MinimumGrossProfit:  "0.00001",
		SourceFilter:        &SourceFilter{Venues: []Venue{VenueUniswapV3, VenueUniswapV4}},
		StateReference:      &StateReference{Kind: StateReferenceKindEVMBlock, Number: 123, Hash: "0xabc", Timestamp: 456},
		EvaluatedAt:         789,
		EvaluatedPoolCount:  2,
		EvaluatedRouteCount: 2,
		ExecutableRoutes: []RouteResult{{
			RouteID:     "route_test",
			Direction:   RouteDirectionUniswapV3ToV4,
			Legs:        []LegResult{{Venue: VenueUniswapV3, PoolID: "pool", TokenIn: "WBTC", TokenOut: "USDT", AmountIn: "0.1", AmountOut: "100"}},
			GrossProfit: "0.01",
		}},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"evaluationId":"eval_test","arbitrageType":"atomic","chain":"ethereum","network":"mainnet","symbol":"WBTC/USDT","inputAsset":"WBTC","amountIn":"0.1","minimumGrossProfit":"0.00001","sourceFilter":{"venues":["uniswap-v3","uniswap-v4"]},"stateReference":{"kind":"evm-block","number":123,"hash":"0xabc","timestamp":456},"evaluatedAt":789,"evaluatedPoolCount":2,"evaluatedRouteCount":2,"executableRoutes":[{"routeId":"route_test","direction":"uniswap-v3-to-v4","legs":[{"venue":"uniswap-v3","poolId":"pool","tokenIn":"WBTC","tokenOut":"USDT","amountIn":"0.1","amountOut":"100"}],"grossProfit":"0.01"}]}`
	if string(data) != want {
		t.Fatalf("Marshal() = %s, want %s", data, want)
	}
	var decoded Result
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, result) {
		t.Fatalf("Unmarshal() = %#v, want %#v", decoded, result)
	}
}

func TestSuiResultOmitsTopLevelStateReferenceAndUsesRouteReference(t *testing.T) {
	t.Parallel()

	result := Result{EvaluationID: "eval_sui", ArbitrageType: ArbitrageTypeAtomic, Chain: k4k3ruOnchainCore.ChainSui, Network: k4k3ruOnchainCore.NetworkMainnet, Symbol: "SUI/USDC", InputAsset: "SUI", AmountIn: "0.1", MinimumGrossProfit: "0", EvaluatedAt: 1, EvaluatedPoolCount: 4, EvaluatedRouteCount: 12, ExecutableRoutes: []RouteResult{{RouteID: "route_sui", Direction: "cetus-to-bluefin", StateReference: &StateReference{Kind: StateReferenceKindSuiCheckpoint, Number: 123}, Legs: []LegResult{}, GrossProfit: "0.01"}}}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"minimumGrossProfit":"0","stateReference"`) || !strings.Contains(string(data), `"stateReference":{"kind":"sui-checkpoint","number":123}`) {
		t.Fatalf("Marshal() = %s", data)
	}
}

func TestRouteDirectionForVenues(t *testing.T) {
	t.Parallel()

	if got := RouteDirectionForVenues(VenueRaydium, VenueMeteora); got != "raydium-to-meteora" {
		t.Fatalf("RouteDirectionForVenues() = %q", got)
	}
	if got := RouteDirectionForVenues(VenueUnknown, VenueMeteora); got != "" {
		t.Fatalf("RouteDirectionForVenues() = %q, want empty", got)
	}
	if got := RouteDirectionForVenues(VenueUniswapV3, VenueUniswapV4); got != RouteDirectionUniswapV3ToV4 {
		t.Fatalf("RouteDirectionForVenues() = %q, want %q", got, RouteDirectionUniswapV3ToV4)
	}
}

func TestStateReferenceKindSuiCheckpoint(t *testing.T) {
	t.Parallel()

	if StateReferenceKindSuiCheckpoint != "sui-checkpoint" {
		t.Fatalf("StateReferenceKindSuiCheckpoint = %q", StateReferenceKindSuiCheckpoint)
	}
}
