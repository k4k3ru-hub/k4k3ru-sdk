package scalping

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// MarshalJSON encodes normalized, validated settings suitable for persistence.
// Zero-return TP/SL values are omitted; explicit zero controls are preserved.
//
// Version:
//   - 2026-10-01: Added.
//   - 2026-10-03: Persist product-specific order and position limits.
func (p RunConfiguration) MarshalJSON() ([]byte, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("failed to encode scalping run configuration: %w", err)
	}
	type wire RunConfiguration
	return json.Marshal(wire(p))
}

// UnmarshalJSON decodes strict settings and defaults only omitted controls.
// Failure leaves the receiver unchanged.
//
// Version:
//   - 2026-10-01: Added.
//   - 2026-10-03: Validate and default product-specific position settings.
func (p *RunConfiguration) UnmarshalJSON(data []byte) error {
	if p == nil {
		return v.Invalid("decode scalping run configuration", "destination", "null")
	}
	type wire RunConfiguration
	var decoded wire
	if err := decodeRunJSON(data, &decoded, "marketType", "symbol", "observation", "executionRule"); err != nil {
		return fmt.Errorf("failed to decode scalping run configuration: %w", err)
	}
	value := RunConfiguration(decoded).Normalize()
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping run configuration: %w", err)
	}
	*p = value
	return nil
}

// MarshalJSON encodes either flat start parameters or an execution-ID-only resume.
//
// Version:
//   - 2026-10-01: Added.
//   - 2026-10-03: Persist Perpetual position limits without default order counts.
func (p RunParams) MarshalJSON() ([]byte, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("failed to encode scalping run parameters: %w", err)
	}
	if p.ExecutionID != "" {
		return json.Marshal(struct {
			ExecutionID string `json:"executionId"`
		}{p.ExecutionID})
	}
	// Avoid promoting RunConfiguration.MarshalJSON through embedding.
	type configuration RunConfiguration
	return json.Marshal(struct {
		IdempotencyKey string `json:"idempotencyKey"`
		configuration
	}{p.IdempotencyKey, configuration(*p.Params)})
}

// UnmarshalJSON validates a flat start or reference-only resume atomically.
// Null values, unknown or duplicate fields and resume overrides are rejected.
//
// Version:
//   - 2026-10-01: Added.
//   - 2026-10-03: Reject Perpetual order counts and validate position settings.
func (p *RunParams) UnmarshalJSON(data []byte) error {
	const op = "decode scalping run parameters"
	if p == nil {
		return v.Invalid(op, "destination", "null")
	}
	var fields map[string]json.RawMessage
	if err := decodeRunJSON(data, &fields); err != nil {
		return fmt.Errorf("failed to decode scalping run parameters: %w", err)
	}
	var value RunParams
	if _, resume := fields["executionId"]; resume {
		var reference struct {
			ExecutionID string `json:"executionId"`
		}
		if err := decodeRunJSON(data, &reference, "executionId"); err != nil {
			return fmt.Errorf("failed to decode scalping run parameters: %w", err)
		}
		value.ExecutionID = reference.ExecutionID
		if strings.TrimSpace(value.ExecutionID) == "" {
			return v.Invalid(op, "execution_id", "empty")
		}
	} else {
		type configuration RunConfiguration
		var start struct {
			IdempotencyKey string `json:"idempotencyKey"`
			configuration
		}
		if err := decodeRunJSON(data, &start, "idempotencyKey", "marketType", "symbol", "observation", "executionRule"); err != nil {
			return fmt.Errorf("failed to decode scalping run parameters: %w", err)
		}
		config := RunConfiguration(start.configuration)
		value.IdempotencyKey, value.Params = start.IdempotencyKey, &config
	}
	value = value.Normalize()
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping run parameters: %w", err)
	}
	*p = value
	return nil
}

func decodeRunJSON(data []byte, destination any, required ...string) error {
	if err := v.Decode(data, destination, required...); err != nil {
		return fmt.Errorf("failed to decode scalping run json: %w", err)
	}
	// Optional pointers distinguish omission from explicit zero. Reject explicit
	// null throughout the request, including bounds inside shared child types.
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to decode scalping run json: %w: %w", apperror.InvalidParameter(), err)
		}
		if token == nil {
			return v.Invalid("decode scalping run json", "field", "null")
		}
	}
}
