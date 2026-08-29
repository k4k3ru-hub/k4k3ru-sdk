package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAccountAPIRequestCredentialCreationOTPParamsJSON(t *testing.T) {
	t.Parallel()

	email := "user@example.com"
	want := RequestCredentialCreationOTPParams{Email: &email}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"email":"user@example.com"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got RequestCredentialCreationOTPParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestAccountAPIRequestCredentialCreationOTPParamsPhoneJSON(t *testing.T) {
	t.Parallel()

	phone := "+819012345678"
	want := RequestCredentialCreationOTPParams{Phone: &phone}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"phone":"+819012345678"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got RequestCredentialCreationOTPParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestAccountAPIRequestCredentialCreationOTPResultJSON(t *testing.T) {
	t.Parallel()

	email := "user@example.com"
	want := RequestCredentialCreationOTPResult{
		Purpose:   "account.api.create_credential",
		Email:     &email,
		ExpiresAt: "2026-08-29T12:34:56Z",
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"purpose":"account.api.create_credential","email":"user@example.com","expiresAt":"2026-08-29T12:34:56Z"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got RequestCredentialCreationOTPResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}
