package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// SignHMACSHA256 signs a payload using an unpadded Base64 URL-encoded secret key.
//
// Parameters:
//   - secretKey: unpadded Base64 URL-encoded HMAC-SHA256 secret key.
//   - payload: canonical signature payload.
//
// Returns:
//   - Unpadded Base64 URL-encoded signature.
//   - Validation or signing error.
//
// Version:
//   - 2026-08-28: Added.
func SignHMACSHA256(secretKey string, payload []byte) (string, error) {
	if secretKey == "" {
		return "", apperror.Tracef("failed to sign hmac sha256 payload: %w: secret_key=empty", apperror.InvalidParameter())
	}
	if len(payload) == 0 {
		return "", apperror.Tracef("failed to sign hmac sha256 payload: %w: payload=empty", apperror.InvalidParameter())
	}

	decodedSecretKey, err := base64.RawURLEncoding.DecodeString(secretKey)
	if err != nil {
		return "", apperror.Tracef("failed to sign hmac sha256 payload: secret_key=invalid: %w", err)
	}
	if len(decodedSecretKey) == 0 {
		return "", apperror.Tracef("failed to sign hmac sha256 payload: %w: secret_key=empty", apperror.InvalidParameter())
	}

	messageAuthenticationCode := hmac.New(sha256.New, decodedSecretKey)
	if _, err := messageAuthenticationCode.Write(payload); err != nil {
		return "", apperror.Tracef("failed to sign hmac sha256 payload: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(messageAuthenticationCode.Sum(nil)), nil
}
