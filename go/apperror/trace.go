package apperror

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// Tracef formats an error and adds the caller's source location.
//
// Parameters:
//   - format: error format string.
//   - args: error format arguments.
//
// Returns:
//   - Formatted error with the caller's source location when available.
//
// Version:
//   - 2026-08-28: Added.
func Tracef(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return err
	}

	return fmt.Errorf("%s:%d: %w", filepath.Base(file), line, err)
}
