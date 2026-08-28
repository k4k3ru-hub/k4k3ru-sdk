package signature

import (
	"errors"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
)

func TestBuildPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  []byte
		want    string
		wantErr bool
	}{
		{
			name:   "canonical object",
			params: []byte(`{"z":2,"a":{"d":4,"c":3},"items":[{"b":2,"a":1}]}`),
			want:   "AccountEmail.CreateCredential\n1787846400\nnonce-1\n{\"a\":{\"c\":3,\"d\":4},\"items\":[{\"a\":1,\"b\":2}],\"z\":2}",
		},
		{
			name: "omitted params",
			want: "AccountEmail.CreateCredential\n1787846400\nnonce-1\nnull",
		},
		{
			name:   "preserve number text",
			params: []byte(`{"value":1.2300e+4}`),
			want:   "AccountEmail.CreateCredential\n1787846400\nnonce-1\n{\"value\":1.2300e+4}",
		},
		{
			name:    "multiple root values",
			params:  []byte(`{} {}`),
			wantErr: true,
		},
		{
			name:    "invalid json",
			params:  []byte(`{"value":`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := BuildPayload(
				jsonrpc.MethodAccountEmailCreateCredential,
				1787846400,
				"nonce-1",
				testCase.params,
			)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("BuildPayload() error = nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildPayload() error = %v", err)
			}
			if string(got) != testCase.want {
				t.Fatalf("BuildPayload() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestBuildPayloadValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		method    jsonrpc.Method
		timestamp int64
		nonce     string
	}{
		{name: "empty method", timestamp: 1, nonce: "nonce"},
		{name: "invalid timestamp", method: "Example.Method", nonce: "nonce"},
		{name: "empty nonce", method: "Example.Method", timestamp: 1},
	}

	for _, tt := range tests {
		testCase := tt
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := BuildPayload(testCase.method, testCase.timestamp, testCase.nonce, nil)
			if !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("BuildPayload() error = %v, want invalid_parameter", err)
			}
		})
	}
}
