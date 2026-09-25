// Package jsonobject decodes strict SDK request objects without server dependencies.
package jsonobject

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// Decode decodes one object, rejecting duplicate, unknown and missing required fields.
// Required fields must be present and non-null; callers validate their values.
//
// Version:
//   - 2026-09-25: Added.
func Decode(data []byte, destination any, required ...string) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return invalid("object=invalid")
	}
	scanner := json.NewDecoder(bytes.NewReader(data))
	scanner.UseNumber()
	if err := scan(scanner, 0); err != nil {
		return fmt.Errorf("failed to decode json object: %w", err)
	}
	if _, err := scanner.Token(); err != io.EOF {
		if err != nil {
			return decodingError(err)
		}
		return invalid("trailing_value=invalid")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return decodingError(err)
	}
	for _, key := range required {
		value, exists := fields[key]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("failed to decode json object: %w: required_field=%q", apperror.InvalidParameter(), key)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		// Preserve the error chain without including a possibly sensitive JSON payload.
		return fmt.Errorf("failed to decode json object: %w: %w", apperror.InvalidParameter(), err)
	}
	return nil
}

func scan(d *json.Decoder, depth int) error {
	if depth > 128 {
		return invalid("nesting=out_of_range")
	}
	token, err := d.Token()
	if err != nil {
		return decodingError(err)
	}
	delimiter, container := token.(json.Delim)
	if !container {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]bool)
		for d.More() {
			token, err := d.Token()
			if err != nil {
				return decodingError(err)
			}
			key, ok := token.(string)
			if !ok || seen[strings.ToLower(key)] {
				return invalid("duplicate_field=invalid")
			}
			seen[strings.ToLower(key)] = true
			if err := scan(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := scan(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return invalid("delimiter=invalid")
	}
	if _, err := d.Token(); err != nil {
		return decodingError(err)
	}
	return nil
}

func invalid(state string) error {
	return fmt.Errorf("failed to inspect json object: %w: %s", apperror.InvalidParameter(), state)
}

func decodingError(err error) error {
	return fmt.Errorf("failed to decode json object: %w: %w", apperror.InvalidParameter(), err)
}
