package jsonrpc

import "fmt"

const maxMethodLength = 64

type Method string

const (
	MethodAccountEmailRequestCredentialCreationOTP Method = "AccountEmail.RequestCredentialCreationOTP"
	MethodAccountEmailCreateCredential             Method = "AccountEmail.CreateCredential"
)

// Validate validates the JSON-RPC method.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-28: Added.
func (m Method) Validate() error {
	if m == "" {
		return fmt.Errorf("failed to validate json rpc method: method=empty")
	}
	if len(m) > maxMethodLength {
		return fmt.Errorf(
			"failed to validate json rpc method: method=too_long actual_length=%d max_length=%d",
			len(m),
			maxMethodLength,
		)
	}

	return nil
}
