package validation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
	"unicode"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// Invalid returns an inspectable validation error without including input values.
//
// Version:
//   - 2026-09-23: Added.
func Invalid(operation, field, state string) error {
	return fmt.Errorf("failed to %s: %w: %s=%s", operation, apperror.InvalidParameter(), field, state)
}

// Text validates the length of an identifier without displaying its contents.
//
// Version:
//   - 2026-09-23: Added.
func Text(operation, field, value string, maximum int) error {
	if strings.TrimSpace(value) == "" {
		return Invalid(operation, field, "empty")
	}
	if len(value) > maximum {
		return Invalid(operation, field, "too_long")
	}
	if value != strings.TrimSpace(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return Invalid(operation, field, "invalid")
	}
	return nil
}

// Number parses bounded base-ten notation without exponents or floating point.
//
// Version:
//   - 2026-09-23: Added.
func Number(operation, field, value string, integer, signed bool) (*big.Rat, error) {
	if err := Text(operation, field, value, 384); err != nil {
		return nil, err
	}
	lexeme := value
	if signed && strings.HasPrefix(lexeme, "-") {
		lexeme = lexeme[1:]
	}
	parts := strings.Split(lexeme, ".")
	if len(parts) > 2 || (integer && len(parts) != 1) {
		return nil, Invalid(operation, field, "invalid")
	}
	for _, part := range parts {
		if part == "" {
			return nil, Invalid(operation, field, "invalid")
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return nil, Invalid(operation, field, "invalid")
			}
		}
	}
	number, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, Invalid(operation, field, "invalid")
	}
	return number, nil
}

// StringPointer copies an optional string without aliasing the input.
//
// Version:
//   - 2026-09-23: Added.
func StringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	copy := strings.TrimSpace(*value)
	return &copy
}

// Pointer copies an optional scalar without aliasing the input.
//
// Version:
//   - 2026-09-23: Added.
func Pointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// Decode decodes a single object, rejecting unknown and duplicate fields.
// Required fields must be present and non-null; semantic checks belong to DTOs.
//
// Version:
//   - 2026-09-23: Added.
func Decode(data []byte, destination any, required ...string) error {
	const operation = "decode trade hub json"
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return Invalid(operation, "object", "invalid")
	}
	scanner := json.NewDecoder(bytes.NewReader(trimmed))
	scanner.UseNumber()
	if err := scanValue(scanner, 0); err != nil {
		return fmt.Errorf("failed to decode trade hub json: %w", err)
	}
	if _, err := scanner.Token(); err != io.EOF {
		if err != nil {
			return fmt.Errorf("failed to decode trade hub json: %w: %w", apperror.InvalidParameter(), err)
		}
		return Invalid(operation, "trailing_value", "invalid")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return fmt.Errorf("failed to decode trade hub json: %w: %w", apperror.InvalidParameter(), err)
	}
	for _, field := range required {
		value, ok := fields[field]
		if !ok {
			return Invalid(operation, field, "empty")
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return Invalid(operation, field, "null")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("failed to decode trade hub json: %w: %w", apperror.InvalidParameter(), err)
	}
	return nil
}

func scanValue(decoder *json.Decoder, depth int) error {
	if depth > 128 {
		return Invalid("inspect trade hub json", "nesting", "out_of_range")
	}
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to inspect trade hub json: %w: %w", apperror.InvalidParameter(), err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	if delimiter == '{' {
		seen := make(map[string]struct{})
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("failed to inspect trade hub json: %w: %w", apperror.InvalidParameter(), err)
			}
			name, ok := key.(string)
			if !ok {
				return Invalid("inspect trade hub json", "field", "invalid")
			}
			name = strings.ToLower(name)
			if _, duplicate := seen[name]; duplicate {
				return Invalid("inspect trade hub json", "duplicate_field", "invalid")
			}
			seen[name] = struct{}{}
			if err := scanValue(decoder, depth+1); err != nil {
				return err
			}
		}
	} else if delimiter == '[' {
		for decoder.More() {
			if err := scanValue(decoder, depth+1); err != nil {
				return err
			}
		}
	} else {
		return Invalid("inspect trade hub json", "delimiter", "invalid")
	}
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("failed to inspect trade hub json: %w: %w", apperror.InvalidParameter(), err)
	}
	return nil
}
