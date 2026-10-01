package scalping

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
)

// TestRunResultPreservesRestoredSettings verifies environment and account recovery.
//
// Version:
//   - 2026-10-01: Added.
func TestRunResultPreservesRestoredSettings(t *testing.T) {
	for _, name := range []string{"run_spot_buy", "run_perpetual_sell"} {
		t.Run(name, func(t *testing.T) {
			var request RunParams
			if err := json.Unmarshal(runExample(t, name), &request); err != nil {
				t.Fatal(err)
			}
			request.Params.ExecutionRule.ReserveBufferBPS = pointer(uint64(0))
			request.Params.ExecutionRule.Exit.TakeProfit = &rule.Trigger{Type: rule.TriggerTypeReturnBPS, Value: "-0.0"}
			ack := RunResult{ExecutionID: "scalp_one", SubscriptionKey: "scalpsub_one", Params: request.Params}
			data, err := json.Marshal(ack)
			if err != nil {
				t.Fatal(err)
			}
			var restored RunResult
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(restored, ack.Normalize()) {
				t.Fatal("acknowledgement changed saved settings")
			}
			if !restored.Params.TestMode || *restored.Params.ExecutionRule.ReserveBufferBPS != 0 || restored.Params.ExecutionRule.Exit.TakeProfit != nil {
				t.Fatal("restoration changed mode, buffer or disabled trigger")
			}
			restored.Params.ExecutionRule.Markets[0].AccountAddress = "another-account"
			if request.Params.ExecutionRule.Markets[0].AccountAddress == "another-account" || request.Params.ExecutionRule.Exit.TakeProfit == nil {
				t.Fatal("encoding aliased or mutated caller configuration")
			}
		})
	}
}

// TestRunResultRejectsIncompleteACK verifies strict acknowledgement validation.
//
// Version:
//   - 2026-10-01: Added.
func TestRunResultRejectsIncompleteACK(t *testing.T) {
	params := runConfig()
	valid := RunResult{ExecutionID: "e", SubscriptionKey: "s", Params: &params}
	data, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		`{}`, `null`, `{"executionId":"e","subscriptionKey":"s"}`,
		`{"executionId":"e","subscriptionKey":"s","params":null}`,
		strings.Replace(string(data), `"executionId":"e"`, `"executionId":""`, 1),
		strings.Replace(string(data), `"subscriptionKey":"s"`, `"subscriptionKey":null`, 1),
		strings.Replace(string(data), `"subscriptionKey":"s"`, `"subscriptionKey":"s","SUBSCRIPTIONKEY":"other"`, 1),
		strings.Replace(string(data), `"executionId":"e"`, `"executionId":"e","unknown":1`, 1),
	} {
		original := valid.Normalize()
		decoded := original.Normalize()
		if err := json.Unmarshal([]byte(raw), &decoded); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("expected invalid acknowledgement: %v", err)
		}
		if !reflect.DeepEqual(original, decoded) {
			t.Fatal("failed decode replaced saved acknowledgement")
		}
	}
	valid.Params = nil
	if _, err := json.Marshal(valid); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatalf("encoded acknowledgement without settings: %v", err)
	}
}
