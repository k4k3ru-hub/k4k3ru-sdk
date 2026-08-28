package signature

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

const nonceSizeBytes = 16

// GenerateNonce generates a cryptographically random request nonce.
//
// Returns:
//   - Hex-encoded 128-bit nonce.
//   - Randomness-source error.
//
// Version:
//   - 2026-08-28: Added.
func GenerateNonce() (string, error) {
	value := make([]byte, nonceSizeBytes)
	if _, err := rand.Read(value); err != nil {
		return "", apperror.Tracef("failed to generate request nonce: %w", err)
	}

	return hex.EncodeToString(value), nil
}
