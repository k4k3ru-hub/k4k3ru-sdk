package scalping

import (
	"encoding/json"
	"fmt"
	"strings"

	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// SubscribeParams starts an execution or resumes an existing one. Params is
// flattened on the wire, keeping marketType directly inside JSON-RPC params.
type SubscribeParams struct {
	ExecutionID    string  `json:"executionId,omitempty"`
	IdempotencyKey string  `json:"idempotencyKey,omitempty"`
	Params         *Params `json:"-"`
}

// Normalize copies subscription parameters without adding defaults.
//
// Version:
//   - 2026-09-24: Added.
func (p SubscribeParams) Normalize() SubscribeParams {
	p.ExecutionID = strings.TrimSpace(p.ExecutionID)
	p.IdempotencyKey = strings.TrimSpace(p.IdempotencyKey)
	if p.Params != nil {
		copy := p.Params.Normalize()
		p.Params = &copy
	}
	return p
}

// Validate requires either an execution reference or a complete idempotent start.
//
// Version:
//   - 2026-09-24: Added.
func (p SubscribeParams) Validate() error {
	p = p.Normalize()
	const op = "validate scalping subscription parameters"
	if p.ExecutionID != "" {
		if p.Params != nil || p.IdempotencyKey != "" {
			return v.Invalid(op, "resume_overrides", "invalid")
		}
		return v.Text(op, "execution_id", p.ExecutionID, 128)
	}
	if err := v.Text(op, "idempotency_key", p.IdempotencyKey, 128); err != nil {
		return err
	}
	if p.Params == nil {
		return v.Invalid(op, "parameters", "null")
	}
	if err := p.Params.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping subscription parameters: %w", err)
	}
	return nil
}

// MarshalJSON encodes the flat start or reference-only resume contract.
//
// Version:
//   - 2026-09-24: Added.
func (p SubscribeParams) MarshalJSON() ([]byte, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("failed to encode scalping subscription parameters: %w", err)
	}
	if p.ExecutionID != "" {
		return json.Marshal(struct {
			ExecutionID string `json:"executionId"`
		}{p.ExecutionID})
	}
	// A defined type avoids promoting Params.UnmarshalJSON through embedding.
	type configuration Params
	return json.Marshal(struct {
		IdempotencyKey string `json:"idempotencyKey"`
		configuration
	}{p.IdempotencyKey, configuration(*p.Params)})
}

// UnmarshalJSON decodes a start or resume, rejecting even null resume overrides.
// Decode failure leaves the receiver unchanged.
//
// Version:
//   - 2026-09-24: Added.
func (p *SubscribeParams) UnmarshalJSON(data []byte) error {
	const op = "decode scalping subscription parameters"
	if p == nil {
		return v.Invalid(op, "destination", "null")
	}
	var fields map[string]json.RawMessage
	if err := v.Decode(data, &fields); err != nil {
		return fmt.Errorf("failed to decode scalping subscription parameters: %w", err)
	}
	var value SubscribeParams
	if _, resume := fields["executionId"]; resume {
		var reference struct {
			ExecutionID string `json:"executionId"`
		}
		if err := v.Decode(data, &reference, "executionId"); err != nil {
			return fmt.Errorf("failed to decode scalping subscription parameters: %w", err)
		}
		value.ExecutionID = reference.ExecutionID
		if strings.TrimSpace(value.ExecutionID) == "" {
			return v.Invalid(op, "execution_id", "empty")
		}
	} else {
		key, exists := fields["idempotencyKey"]
		if !exists {
			return v.Invalid(op, "idempotency_key", "empty")
		}
		if err := json.Unmarshal(key, &value.IdempotencyKey); err != nil {
			return fmt.Errorf("failed to decode scalping subscription parameters: %w: %w", app.InvalidParameter(), err)
		}
		delete(fields, "idempotencyKey")
		encoded, err := json.Marshal(fields)
		if err != nil {
			return fmt.Errorf("failed to decode scalping subscription parameters: %w", err)
		}
		var config Params
		if err := json.Unmarshal(encoded, &config); err != nil {
			return fmt.Errorf("failed to decode scalping subscription parameters: %w", err)
		}
		value.Params = &config
	}
	value = value.Normalize()
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping subscription parameters: %w", err)
	}
	*p = value
	return nil
}
