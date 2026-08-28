package apperror

import (
	"encoding/json"
	"errors"
	"fmt"
)

type AppError struct {
	code    Code
	message string
}

// New creates an application error.
//
// Parameters:
//   - code: machine-readable application error code.
//   - message: optional human-readable message.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func New(code Code, message string) *AppError {
	return &AppError{code: code, message: message}
}

// AlreadyCreated returns an already-created application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func AlreadyCreated() *AppError { return New(CodeAlreadyCreated, "") }

// Conflict returns a conflict application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func Conflict() *AppError { return New(CodeConflict, "") }

// Expired returns an expired application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func Expired() *AppError { return New(CodeExpired, "") }

// Forbidden returns a forbidden application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func Forbidden() *AppError { return New(CodeForbidden, "") }

// Internal returns an internal application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func Internal() *AppError { return New(CodeInternal, "") }

// InvalidParameter returns an invalid-parameter application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func InvalidParameter() *AppError { return New(CodeInvalidParameter, "") }

// InvalidRequest returns an invalid-request application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func InvalidRequest() *AppError { return New(CodeInvalidRequest, "") }

// NotFound returns a not-found application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func NotFound() *AppError { return New(CodeNotFound, "") }

// OutOfTicks returns an out-of-ticks application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func OutOfTicks() *AppError { return New(CodeOutOfTicks, "") }

// Unauthorized returns an unauthorized application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func Unauthorized() *AppError { return New(CodeUnauthorized, "") }

// Unexpected returns an unexpected application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func Unexpected() *AppError { return New(CodeUnexpected, "") }

// Unsupported returns an unsupported application error.
//
// Returns:
//   - Application error.
//
// Version:
//   - 2026-08-28: Added.
func Unsupported() *AppError { return New(CodeUnsupported, "") }

// AsAppError extracts an application error from an error chain.
//
// Parameters:
//   - err: error chain to inspect.
//
// Returns:
//   - Application error, or nil when the chain does not contain one.
//
// Version:
//   - 2026-08-28: Added.
func AsAppError(err error) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if !errors.As(err, &appErr) {
		return nil
	}

	return appErr
}

// Normalize converts an error to an application error.
//
// Unknown and nil errors are converted to an unexpected application error.
//
// Parameters:
//   - err: error to normalize.
//
// Returns:
//   - Application error from the error chain, or an unexpected application error.
//
// Version:
//   - 2026-08-28: Added.
func Normalize(err error) *AppError {
	if err == nil {
		return Unexpected()
	}
	if appErr := AsAppError(err); appErr != nil {
		return appErr
	}

	return Unexpected()
}

// WithMessage returns a copy with the provided message.
//
// Parameters:
//   - message: human-readable message.
//
// Returns:
//   - Application error copy.
//
// Version:
//   - 2026-08-28: Added.
func (e *AppError) WithMessage(message string) *AppError {
	if e == nil {
		return nil
	}

	cloned := *e
	cloned.message = message
	return &cloned
}

// Error returns the application error string.
//
// Returns:
//   - Application error string.
//
// Version:
//   - 2026-08-28: Added.
func (e *AppError) Error() string {
	if e == nil {
		return "null"
	}

	switch {
	case e.code != "" && e.message != "":
		return fmt.Sprintf("err_code=%q err_message=%q", e.code, e.message)
	case e.code != "":
		return fmt.Sprintf("err_code=%q", e.code)
	case e.message != "":
		return fmt.Sprintf("err_message=%q", e.message)
	default:
		return "unknown_error"
	}
}

// Is reports whether the target has the same non-empty application error code.
//
// Parameters:
//   - target: error to compare.
//
// Returns:
//   - True when the application error codes match.
//
// Version:
//   - 2026-08-28: Added.
func (e *AppError) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}

	var targetError *AppError
	if !errors.As(target, &targetError) {
		return false
	}

	return e.code != "" && e.code == targetError.code
}

// Code returns the machine-readable application error code.
//
// Returns:
//   - Application error code.
//
// Version:
//   - 2026-08-28: Added.
func (e *AppError) Code() Code {
	if e == nil {
		return ""
	}
	return e.code
}

// Message returns the human-readable application error message.
//
// Returns:
//   - Application error message.
//
// Version:
//   - 2026-08-28: Added.
func (e *AppError) Message() string {
	if e == nil {
		return ""
	}
	return e.message
}

// MarshalJSON encodes the application error JSON object.
//
// Returns:
//   - Encoded JSON.
//   - Encoding error.
//
// Version:
//   - 2026-08-28: Added.
func (e *AppError) MarshalJSON() ([]byte, error) {
	type appError struct {
		Code    Code   `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}

	if e == nil {
		return []byte("null"), nil
	}

	return json.Marshal(appError{Code: e.code, Message: e.message})
}

// UnmarshalJSON decodes an application error JSON object.
//
// Parameters:
//   - data: encoded JSON.
//
// Returns:
//   - Decoding error.
//
// Version:
//   - 2026-08-28: Added.
func (e *AppError) UnmarshalJSON(data []byte) error {
	if e == nil {
		return fmt.Errorf("failed to decode application error: application_error=null")
	}

	type appError struct {
		Code    Code   `json:"code"`
		Message string `json:"message"`
	}

	var value appError
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("failed to decode application error: %w", err)
	}

	e.code = value.Code
	e.message = value.Message
	return nil
}
