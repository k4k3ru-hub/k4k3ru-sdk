package scalping

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// TestSubscribeStartAndResume verifies flat start encoding and reference-only recovery.
//
// Version:
//   - 2026-09-24: Added.
func TestSubscribeStartAndResume(t *testing.T) {
	p := spotParams()
	for _, request := range []SubscribeParams{{IdempotencyKey: "start-one", Params: &p}, {ExecutionID: "scalp_one"}} {
		encoded, err := json.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		var decoded SubscribeParams
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(request, decoded) {
			t.Fatal("request changed")
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			t.Fatal(err)
		}
		if _, nested := fields["params"]; nested {
			t.Fatal("nested marketType")
		}
		if request.Params != nil && string(fields["marketType"]) != `"spot"` {
			t.Fatal("top-level marketType missing")
		}
		if request.ExecutionID != "" && len(fields) != 1 {
			t.Fatal("resume contains overrides")
		}
	}
	copy := (SubscribeParams{Params: &p, IdempotencyKey: " start-one "}).Normalize()
	copy.Params.ExecutionRule.Open.Spot.Amount = "2"
	if p.ExecutionRule.Open.Spot.Amount == "2" || copy.IdempotencyKey != "start-one" {
		t.Fatal("normalization aliases parameters")
	}
}

// TestSubscribeRejectsOverrides verifies retries cannot silently replace execution settings.
//
// Version:
//   - 2026-09-24: Added.
func TestSubscribeRejectsOverrides(t *testing.T) {
	for _, wire := range []string{
		`{}`, `null`, `{"executionId":null}`, `{"executionId":""}`,
		`{"executionId":"scalp_one","marketType":"spot"}`,
		`{"executionId":"scalp_one","marketType":null}`,
		`{"executionId":"scalp_one","idempotencyKey":""}`,
		`{"executionId":"scalp_one","accountId":1}`,
		`{"executionId":"scalp_one","EXECUTIONID":"scalp_two"}`,
		`{"idempotencyKey":"key","params":{}}`,
	} {
		original := SubscribeParams{ExecutionID: "unchanged"}
		if err := json.Unmarshal([]byte(wire), &original); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("accepted invalid request: %v", err)
		}
		if original.ExecutionID != "unchanged" {
			t.Fatal("failed decode changed receiver")
		}
	}
	p := spotParams()
	for _, request := range []SubscribeParams{{Params: &p}, {IdempotencyKey: "key"}, {ExecutionID: "scalp_one", Params: &p}, {ExecutionID: "scalp_one", IdempotencyKey: "key"}} {
		if _, err := json.Marshal(request); err == nil {
			t.Fatal("invalid request encoded")
		}
	}
}
