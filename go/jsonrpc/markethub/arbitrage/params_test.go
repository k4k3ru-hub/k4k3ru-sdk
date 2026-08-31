package arbitrage

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func validParams() Params {
	return Params{
		ArbitrageType:      ArbitrageTypeAtomic,
		Chain:              k4k3ruOnchainCore.ChainEthereum,
		Network:            k4k3ruOnchainCore.NetworkMainnet,
		Symbol:             "WBTC/USDT",
		InputAsset:         "WBTC",
		AmountIn:           "0.1",
		MinimumGrossProfit: "0.00001",
		SourceFilter:       &SourceFilter{Venues: []Venue{VenueUniswapV3, VenueUniswapV4}},
	}
}

func TestParamsJSONRoundTrip(t *testing.T) {
	t.Parallel()

	want := validParams()
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON := `{"arbitrageType":"atomic","chain":"ethereum","network":"mainnet","symbol":"WBTC/USDT","inputAsset":"WBTC","amountIn":"0.1","minimumGrossProfit":"0.00001","sourceFilter":{"venues":["uniswap-v3","uniswap-v4"]}}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}
	var got Params
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestParamsJSONRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	var params Params
	err := json.Unmarshal([]byte(`{"arbitrageType":"atomic","unexpected":true}`), &params)
	if err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
		t.Fatalf("Unmarshal() error = %v, want invalid parameter", err)
	}
}

func TestParamsNormalizeDoesNotMutateSourceFilter(t *testing.T) {
	t.Parallel()

	venues := []Venue{" UNISWAP-V4 ", " UNISWAP-V3 "}
	params := validParams()
	params.ArbitrageType = " ATOMIC "
	params.Chain = " ETHEREUM "
	params.Network = " MAINNET "
	params.Symbol = " wbtc/usdt "
	params.InputAsset = " wbtc "
	params.AmountIn = "0.1000"
	params.MinimumGrossProfit = ""
	params.SourceFilter = &SourceFilter{Venues: venues}
	normalized := params.Normalize()
	if normalized.ArbitrageType != ArbitrageTypeAtomic || normalized.Symbol != "WBTC/USDT" || normalized.AmountIn != "0.1" || normalized.MinimumGrossProfit != "0" {
		t.Fatalf("Normalize() = %#v", normalized)
	}
	wantVenues := []Venue{VenueUniswapV3, VenueUniswapV4}
	if !reflect.DeepEqual(normalized.SourceFilter.Venues, wantVenues) {
		t.Fatalf("Normalize() venues = %#v, want %#v", normalized.SourceFilter.Venues, wantVenues)
	}
	if !reflect.DeepEqual(venues, []Venue{" UNISWAP-V4 ", " UNISWAP-V3 "}) {
		t.Fatalf("Normalize() mutated venues = %#v", venues)
	}
}

func TestParamsValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Params)
	}{
		{name: "arbitrage type", mutate: func(p *Params) { p.ArbitrageType = "cross-chain" }},
		{name: "chain", mutate: func(p *Params) { p.Chain = "invalid" }},
		{name: "network", mutate: func(p *Params) { p.Network = "invalid" }},
		{name: "symbol", mutate: func(p *Params) { p.Symbol = "WBTC" }},
		{name: "input asset", mutate: func(p *Params) { p.InputAsset = "ETH" }},
		{name: "amount", mutate: func(p *Params) { p.AmountIn = "0" }},
		{name: "minimum profit", mutate: func(p *Params) { p.MinimumGrossProfit = "-1" }},
		{name: "one venue", mutate: func(p *Params) { p.SourceFilter.Venues = []Venue{VenueUniswapV3} }},
		{name: "duplicate venue", mutate: func(p *Params) { p.SourceFilter.Venues = []Venue{VenueUniswapV3, VenueUniswapV3} }},
		{name: "unsupported venue", mutate: func(p *Params) { p.SourceFilter.Venues = []Venue{VenueUniswapV3, "curve"} }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			params := validParams()
			testCase.mutate(&params)
			if err := params.Validate(); err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
				t.Fatalf("Validate() error = %v, want invalid parameter", err)
			}
		})
	}
}

func TestParamsValidateAllowsAllConfiguredSources(t *testing.T) {
	t.Parallel()

	params := validParams()
	params.SourceFilter = nil
	if err := params.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestParamsValidateAllowsAerodromeSource(t *testing.T) {
	t.Parallel()

	params := validParams()
	params.Chain = k4k3ruOnchainCore.ChainBase
	params.Symbol = "WETH/USDC"
	params.InputAsset = "WETH"
	params.SourceFilter.Venues = []Venue{VenueAerodrome, VenueUniswapV3, VenueUniswapV4}
	if err := params.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	normalized := params.Normalize()
	want := []Venue{VenueAerodrome, VenueUniswapV3, VenueUniswapV4}
	if !reflect.DeepEqual(normalized.SourceFilter.Venues, want) {
		t.Fatalf("Normalize() venues = %#v, want %#v", normalized.SourceFilter.Venues, want)
	}
}

func TestParamsValidateAllowsSolanaSources(t *testing.T) {
	t.Parallel()

	params := validParams()
	params.Chain = k4k3ruOnchainCore.ChainSolana
	params.Symbol = "SOL/USDC"
	params.InputAsset = "SOL"
	params.SourceFilter.Venues = []Venue{VenueMeteora, VenueRaydium}
	if err := params.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestParamsSubscriptionKeyNormalizesEquivalentSelections(t *testing.T) {
	t.Parallel()

	left := validParams()
	right := validParams()
	right.ArbitrageType = " ATOMIC "
	right.Chain = " ETHEREUM "
	right.Network = " MAINNET "
	right.Symbol = " wbtc/usdt "
	right.InputAsset = " wbtc "
	right.AmountIn = "0.1000"
	right.MinimumGrossProfit = "0.0000100"
	right.SourceFilter.Venues = []Venue{" UNISWAP-V4 ", " UNISWAP-V3 "}
	leftKey, err := left.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	rightKey, err := right.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	if leftKey != rightKey {
		t.Fatalf("keys differ: left=%q right=%q", leftKey, rightKey)
	}
	want := "MarketHub.Arbitrage:at=ATOMIC:c=ETHEREUM:n=MAINNET:s=WBTC/USDT:ia=WBTC:ai=0.1:mgp=0.00001:src=UNISWAP-V3,UNISWAP-V4"
	if leftKey != want {
		t.Fatalf("SubscriptionKey() = %q, want %q", leftKey, want)
	}
}
