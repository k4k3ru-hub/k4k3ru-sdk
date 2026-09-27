package scalping

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// TestSpotMaximumAmountRoundTrip verifies exact per-Open bounds without inferred observations.
//
// Version:
//   - 2026-09-27: Added.
func TestSpotMaximumAmountRoundTrip(t *testing.T) {
	for _, amount := range []string{"1", "100000000", "9007199254740993", "18446744073709551615"} {
		t.Run(amount, func(t *testing.T) {
			p := spotParams()
			p.ExecutionRule.Open.Spot.MaximumAmount = " " + amount + " "
			normalized := p.Normalize()
			if normalized.ExecutionRule.Open.Spot.MaximumAmount != amount {
				t.Fatal("maximum amount was not normalized exactly")
			}
			wire, err := json.Marshal(SubscribeParams{IdempotencyKey: "bounded-open", Params: &p})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(wire), `"spot":{"maximumAmount":"`+amount+`"}`) {
				t.Fatal("maximum amount lost precision or used the former field")
			}
			var decoded SubscribeParams
			if err := json.Unmarshal(wire, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Params == nil || !reflect.DeepEqual(*decoded.Params, normalized) {
				t.Fatal("saved maximum changed on round trip")
			}
			observation := decoded.Params.ObservationParams()
			if observation.Buy != nil || observation.Sell != nil {
				t.Fatal("spend maximum inferred an observation quantity")
			}
			normalized.ExecutionRule.Open.Spot.MaximumAmount = "2"
			if p.ExecutionRule.Open.Spot.MaximumAmount != " "+amount+" " {
				t.Fatal("normalization modified the caller")
			}
		})
	}
}

// TestSpotMaximumAmountRejectsInvalidJSON verifies a fixed input cannot silently become a maximum.
//
// Version:
//   - 2026-09-27: Added.
func TestSpotMaximumAmountRejectsInvalidJSON(t *testing.T) {
	p := spotParams()
	base, err := json.Marshal(SubscribeParams{IdempotencyKey: "bounded-open", Params: &p})
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"missing":      `{}`,
		"null":         `{"maximumAmount":null}`,
		"empty":        `{"maximumAmount":""}`,
		"zero":         `{"maximumAmount":"0"}`,
		"negative":     `{"maximumAmount":"-1"}`,
		"fraction":     `{"maximumAmount":"1.1"}`,
		"exponent":     `{"maximumAmount":"1e6"}`,
		"number":       `{"maximumAmount":1000000}`,
		"too long":     `{"maximumAmount":"` + strings.Repeat("9", 385) + `"}`,
		"former field": `{"amount":"1000000"}`,
		"both fields":  `{"maximumAmount":"1000000","amount":"1000000"}`,
		"duplicate":    `{"maximumAmount":"1","maximumAmount":"2"}`,
	}
	for name, spot := range cases {
		t.Run(name, func(t *testing.T) {
			wire := strings.Replace(string(base), `{"maximumAmount":"1000000"}`, spot, 1)
			if wire == string(base) {
				t.Fatal("test did not replace the spot rule")
			}
			original := SubscribeParams{ExecutionID: "unchanged"}
			decoded := original
			if err := json.Unmarshal([]byte(wire), &decoded); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("expected invalid parameter: %v", err)
			}
			if !reflect.DeepEqual(decoded, original) {
				t.Fatal("failed decoding changed the destination")
			}
		})
	}
	p.ExecutionRule.Open.Spot.MaximumAmount = ""
	if err := (SubscribeParams{IdempotencyKey: "bounded-open", Params: &p}).Validate(); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatalf("Go request without a maximum was accepted: %v", err)
	}
}
