package jsonrpc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAccountEmailRequestCredentialCreationOTPParamsJSON(t *testing.T) {
	t.Parallel()

	want := AccountEmailRequestCredentialCreationOTPParams{Email: "user@example.com"}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"email":"user@example.com"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got AccountEmailRequestCredentialCreationOTPParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestAccountEmailRequestCredentialCreationOTPResultJSON(t *testing.T) {
	t.Parallel()

	want := AccountEmailRequestCredentialCreationOTPResult{
		Purpose:   "account.email.create_credential",
		Email:     "user@example.com",
		ExpiresAt: "2026-08-29T12:34:56Z",
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"purpose":"account.email.create_credential","email":"user@example.com","expiresAt":"2026-08-29T12:34:56Z"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got AccountEmailRequestCredentialCreationOTPResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}
