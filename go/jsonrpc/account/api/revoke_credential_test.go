package api

import (
	"encoding/json"
	"testing"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
)

func TestRevokeCredentialContract(t *testing.T) {
	if rpc.MethodAccountAPIRevokeCredential != "AccountAPI.RevokeCredential" {
		t.Fatal("incorrect method")
	}
	want := RevokeCredentialParams{CredentialID: ^uint64(0)}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"credentialId":"18446744073709551615"}` {
		t.Fatalf("incorrect encoding: %s", raw)
	}
	var got RevokeCredentialParams
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatal("id precision lost")
	}
	for _, invalid := range []string{`{"credentialId":123}`, `{"credentialId":"-1"}`, `{"credentialId":"18446744073709551616"}`} {
		if err := json.Unmarshal([]byte(invalid), &got); err == nil {
			t.Fatalf("invalid id accepted: %s", invalid)
		}
	}
	result := RevokeCredentialResult{CredentialID: want.CredentialID, Status: "example-status"}
	raw, err = json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded RevokeCredentialResult
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != result {
		t.Fatal("result changed")
	}
}
