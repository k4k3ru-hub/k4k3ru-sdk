package api

import (
	"encoding/json"
	"reflect"
	"testing"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
)

func TestListCredentialContract(t *testing.T) {
	if rpc.MethodAccountAPIListCredential != "AccountAPI.ListCredential" {
		t.Fatal("incorrect method")
	}
	params, err := json.Marshal(ListCredentialParams{Page: 2})
	if err != nil {
		t.Fatal(err)
	}
	if string(params) != `{"page":2}` {
		t.Fatalf("params: %s", params)
	}
	algorithm := "ed25519"
	expires := "2026-10-09T00:00:00Z"
	want := ListCredentialResult{Credentials: []ListCredentialCredential{{ID: 18446744073709551615, Name: "bot", APIKey: "example-key", Status: "active", SignatureAlgorithm: &algorithm, Scopes: []string{}, ExpiresAt: &expires, CreatedAt: "2026-09-09T00:00:00Z", UpdatedAt: "2026-09-09T00:00:00Z"}}, Page: 2, Limit: 20, Total: 21, TotalPages: 2}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"credentials":[{"id":"18446744073709551615","name":"bot","apiKey":"example-key","status":"active","signatureAlgorithm":"ed25519","scopes":[],"expiresAt":"2026-10-09T00:00:00Z","createdAt":"2026-09-09T00:00:00Z","updatedAt":"2026-09-09T00:00:00Z"}],"page":2,"limit":20,"total":21,"totalPages":2}`
	if string(raw) != expected {
		t.Fatalf("result: %s", raw)
	}
	var got ListCredentialResult
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("round trip changed result")
	}
}

func TestListCredentialEmptyAndLegacyValues(t *testing.T) {
	var value ListCredentialResult
	for _, raw := range []string{`{"credentials":[],"page":1,"limit":20,"total":0,"totalPages":0}`, `{"credentials":[{"id":"1","signatureAlgorithm":null,"expiresAt":null,"scopes":[]}]}`} {
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			t.Fatal(err)
		}
		if value.Credentials == nil {
			t.Fatal("empty list became nil")
		}
		if len(value.Credentials) > 0 && (value.Credentials[0].ExpiresAt != nil || value.Credentials[0].SignatureAlgorithm != nil) {
			t.Fatal("legacy null changed")
		}
	}
}
