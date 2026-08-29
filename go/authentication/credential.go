package authentication

import (
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// SignatureAlgorithm identifies a supported request-signature algorithm.
type SignatureAlgorithm string

const (
	SignatureAlgorithmHMACSHA256 SignatureAlgorithm = "hmac-sha256"
	SignatureAlgorithmEd25519    SignatureAlgorithm = "ed25519"
)

// Credential contains API authentication material supplied by an application.
type Credential struct {
	APIKey             string
	SecretKey          string
	SignatureAlgorithm SignatureAlgorithm
}

// Validate validates API authentication material without exposing its values.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-29: Added.
func (c Credential) Validate() error {
	if strings.TrimSpace(c.APIKey) == "" {
		return k4k3ruSDKAppError.Tracef("failed to validate authentication credential: %w: api_key=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if strings.TrimSpace(c.SecretKey) == "" {
		return k4k3ruSDKAppError.Tracef("failed to validate authentication credential: %w: secret_key=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if c.SignatureAlgorithm != SignatureAlgorithmHMACSHA256 && c.SignatureAlgorithm != SignatureAlgorithmEd25519 {
		return k4k3ruSDKAppError.Tracef("failed to validate authentication credential: %w: signature_algorithm=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	return nil
}
