package email

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAccountEmailCreateCredentialParamsJSON(t *testing.T) {
	t.Parallel()

	want := CreateCredentialParams{
		Email: "user@example.com",
		Code:  "052784",
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"email":"user@example.com","code":"052784"}`
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

func TestAccountEmailCreateCredentialResultJSON(t *testing.T) {
	t.Parallel()

	want := CreateCredentialResult{
		AccountID:  1786180518874776239,
		Status:     "active",
		Email:      "user@example.com",
		OTPPurpose: "account.email.create_credential",
		BonusTicks: 1000000,
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"accountId":"1786180518874776239","status":"active","email":"user@example.com","otpPurpose":"account.email.create_credential","bonusTicks":1000000}`
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
