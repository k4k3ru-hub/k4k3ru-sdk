package jsonrpc

import (
	"encoding/json"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestResponseUnmarshal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		data     string
		wantCode apperror.Code
		wantData string
	}{
		{
			name:     "result",
			data:     `{"id":1,"result":{"status":"ok"}}`,
			wantData: `{"status":"ok"}`,
		},
		{
			name:     "error",
			data:     `{"id":"request-1","error":{"code":"future_code","message":"future error"}}`,
			wantCode: "future_code",
		},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var response Response
			if err := json.Unmarshal([]byte(testCase.data), &response); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if string(response.Result) != testCase.wantData {
				t.Fatalf("Result = %s, want %s", response.Result, testCase.wantData)
			}
			if testCase.wantCode == "" {
				if response.Error != nil {
					t.Fatalf("Error = %v", response.Error)
				}
				return
			}
			if response.Error == nil || response.Error.Code() != testCase.wantCode {
				t.Fatalf("Error code = %v, want %q", response.Error, testCase.wantCode)
			}
		})
	}
}
