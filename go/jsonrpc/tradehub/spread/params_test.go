package spread

import (
	"encoding/json"
	"errors"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	k4k3ruSDKMarketHubSpread "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/spread"
	k4k3ruSDKTradeHubExecution "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
)

func TestParamsValidateNormalizeAndSubscriptionKey(t *testing.T) {
	slippage, age, ttl := uint64(50), uint64(1_000), uint64(5_000)
	params := Params{
		Opportunity: k4k3ruSDKMarketHubSpread.Params{Symbol: " btc/usdc ", BaseAsset: " btc ", Quantity: "0.001"},
		Execution: ExecutionParams{Signer: " 0xabc ", SubmissionMode: " TRADEHUB-RELAY ", Conditions: &k4k3ruSDKTradeHubExecution.Conditions{
			MinimumNetProfit: "1", MaximumSlippageBPS: &slippage, MaximumOpportunityAgeMS: &age, ExecutionTTLMS: &ttl,
		}},
	}
	if err := params.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	normalized := params.Normalize()
	if normalized.Opportunity.Symbol != "BTC/USDC" || normalized.Execution.Signer != "0xabc" || normalized.Execution.SubmissionMode != k4k3ruSDKTradeHubExecution.SubmissionModeTradeHubRelay {
		t.Fatalf("Normalize() = %#v", normalized)
	}
	key, err := params.SubscriptionKey()
	if err != nil || key == "" {
		t.Fatalf("SubscriptionKey() = %q, %v", key, err)
	}
}

func TestParamsRejectsInvalidAndUnknownFields(t *testing.T) {
	var params Params
	if err := json.Unmarshal([]byte(`{"unknown":true}`), &params); !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	if err := (Params{}).Validate(); !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
		t.Fatalf("Validate() error = %v", err)
	}
}
