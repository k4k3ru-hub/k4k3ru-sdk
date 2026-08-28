package apperror

import (
	"errors"
	"regexp"
	"testing"
)

func TestTracef(t *testing.T) {
	t.Parallel()

	err := Tracef("failed to perform operation: %w", InvalidParameter())
	if !errors.Is(err, InvalidParameter()) {
		t.Fatalf("errors.Is() = false: error=%v", err)
	}

	pattern := `^trace_test\.go:[0-9]+: failed to perform operation: err_code="invalid_parameter"$`
	if !regexp.MustCompile(pattern).MatchString(err.Error()) {
		t.Fatalf("Tracef() error = %q, want pattern %q", err.Error(), pattern)
	}
}
