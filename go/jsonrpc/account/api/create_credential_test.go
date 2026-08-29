package api

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAccountAPICreateCredentialParamsJSON(t *testing.T) {
	t.Parallel()

	email := "user@example.com"
	want := CreateCredentialParams{
		Email:              &email,
		Code:               "052784",
		APIName:            "trading-bot",
		SignatureAlgorithm: "ed25519",
		ExpiresIn:          "30d",
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"email":"user@example.com","code":"052784","apiName":"trading-bot","signatureAlgorithm":"ed25519","expiresIn":"30d"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got CreateCredentialParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestAccountAPICreateCredentialParamsPhoneJSON(t *testing.T) {
	t.Parallel()

	phone := "+819012345678"
	want := CreateCredentialParams{
		Phone:              &phone,
		Code:               "052784",
		APIName:            "trading-bot",
		SignatureAlgorithm: "hmac-sha256",
		ExpiresIn:          "7d",
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"phone":"+819012345678","code":"052784","apiName":"trading-bot","signatureAlgorithm":"hmac-sha256","expiresIn":"7d"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got CreateCredentialParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestAccountAPICreateCredentialResultJSON(t *testing.T) {
	t.Parallel()

	want := CreateCredentialResult{
		AccountID:          1786180518874776239,
		APIName:            "trading-bot",
		APIKey:             "api-key",
		SignatureAlgorithm: "ed25519",
		SecretKey:          "secret-key",
		ExpiresAt:          "2026-09-28T12:34:56Z",
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"accountId":"1786180518874776239","apiName":"trading-bot","apiKey":"api-key","signatureAlgorithm":"ed25519","secretKey":"secret-key","expiresAt":"2026-09-28T12:34:56Z"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got CreateCredentialResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}
