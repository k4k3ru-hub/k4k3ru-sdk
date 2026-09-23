package validation

import (
	"errors"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// TestNumberBaseTen verifies exact values and rejects alternate numeric syntax.
//
// Version:
//   - 2026-09-23: Added.
func TestNumberBaseTen(t *testing.T) {
	for input, expected := range map[string]string{
		"0008": "8", "0010": "10", "00.010": "1/100", "-0010.25": "-41/4",
		"9007199254740993": "9007199254740993", "-0": "0",
	} {
		number, err := Number("parse amount", "amount", input, false, true)
		if err != nil {
			t.Fatalf("parse %q: %v", input, err)
		}
		if number.RatString() != expected {
			t.Fatalf("parse %q: got %s, want %s", input, number.RatString(), expected)
		}
	}
	for _, input := range []string{"", "1/2", "0x10", "1e2", "+1", ".5", "1.", "--1", "1_000", "NaN", strings.Repeat("1", 385)} {
		if _, err := Number("parse amount", "amount", input, false, true); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid number accepted: %v", err)
		}
	}
	for _, input := range []string{"-1", "0.5"} {
		if _, err := Number("parse amount", "amount", input, true, false); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid unsigned integer accepted: %v", err)
		}
	}
}

// TestDecodeRejectsAmbiguousObjects verifies strict JSON at nested boundaries.
//
// Version:
//   - 2026-09-23: Added.
func TestDecodeRejectsAmbiguousObjects(t *testing.T) {
	for _, input := range []string{
		`null`, `[]`, `{}`, `{"value":null}`, `{"value":1} {"value":2}`,
		`{"value":1,"VALUE":2}`, `{"value":{"key":1,"\u006bey":2}}`,
		`{"value":1,"unknown":true}`, `{"value":`,
		`{"value":` + strings.Repeat("[", 130) + "0" + strings.Repeat("]", 130) + "}",
	} {
		var destination struct {
			Value any `json:"value"`
		}
		if err := Decode([]byte(input), &destination, "value"); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid object accepted: %v", err)
		}
	}
}
