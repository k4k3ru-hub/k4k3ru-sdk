package signature

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
)

// BuildPayload builds the canonical K4K3RU request-signature payload.
//
// Parameters:
//   - method: JSON-RPC method.
//   - timestamp: request timestamp in Unix seconds.
//   - nonce: unique request nonce.
//   - params: raw JSON request parameters; an omitted value is canonicalized as null.
//
// Returns:
//   - Canonical signature payload bytes.
//   - Validation or canonicalization error.
//
// Version:
//   - 2026-08-28: Added.
func BuildPayload(method jsonrpc.Method, timestamp int64, nonce string, params []byte) ([]byte, error) {
	if err := method.Validate(); err != nil {
		return nil, apperror.Tracef("failed to build signature payload: %w", err)
	}
	if timestamp <= 0 {
		return nil, apperror.Tracef(
			"failed to build signature payload: %w: timestamp=out_of_range min_value=1",
			apperror.InvalidParameter(),
		)
	}
	if nonce == "" {
		return nil, apperror.Tracef("failed to build signature payload: %w: nonce=empty", apperror.InvalidParameter())
	}

	canonicalParams, err := canonicalizeJSON(params)
	if err != nil {
		return nil, apperror.Tracef("failed to build signature payload: %w", err)
	}

	payload := strings.Join([]string{
		string(method),
		strconv.FormatInt(timestamp, 10),
		nonce,
		string(canonicalParams),
	}, "\n")

	return []byte(payload), nil
}

func canonicalizeJSON(data []byte) ([]byte, error) {
	if len(data) == 0 {
		data = []byte("null")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, apperror.Tracef("failed to canonicalize json: %w", err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, apperror.Tracef(
				"failed to canonicalize json: %w: json=invalid",
				apperror.InvalidParameter(),
			)
		}
		return nil, apperror.Tracef("failed to canonicalize json: %w", err)
	}

	var buffer bytes.Buffer
	if err := encodeCanonical(&buffer, value); err != nil {
		return nil, apperror.Tracef("failed to canonicalize json: %w", err)
	}

	return buffer.Bytes(), nil
}

func encodeCanonical(writer io.Writer, value any) error {
	switch typedValue := value.(type) {
	case nil:
		_, err := io.WriteString(writer, "null")
		return err
	case bool:
		_, err := io.WriteString(writer, strconv.FormatBool(typedValue))
		return err
	case json.Number:
		_, err := io.WriteString(writer, typedValue.String())
		return err
	case string:
		encoded, err := json.Marshal(typedValue)
		if err != nil {
			return apperror.Tracef("failed to encode canonical json string: %w", err)
		}
		_, err = writer.Write(encoded)
		return err
	case []any:
		if _, err := io.WriteString(writer, "["); err != nil {
			return err
		}
		for index, item := range typedValue {
			if index > 0 {
				if _, err := io.WriteString(writer, ","); err != nil {
					return err
				}
			}
			if err := encodeCanonical(writer, item); err != nil {
				return err
			}
		}
		_, err := io.WriteString(writer, "]")
		return err
	case map[string]any:
		keys := make([]string, 0, len(typedValue))
		for key := range typedValue {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		if _, err := io.WriteString(writer, "{"); err != nil {
			return err
		}
		for index, key := range keys {
			if index > 0 {
				if _, err := io.WriteString(writer, ","); err != nil {
					return err
				}
			}
			encodedKey, err := json.Marshal(key)
			if err != nil {
				return apperror.Tracef("failed to encode canonical json object key: %w", err)
			}
			if _, err := writer.Write(encodedKey); err != nil {
				return err
			}
			if _, err := io.WriteString(writer, ":"); err != nil {
				return err
			}
			if err := encodeCanonical(writer, typedValue[key]); err != nil {
				return err
			}
		}
		_, err := io.WriteString(writer, "}")
		return err
	default:
		return fmt.Errorf("failed to encode canonical json: value=invalid")
	}
}
