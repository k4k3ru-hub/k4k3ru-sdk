package apperror

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestAppErrorJSONRoundTrip(t *testing.T) {
	t.Parallel()

	want := New(CodeInvalidParameter, "invalid value")
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(encoded) != `{"code":"invalid_parameter","message":"invalid value"}` {
		t.Fatalf("Marshal() = %s", encoded)
	}

	var got AppError
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.Code() != want.Code() || got.Message() != want.Message() {
		t.Fatalf("Unmarshal() = code=%q message=%q", got.Code(), got.Message())
	}
}

func TestAppErrorSupportsErrorChains(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("failed to perform operation: %w", InvalidParameter().WithMessage("invalid value"))
	if !errors.Is(err, InvalidParameter()) {
		t.Fatalf("errors.Is() = false: error=%v", err)
	}

	appErr := AsAppError(err)
	if appErr == nil {
		t.Fatal("AsAppError() = nil")
	}
	if appErr.Code() != CodeInvalidParameter || appErr.Message() != "invalid value" {
		t.Fatalf("AsAppError() = code=%q message=%q", appErr.Code(), appErr.Message())
	}
}

func TestApplicationErrorConstructorsReturnIndependentValues(t *testing.T) {
	t.Parallel()

	first := InvalidParameter()
	second := InvalidParameter()
	if first == second {
		t.Fatal("InvalidParameter() returned a shared pointer")
	}
}

func TestApplicationErrorConstructors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		got      *AppError
		wantCode Code
	}{
		{name: "conflict", got: Conflict(), wantCode: CodeConflict},
		{name: "forbidden", got: Forbidden(), wantCode: CodeForbidden},
		{name: "internal", got: Internal(), wantCode: CodeInternal},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if testCase.got.Code() != testCase.wantCode {
				t.Fatalf("Code() = %q, want %q", testCase.got.Code(), testCase.wantCode)
			}
			if !errors.Is(testCase.got, New(testCase.wantCode, "another message")) {
				t.Fatalf("errors.Is() = false: code=%q", testCase.wantCode)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()

	known := InvalidParameter().WithMessage("invalid value")
	tests := []struct {
		name     string
		err      error
		wantCode Code
	}{
		{name: "nil", wantCode: CodeUnexpected},
		{name: "known", err: fmt.Errorf("failed to perform operation: %w", known), wantCode: CodeInvalidParameter},
		{name: "unknown", err: errors.New("unknown error"), wantCode: CodeUnexpected},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := Normalize(testCase.err)
			if got.Code() != testCase.wantCode {
				t.Fatalf("Normalize() code = %q, want %q", got.Code(), testCase.wantCode)
			}
		})
	}

	if got := Normalize(known); got != known {
		t.Fatal("Normalize() did not return the application error from the chain")
	}
}
