package execution

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestObservationParamsRejectUnknownOwnerAndInvalidIDs verifies observation params reject unknown owner and invalid ids.
//
// Version:
//   - 2026-09-16: Added.
func TestObservationParamsRejectUnknownOwnerAndInvalidIDs(t *testing.T) {
	for _, raw := range []string{`{"executionId":"exec_one","accountId":7}`, `{"executionId":""}`, `{"executionId":"a b"}`, `{"executionId":"` + strings.Repeat("x", 65) + `"}`, `null`} {
		var p SubscribeParams
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	var p SubscribeParams
	if err := json.Unmarshal([]byte(`{"executionId":" Exec_A "}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.Normalize().ExecutionID != "Exec_A" {
		t.Fatal(p)
	}
	var u UnsubscribeParams
	if err := json.Unmarshal([]byte(`{"executionId":"exec_one","subscriptionKey":"sub_A","extra":true}`), &u); err == nil {
		t.Fatal("unknown field accepted")
	}
}

// TestObservationEventValidation verifies observation event validation.
//
// Version:
//   - 2026-09-16: Added.
func TestObservationEventValidation(t *testing.T) {
	hash := "0x" + strings.Repeat("1", 64)
	block := uint64(10)
	e := SubscriptionEvent{ExecutionID: "exec_one", SubscriptionKey: "sub_one", Sequence: 1, Kind: ExecutionEventSnapshot, Snapshot: &ExecutionSnapshot{Status: ObservationStatusSuccess, ObservedAt: 1, Onchain: &OnchainExecution{ChainFamily: ChainFamilyEVM, Chain: "base", Network: "sepolia", TransactionID: hash, BlockNumber: &block, BlockHash: hash}}}
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	e.Snapshot.Status = "confirming"
	if err := e.Validate(); err == nil {
		t.Fatal("unimplemented confirming accepted")
	}
	e.Snapshot.Status = ObservationStatusFailed
	if err := e.Validate(); err == nil {
		t.Fatal("missing failure accepted")
	}
	e.Snapshot.Failure = &ExecutionFailure{Code: "transaction_reverted"}
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	e.Kind = ExecutionEventError
	e.Error = &ObservationError{Code: "observation_unavailable", Retryable: true}
	if err := e.Validate(); err == nil {
		t.Fatal("mixed payload accepted")
	}
	e.Snapshot = nil
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
}
