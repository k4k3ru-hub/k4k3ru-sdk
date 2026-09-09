package api

import (
	"encoding/json"
	"testing"
)

func TestInternalCredentialAccountIDs(t *testing.T) {
	for _, p := range []any{
		&InternalListCredentialParams{}, &InternalRequestCredentialCreationOTPParams{}, &InternalCreateCredentialParams{}, &InternalRevokeCredentialParams{},
	} {
		if err := json.Unmarshal([]byte(`{"accountId":"18446744073709551615"}`), p); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(data, &value); err != nil {
			t.Fatal(err)
		}
		if value["accountId"] != "18446744073709551615" {
			t.Fatal("account precision lost")
		}
		if err := json.Unmarshal([]byte(`{"accountId":123}`), p); err == nil {
			t.Fatal("numeric account accepted")
		}
	}
}
