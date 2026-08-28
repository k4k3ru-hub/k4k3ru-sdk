package apperror

import "fmt"

const maxCodeLength = 64

type Code string

const (
	CodeAlreadyCreated   Code = "already_created"
	CodeConflict         Code = "conflict"
	CodeExpired          Code = "expired"
	CodeForbidden        Code = "forbidden"
	CodeInternal         Code = "internal_error"
	CodeInvalidParameter Code = "invalid_parameter"
	CodeInvalidRequest   Code = "invalid_request"
	CodeNotFound         Code = "not_found"
	CodeOutOfTicks       Code = "out_of_ticks"
	CodeUnauthorized     Code = "unauthorized"
	CodeUnexpected       Code = "unexpected"
	CodeUnsupported      Code = "unsupported"
)

// Validate validates the application error code.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-28: Added.
func (c Code) Validate() error {
	if c == "" {
		return fmt.Errorf("failed to validate application error code: %w: code=empty", InvalidParameter())
	}
	if len(c) > maxCodeLength {
		return fmt.Errorf(
			"failed to validate application error code: %w: code=too_long actual_length=%d max_length=%d",
			InvalidParameter(),
			len(c),
			maxCodeLength,
		)
	}

	return nil
}
