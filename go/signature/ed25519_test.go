package signature

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func TestSignEd25519(t *testing.T) {
	t.Parallel()

	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	encodedPrivateKey := base64.RawURLEncoding.EncodeToString(privateKey)
	payload := []byte("payload")

	got, err := SignEd25519(encodedPrivateKey, payload)
	if err != nil {
		t.Fatalf("SignEd25519() error = %v", err)
	}
	decodedSignature, err := base64.RawURLEncoding.DecodeString(got)
	if err != nil {
		t.Fatalf("signature is not unpadded Base64 URL: %v", err)
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	if !ed25519.Verify(publicKey, payload, decodedSignature) {
		t.Fatal("SignEd25519() produced an invalid signature")
	}
}
