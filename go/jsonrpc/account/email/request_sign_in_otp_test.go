package email

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAccountEmailRequestSignInOTPParamsJSON(t *testing.T) {
	t.Parallel()

	want := RequestSignInOTPParams{Email: "user@example.com"}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `{"email":"user@example.com"}` {
		t.Fatalf("Marshal() = %s, want %s", data, `{"email":"user@example.com"}`)
	}

	var got RequestSignInOTPParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestAccountEmailRequestSignInOTPResultJSON(t *testing.T) {
	t.Parallel()

	want := RequestSignInOTPResult{ExpiresAt: "2026-08-29T12:34:56Z"}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `{"expiresAt":"2026-08-29T12:34:56Z"}` {
		t.Fatalf("Marshal() = %s, want %s", data, `{"expiresAt":"2026-08-29T12:34:56Z"}`)
	}

	var got RequestSignInOTPResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}
