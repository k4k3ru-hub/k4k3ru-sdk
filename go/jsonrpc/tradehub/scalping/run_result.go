package scalping

import (
	"encoding/json"
	"fmt"
	"strings"

	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// RunResult acknowledges a durable configuration and its connection-local stream.
// Params contains the saved configuration on both start and resume. The ACK
// does not assert that observations are ready or an order has been submitted.
type RunResult struct {
	ExecutionID     string            `json:"executionId"`
	SubscriptionKey string            `json:"subscriptionKey"`
	Params          *RunConfiguration `json:"params"`
}

// Normalize returns a detached acknowledgement with canonical saved settings.
//
// Version:
//   - 2026-10-01: Added.
func (r RunResult) Normalize() RunResult {
	r.ExecutionID = strings.TrimSpace(r.ExecutionID)
	r.SubscriptionKey = strings.TrimSpace(r.SubscriptionKey)
	if r.Params != nil {
		settings := r.Params.Normalize()
		r.Params = &settings
	}
	return r
}

// Validate requires the durable ID, stream key and the acknowledged configuration.
//
// Version:
//   - 2026-10-01: Added.
func (r RunResult) Validate() error {
	r = r.Normalize()
	if err := validateSubscriptionReference("validate scalping run acknowledgement", r.ExecutionID, r.SubscriptionKey); err != nil {
		return err
	}
	if r.Params == nil {
		return v.Invalid("validate scalping run acknowledgement", "parameters", "null")
	}
	if err := r.Params.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping run acknowledgement: %w", err)
	}
	return nil
}

// MarshalJSON encodes a validated acknowledgement with a detached configuration.
//
// Version:
//   - 2026-10-01: Added.
func (r RunResult) MarshalJSON() ([]byte, error) {
	r = r.Normalize()
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("failed to encode scalping run acknowledgement: %w", err)
	}
	type wire RunResult
	encoded, err := json.Marshal(wire(r))
	if err != nil {
		return nil, fmt.Errorf("failed to encode scalping run acknowledgement: %w", err)
	}
	return encoded, nil
}

// UnmarshalJSON decodes a complete acknowledgement without mutating on failure.
//
// Version:
//   - 2026-10-01: Added.
func (r *RunResult) UnmarshalJSON(data []byte) error {
	if r == nil {
		return v.Invalid("decode scalping run acknowledgement", "destination", "null")
	}
	type wire RunResult
	var decoded wire
	if err := decodeRunJSON(data, &decoded, "executionId", "subscriptionKey", "params"); err != nil {
		return fmt.Errorf("failed to decode scalping run acknowledgement: %w", err)
	}
	value := RunResult(decoded).Normalize()
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping run acknowledgement: %w", err)
	}
	*r = value
	return nil
}
