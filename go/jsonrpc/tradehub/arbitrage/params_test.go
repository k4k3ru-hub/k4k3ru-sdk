package arbitrage

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	k4k3ruSDKMarketHubArbitrage "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/arbitrage"
	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

func validParams() Params {
	maximumSlippageBPS := uint64(30)
	maximumOpportunityAgeMS := uint64(1_000)
	executionTTLMS := uint64(2_000)
	return Params{
		Opportunity: k4k3ruSDKMarketHubArbitrage.Params{
			ArbitrageType:      k4k3ruSDKMarketHubArbitrage.ArbitrageTypeAtomic,
			Chain:              k4k3ruOnchainCore.ChainSui,
			Network:            k4k3ruOnchainCore.NetworkMainnet,
			Symbol:             "SUI/USDC",
			InputAsset:         "SUI",
			AmountIn:           "100",
			MinimumGrossProfit: "0.5",
			SourceFilter: &k4k3ruSDKMarketHubArbitrage.SourceFilter{
				Venues: []k4k3ruSDKMarketHubArbitrage.Venue{
					k4k3ruSDKMarketHubArbitrage.VenueCetus,
					k4k3ruSDKMarketHubArbitrage.VenueTurbos,
				},
			},
		},
		Execution: ExecutionParams{
			Signer:         "0x123",
			SubmissionMode: SubmissionModeTradeHubRelay,
			Conditions: &ExecutionConditions{
				MinimumNetProfit:        "0.2",
				MaximumSlippageBPS:      &maximumSlippageBPS,
				MaximumExecutionCost:    "0.1",
				MaximumOpportunityAgeMS: &maximumOpportunityAgeMS,
				ExecutionTTLMS:          &executionTTLMS,
			},
		},
	}
}

func TestParamsJSONRoundTrip(t *testing.T) {
	t.Parallel()

	want := validParams()
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON := `{"opportunity":{"arbitrageType":"atomic","chain":"sui","network":"mainnet","symbol":"SUI/USDC","inputAsset":"SUI","amountIn":"100","minimumGrossProfit":"0.5","sourceFilter":{"venues":["cetus","turbos"]}},"execution":{"signer":"0x123","submissionMode":"tradehub-relay","conditions":{"minimumNetProfit":"0.2","maximumSlippageBps":30,"maximumExecutionCost":"0.1","maximumOpportunityAgeMs":1000,"executionTtlMs":2000}}}`
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

	tests := []struct {
		name string
		data string
	}{
		{name: "params", data: `{"unexpected":true}`},
		{name: "execution", data: `{"opportunity":{},"execution":{"unexpected":true}}`},
		{name: "conditions", data: `{"opportunity":{},"execution":{"conditions":{"unexpected":true}}}`},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var params Params
			err := json.Unmarshal([]byte(testCase.data), &params)
			if err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
				t.Fatalf("Unmarshal() error = %v, want invalid parameter", err)
			}
		})
	}
}

func TestParamsUnmarshalJSONRejectsTrailingValue(t *testing.T) {
	t.Parallel()

	var params Params
	err := params.UnmarshalJSON([]byte(`{ } { }`))
	if err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
		t.Fatalf("UnmarshalJSON() error = %v, want invalid parameter", err)
	}
}

func TestParamsNormalizeDoesNotMutateConditions(t *testing.T) {
	t.Parallel()

	params := validParams()
	originalConditions := params.Execution.Conditions
	params.Opportunity.Chain = " SUI "
	params.Execution.Signer = " 0x123 "
	params.Execution.SubmissionMode = " TRADEHUB-RELAY "
	params.Execution.Conditions.MinimumNetProfit = " 0.2000 "
	params.Execution.Conditions.MaximumExecutionCost = " 0.1000 "
	normalized := params.Normalize()

	if normalized.Opportunity.Chain != k4k3ruOnchainCore.ChainSui || normalized.Execution.Signer != "0x123" || normalized.Execution.SubmissionMode != SubmissionModeTradeHubRelay {
		t.Fatalf("Normalize() = %#v", normalized)
	}
	if normalized.Execution.Conditions.MinimumNetProfit != "0.2" || normalized.Execution.Conditions.MaximumExecutionCost != "0.1" {
		t.Fatalf("Normalize() conditions = %#v", normalized.Execution.Conditions)
	}
	if normalized.Execution.Conditions == originalConditions {
		t.Fatal("Normalize() reused the source conditions pointer")
	}
	if originalConditions.MinimumNetProfit != " 0.2000 " || originalConditions.MaximumExecutionCost != " 0.1000 " {
		t.Fatalf("Normalize() mutated source conditions = %#v", originalConditions)
	}
}

func TestParamsValidateSupportsInitialChains(t *testing.T) {
	t.Parallel()

	for _, chain := range []k4k3ruOnchainCore.Chain{k4k3ruOnchainCore.ChainSui, k4k3ruOnchainCore.ChainBase} {
		params := validParams()
		params.Opportunity.Chain = chain
		if err := params.Validate(); err != nil {
			t.Fatalf("Validate() chain=%q error = %v", chain, err)
		}
	}
}

func TestParamsValidateRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Params)
	}{
		{name: "opportunity", mutate: func(p *Params) { p.Opportunity.AmountIn = "0" }},
		{name: "unsupported chain", mutate: func(p *Params) { p.Opportunity.Chain = k4k3ruOnchainCore.ChainEthereum }},
		{name: "empty signer", mutate: func(p *Params) { p.Execution.Signer = "" }},
		{name: "submission mode", mutate: func(p *Params) { p.Execution.SubmissionMode = "direct" }},
		{name: "null conditions", mutate: func(p *Params) { p.Execution.Conditions = nil }},
		{name: "empty minimum net profit", mutate: func(p *Params) { p.Execution.Conditions.MinimumNetProfit = "" }},
		{name: "invalid minimum net profit", mutate: func(p *Params) { p.Execution.Conditions.MinimumNetProfit = "one" }},
		{name: "negative minimum net profit", mutate: func(p *Params) { p.Execution.Conditions.MinimumNetProfit = "-1" }},
		{name: "null maximum slippage", mutate: func(p *Params) { p.Execution.Conditions.MaximumSlippageBPS = nil }},
		{name: "invalid maximum execution cost", mutate: func(p *Params) { p.Execution.Conditions.MaximumExecutionCost = "one" }},
		{name: "negative maximum execution cost", mutate: func(p *Params) { p.Execution.Conditions.MaximumExecutionCost = "-1" }},
		{name: "null maximum opportunity age", mutate: func(p *Params) { p.Execution.Conditions.MaximumOpportunityAgeMS = nil }},
		{name: "zero maximum opportunity age", mutate: func(p *Params) { zero := uint64(0); p.Execution.Conditions.MaximumOpportunityAgeMS = &zero }},
		{name: "null execution ttl", mutate: func(p *Params) { p.Execution.Conditions.ExecutionTTLMS = nil }},
		{name: "zero execution ttl", mutate: func(p *Params) { zero := uint64(0); p.Execution.Conditions.ExecutionTTLMS = &zero }},
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

func TestExecutionConditionsValidateAllowsZeroSlippageAndOptionalCost(t *testing.T) {
	t.Parallel()

	params := validParams()
	zero := uint64(0)
	params.Execution.Conditions.MaximumSlippageBPS = &zero
	params.Execution.Conditions.MaximumExecutionCost = ""
	if err := params.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestSubmissionModeValidate(t *testing.T) {
	t.Parallel()

	if err := SubmissionModeTradeHubRelay.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := SubmissionModeUnknown.Validate(); err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
		t.Fatalf("Validate() error = %v, want invalid parameter", err)
	}
}
