package authentication

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
)

func TestSignRequestSupportsConfiguredAlgorithms(t *testing.T) {
	t.Parallel()

	hmacSecret := base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	ed25519Secret := base64.RawURLEncoding.EncodeToString(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)))
	for _, credential := range []Credential{
		{APIKey: "api-key", SecretKey: hmacSecret, SignatureAlgorithm: SignatureAlgorithmHMACSHA256},
		{APIKey: "api-key", SecretKey: ed25519Secret, SignatureAlgorithm: SignatureAlgorithmEd25519},
	} {
		auth, err := SignRequest(k4k3ruSDKJSONRPC.MethodMarketHubAggregationSubscribe, json.RawMessage(`{"symbol":"BTC/USDC"}`), credential, 1788019200, "nonce")
		if err != nil {
			t.Fatalf("SignRequest() error = %v", err)
		}
		if auth.APIKey != credential.APIKey || auth.Timestamp != 1788019200 || auth.Nonce != "nonce" || auth.Signature == "" {
			t.Fatalf("SignRequest() = %#v", auth)
		}
	}
}

func TestCredentialValidateRejectsInvalidFields(t *testing.T) {
	t.Parallel()

	for _, credential := range []Credential{
		{},
		{APIKey: "api-key"},
		{APIKey: "api-key", SecretKey: "secret", SignatureAlgorithm: "rsa"},
	} {
		if err := credential.Validate(); !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
			t.Fatalf("Validate() error = %v", err)
		}
	}
}
