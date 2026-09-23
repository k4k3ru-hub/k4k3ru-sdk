package scalping

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// TestConditionsWindowJSON verifies omission, inclusive bounds and atomic decode failures.
//
// Version:
//   - 2026-09-24: Added.
func TestConditionsWindowJSON(t *testing.T) {
	for _, test := range []struct {
		name   string
		prefix string
		want   uint64
	}{
		{"omitted", "", 60000},
		{"minimum", `"windowMs":1,`, 1},
		{"shorter", `"windowMs":30000,`, 30000},
		{"maximum", `"windowMs":60000,`, 60000},
		{"case variant", `"WindowMs":30000,`, 30000},
		{"zero", `"windowMs":0,`, 0},
		{"null", `"windowMs":null,`, 0},
		{"case variant null", `"WindowMs":null,`, 0},
		{"above maximum", `"windowMs":60001,`, 0},
		{"max uint64", `"windowMs":18446744073709551615,`, 0},
		{"overflow", `"windowMs":18446744073709551616,`, 0},
		{"negative", `"windowMs":-1,`, 0},
		{"fraction", `"windowMs":1.5,`, 0},
		{"exponent", `"windowMs":6e4,`, 0},
		{"string", `"windowMs":"60000",`, 0},
		{"boolean", `"windowMs":true,`, 0},
		{"duplicate", `"windowMs":30000,"windowMs":60000,`, 0},
		{"case duplicate", `"windowMs":30000,"WindowMs":60000,`, 0},
		{"unknown field", `"unexpected":true,`, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := spotParams().Conditions
			decoded := original.Normalize()
			data := []byte(`{` + test.prefix + `"maximumDataAgeMs":2000,"tradeCount":{"minimum":0}}`)
			err := json.Unmarshal(data, &decoded)
			if test.want == 0 {
				if !errors.Is(err, apperror.InvalidParameter()) {
					t.Fatalf("expected inspectable validation error, got %v", err)
				}
				if !reflect.DeepEqual(decoded, original) {
					t.Fatal("failed decode changed receiver")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if decoded.WindowMS != test.want || decoded.MaximumDataAgeMS != 2000 || decoded.TradeCount == nil || decoded.TradeCount.Minimum == nil || *decoded.TradeCount.Minimum != 0 {
				t.Fatal("window default or explicit condition changed")
			}
			if decoded.PriceChangeBPS != nil || decoded.QuoteVolume != nil || decoded.BuyVolumeRatioBPS != nil {
				t.Fatal("trading conditions were invented")
			}
		})
	}
	var receiver *Conditions
	if err := receiver.UnmarshalJSON([]byte(`{}`)); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("nil receiver did not return a validation error")
	}
}

// TestConditionsWindowRequiresOtherInputs verifies only the observation window receives a default.
//
// Version:
//   - 2026-09-24: Added.
func TestConditionsWindowRequiresOtherInputs(t *testing.T) {
	for _, data := range []string{
		`null`, `[]`, `{}`,
		`{"tradeCount":{"minimum":1}}`,
		`{"maximumDataAgeMs":null,"tradeCount":{"minimum":1}}`,
		`{"maximumDataAgeMs":0,"tradeCount":{"minimum":1}}`,
		`{"maximumDataAgeMs":2000}`,
		`{"maximumDataAgeMs":2000,"tradeCount":{}}`,
		`{"maximumDataAgeMs":2000,"tradeCount":{"minimum":1,"unknown":2}}`,
	} {
		var value Conditions
		if err := json.Unmarshal([]byte(data), &value); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid conditions accepted: %v", err)
		}
	}
	var value Conditions
	err := json.Unmarshal([]byte(`{"windowMs":"60000","maximumDataAgeMs":2000,"tradeCount":{"minimum":1}}`), &value)
	var typeError *json.UnmarshalTypeError
	if !errors.As(err, &typeError) {
		t.Fatalf("underlying decode error was lost: %v", err)
	}
}

// TestConditionsWindowGoValues preserves the scalar API and rejects explicit invalid Go values.
//
// Version:
//   - 2026-09-24: Added.
func TestConditionsWindowGoValues(t *testing.T) {
	if DefaultWindowMS != 60000 || MaximumWindowMS != 60000 {
		t.Fatal("window constants changed")
	}
	for _, window := range []uint64{0, 1, 30000, DefaultWindowMS, MaximumWindowMS + 1, math.MaxUint64} {
		p := spotParams()
		p.Conditions.WindowMS = window
		normalized := p.Normalize()
		if normalized.Conditions.WindowMS != window {
			t.Fatal("normalization replaced an explicit window")
		}
		err := normalized.Validate()
		if window == 0 || window > MaximumWindowMS {
			if !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("invalid Go window accepted: %v", err)
			}
			if _, err := json.Marshal(SubscribeParams{IdempotencyKey: "key", Params: &p}); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("invalid Go window encoded: %v", err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
	}
}

// TestSubscribeWindowDefaultCanonicalization verifies equivalent start settings and stable resumes.
//
// Version:
//   - 2026-09-24: Added.
func TestSubscribeWindowDefaultCanonicalization(t *testing.T) {
	p := spotParams()
	p.Conditions.WindowMS = DefaultWindowMS
	request := SubscribeParams{IdempotencyKey: "key", Params: &p}
	explicit, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	omitted := strings.Replace(string(explicit), `"windowMs":60000,`, "", 1)
	if omitted == string(explicit) {
		t.Fatal("test did not omit the window")
	}
	var decoded SubscribeParams
	if err := json.Unmarshal([]byte(omitted), &decoded); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if string(canonical) != string(explicit) || !reflect.DeepEqual(request, decoded) {
		t.Fatal("omission and explicit default produced different settings")
	}
	for _, invalid := range []string{"0", "null", "60001"} {
		previous := decoded.Normalize()
		data := strings.Replace(string(explicit), `"windowMs":60000`, `"windowMs":`+invalid, 1)
		if err := json.Unmarshal([]byte(data), &decoded); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid start accepted: %v", err)
		}
		if !reflect.DeepEqual(previous, decoded) {
			t.Fatal("failed start changed receiver")
		}
	}
	if err := json.Unmarshal([]byte(`{"executionId":"scalp_one"}`), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Params != nil {
		t.Fatal("resume received default settings")
	}
	if err := json.Unmarshal([]byte(`{"executionId":"scalp_one","conditions":{"maximumDataAgeMs":2000,"tradeCount":{"minimum":1}}}`), &decoded); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("resume accepted defaulted settings override")
	}
}
