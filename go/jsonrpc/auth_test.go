package jsonrpc

import (
	"errors"
	"testing"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestAuthValidateTimestamp(t *testing.T) {
	t.Parallel()

	nowUnix := time.Now().Unix()
	tests := []struct {
		name      string
		auth      Auth
		maxAgeSec int64
		wantErr   error
	}{
		{name: "current", auth: Auth{Timestamp: nowUnix}, maxAgeSec: 60},
		{name: "empty", maxAgeSec: 60, wantErr: apperror.InvalidParameter()},
		{name: "negative maximum age", auth: Auth{Timestamp: nowUnix}, maxAgeSec: -1, wantErr: apperror.InvalidParameter()},
		{name: "future", auth: Auth{Timestamp: nowUnix + 60}, maxAgeSec: 60, wantErr: apperror.InvalidParameter()},
		{name: "expired", auth: Auth{Timestamp: nowUnix - 120}, maxAgeSec: 60, wantErr: apperror.Expired()},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.auth.ValidateTimestamp(testCase.maxAgeSec)
			if testCase.wantErr == nil {
				if err != nil {
					t.Fatalf("ValidateTimestamp() error = %v", err)
				}
				return
			}
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("ValidateTimestamp() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}
