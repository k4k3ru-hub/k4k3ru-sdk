package signature

import (
	"crypto/ed25519"
	"encoding/base64"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// SignEd25519 signs a payload using an unpadded Base64 URL-encoded private key.
//
// Parameters:
//   - privateKey: unpadded Base64 URL-encoded Ed25519 private key.
//   - payload: canonical signature payload.
//
// Returns:
//   - Unpadded Base64 URL-encoded signature.
//   - Validation or signing error.
//
// Version:
//   - 2026-08-28: Added.
func SignEd25519(privateKey string, payload []byte) (string, error) {
	if privateKey == "" {
		return "", apperror.Tracef("failed to sign ed25519 payload: %w: private_key=empty", apperror.InvalidParameter())
	}
	if len(payload) == 0 {
		return "", apperror.Tracef("failed to sign ed25519 payload: %w: payload=empty", apperror.InvalidParameter())
	}

	decodedPrivateKey, err := base64.RawURLEncoding.DecodeString(privateKey)
	if err != nil {
		return "", apperror.Tracef("failed to sign ed25519 payload: private_key=invalid: %w", err)
	}
	if len(decodedPrivateKey) != ed25519.PrivateKeySize {
		return "", apperror.Tracef(
			"failed to sign ed25519 payload: %w: private_key=invalid actual_length=%d expected_length=%d",
			apperror.InvalidParameter(),
			len(decodedPrivateKey),
			ed25519.PrivateKeySize,
		)
	}

	signature := ed25519.Sign(ed25519.PrivateKey(decodedPrivateKey), payload)
	return base64.RawURLEncoding.EncodeToString(signature), nil
}
