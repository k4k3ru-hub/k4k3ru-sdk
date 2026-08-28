package apperror

import (
	"errors"
	"strings"
	"testing"
)

func TestCodeValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		code    Code
		wantErr bool
	}{
		{name: "known", code: CodeInvalidParameter},
		{name: "unknown", code: "future_code"},
		{name: "maximum length", code: Code(strings.Repeat("a", maxCodeLength))},
		{name: "empty", wantErr: true},
		{name: "too long", code: Code(strings.Repeat("a", maxCodeLength+1)), wantErr: true},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.code.Validate()
			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate() error = nil")
			}
			if !errors.Is(err, InvalidParameter()) {
				t.Fatalf("Validate() error = %v, want invalid_parameter", err)
			}
		})
	}
}
