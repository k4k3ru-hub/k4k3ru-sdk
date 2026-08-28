package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestSignHMACSHA256(t *testing.T) {
	t.Parallel()

	secret := []byte("0123456789abcdef0123456789abcdef")
	encodedSecret := base64.RawURLEncoding.EncodeToString(secret)
	payload := []byte("payload")

	got, err := SignHMACSHA256(encodedSecret, payload)
	if err != nil {
		t.Fatalf("SignHMACSHA256() error = %v", err)
	}

	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write(payload); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("SignHMACSHA256() = %q, want %q", got, want)
	}
	if _, err := base64.RawURLEncoding.DecodeString(got); err != nil {
		t.Fatalf("signature is not unpadded Base64 URL: %v", err)
	}
}
