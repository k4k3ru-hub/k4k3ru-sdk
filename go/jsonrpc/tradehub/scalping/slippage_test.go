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

// TestSharedSlippageDefaults verifies JSON and Go callers share canonical round-trip settings.
//
// Version:
//   - 2026-09-27: Added.
func TestSharedSlippageDefaults(t *testing.T) {
	for _, build := range []func() Params{spotParams, perpParams} {
		for _, tc := range []struct {
			name  string
			input *uint64
			want  uint64
		}{
			{"omitted", nil, rule.DefaultMaximumSlippageBPS},
			{"zero", pointer(uint64(0)), 0},
			{"custom", pointer(uint64(125)), 125},
			{"upper bound", pointer(uint64(10000)), 10000},
		} {
			p := build()
			t.Run(string(p.MarketType)+"/"+tc.name, func(t *testing.T) {
				p.ExecutionRule.MaximumSlippageBPS = tc.input
				if err := p.Validate(); err != nil {
					t.Fatal(err)
				}
				encoded, err := json.Marshal(p)
				if err != nil {
					t.Fatal(err)
				}
				if tc.input == nil && strings.Contains(string(encoded), "maximumSlippageBps") {
					t.Fatal("unset optional field was serialized")
				}
				var decoded Params
				if err := json.Unmarshal(encoded, &decoded); err != nil {
					t.Fatal(err)
				}
				if decoded.ExecutionRule.MaximumSlippageBPS == nil || *decoded.ExecutionRule.MaximumSlippageBPS != tc.want {
					t.Fatal("incorrect shared slippage")
				}
				if !reflect.DeepEqual(p.Normalize(), decoded) {
					t.Fatal("Go and JSON normalization differ")
				}
				wire, err := json.Marshal(SubscribeParams{IdempotencyKey: "start", Params: &p})
				if err != nil {
					t.Fatal(err)
				}
				var request SubscribeParams
				if err := json.Unmarshal(wire, &request); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(request.Params, &decoded) || strings.Count(string(wire), `"maximumSlippageBps"`) != 1 {
					t.Fatal("subscription lost the shared setting")
				}
				*request.Params.ExecutionRule.MaximumSlippageBPS = 75
				if tc.input != nil && *tc.input != tc.want || tc.input == nil && p.ExecutionRule.MaximumSlippageBPS != nil {
					t.Fatal("normalization mutated the caller")
				}
			})
		}
	}
}

// TestSharedSlippageRejectsInvalidJSON verifies legacy fields cannot be silently defaulted.
//
// Version:
//   - 2026-09-27: Added.
func TestSharedSlippageRejectsInvalidJSON(t *testing.T) {
	base, err := json.Marshal(spotParams())
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(map[string]any){
		"null":       func(r map[string]any) { r["maximumSlippageBps"] = nil },
		"negative":   func(r map[string]any) { r["maximumSlippageBps"] = -1 },
		"fractional": func(r map[string]any) { r["maximumSlippageBps"] = 0.5 },
		"string":     func(r map[string]any) { r["maximumSlippageBps"] = "50" },
		"range":      func(r map[string]any) { r["maximumSlippageBps"] = 10001 },
		"old open": func(r map[string]any) {
			r["open"].(map[string]any)["maximumSlippageBps"] = 10
		},
		"old close": func(r map[string]any) {
			r["close"].(map[string]any)["maximumSlippageBps"] = 10
		},
		"old request": func(r map[string]any) {
			delete(r, "maximumSlippageBps")
			r["open"].(map[string]any)["maximumSlippageBps"] = 10
			r["close"].(map[string]any)["maximumSlippageBps"] = 100
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			var object map[string]any
			if err := json.Unmarshal(base, &object); err != nil {
				t.Fatal(err)
			}
			change(object["executionRule"].(map[string]any))
			object["idempotencyKey"] = "start"
			wire, err := json.Marshal(object)
			if err != nil {
				t.Fatal(err)
			}
			original := SubscribeParams{ExecutionID: "unchanged"}
			decoded := original
			if err := json.Unmarshal(wire, &decoded); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("expected invalid parameter: %v", err)
			}
			if !reflect.DeepEqual(original, decoded) {
				t.Fatal("failed decoding changed the destination")
			}
		})
	}
}
