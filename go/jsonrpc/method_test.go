package jsonrpc

import (
	"errors"
	"strings"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestMethodValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		method  Method
		wantErr string
	}{
		{
			name:   "known method",
			method: MethodAccountEmailRequestCredentialCreationOTP,
		},
		{
			name:   "custom method",
			method: "Example.CustomMethod",
		},
		{
			name:   "maximum length",
			method: Method(strings.Repeat("a", maxMethodLength)),
		},
		{
			name:    "empty",
			wantErr: "failed to validate json rpc method: err_code=\"invalid_parameter\": method=empty",
		},
		{
			name:    "too long",
			method:  Method(strings.Repeat("a", maxMethodLength+1)),
			wantErr: "failed to validate json rpc method: err_code=\"invalid_parameter\": method=too_long actual_length=65 max_length=64",
		},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.method.Validate()
			if testCase.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate() error = nil")
			}
			if err.Error() != testCase.wantErr {
				t.Fatalf("Validate() error = %q, want %q", err.Error(), testCase.wantErr)
			}
			if !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
				t.Fatalf("errors.Is(Validate(), InvalidParameter()) = false")
			}
		})
	}
}
