package jsonrpc

import (
	"encoding/json"
	"testing"
)

func TestRequestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		request *Request
		wantErr string
	}{
		{
			name: "valid",
			request: &Request{
				ID:     json.RawMessage(`"request-1"`),
				Method: MethodAccountEmailRequestCredentialCreationOTP,
			},
		},
		{
			name:    "null request",
			wantErr: "failed to validate json rpc request: request=null",
		},
		{
			name:    "empty id",
			request: &Request{Method: MethodAccountEmailRequestCredentialCreationOTP},
			wantErr: "failed to validate json rpc request: id=empty",
		},
		{
			name: "invalid method",
			request: &Request{
				ID: json.RawMessage(`1`),
			},
			wantErr: "failed to validate json rpc request: failed to validate json rpc method: method=empty",
		},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.request.Validate()
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
		})
	}
}

func TestRequestMarshal(t *testing.T) {
	t.Parallel()

	request := Request{
		ID:     json.RawMessage(`1`),
		Method: MethodAccountEmailCreateCredential,
		Params: json.RawMessage(`{"email":"user@example.com","code":"123456"}`),
		Auth: &Auth{
			APIKey:    "api-key",
			Timestamp: 1787846400,
			Nonce:     "nonce-1",
			Signature: "signature",
		},
	}

	got, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"id":1,"method":"AccountEmail.CreateCredential","params":{"email":"user@example.com","code":"123456"},"auth":{"apiKey":"api-key","timestamp":1787846400,"nonce":"nonce-1","signature":"signature"}}`
	if string(got) != want {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
}
