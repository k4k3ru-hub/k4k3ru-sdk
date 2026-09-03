package execution

import (
	"encoding/json"
	"testing"
)

func TestParamsValidate(t *testing.T) {
	t.Parallel()
	slippage, age, ttl := uint64(30), uint64(1000), uint64(2000)
	params := Params{
		Intent: Intent{Kind: IntentKindAtomicArbitrage, EvaluationID: " eval_1 ", RouteID: " route_1 "},
		Signer: " 0x1 ", SubmissionMode: SubmissionModeTradeHubRelay,
		Conditions:     &Conditions{MinimumNetProfit: "0.20", MaximumSlippageBPS: &slippage, MaximumExecutionCost: "0.10", MaximumOpportunityAgeMS: &age, ExecutionTTLMS: &ttl},
		IdempotencyKey: " key_1 ",
	}
	if err := params.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	normalized := params.Normalize()
	if normalized.Conditions.MinimumNetProfit != "0.2" || normalized.Conditions.MaximumExecutionCost != "0.1" {
		t.Fatalf("Normalize() conditions = %+v", normalized.Conditions)
	}
}

func TestParamsUnmarshalJSONRejectsUnknownField(t *testing.T) {
	t.Parallel()
	var params Params
	if err := json.Unmarshal([]byte(`{"unknown":true}`), &params); err == nil {
		t.Fatal("Unmarshal() error = nil")
	}
}
