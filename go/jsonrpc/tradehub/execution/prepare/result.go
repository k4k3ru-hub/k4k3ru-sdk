package prepare

import (
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/swap"
)

type Result struct {
	Kind   Kind                `json:"kind"`
	Swap   *swap.PrepareResult `json:"swap,omitempty"`
	Action *ActionPreparation  `json:"action,omitempty"`
}

type ActionPreparation struct {
	Venue   string `json:"venue"`
	Network string `json:"network"`
	// Payload contains venue-specific data to verify locally before signing.
	Payload      json.RawMessage        `json:"payload"`
	SubmitParams execution.SubmitParams `json:"submitParams"`
	PreparedAt   int64                  `json:"preparedAt"` // Unix microseconds, as for swaps.
	ExpiresAt    int64                  `json:"expiresAt"`
}

// Validate checks result variants and the unsigned submission reference.
//
// Version:
//   - 2026-09-29: Added.
func (r Result) Validate() error {
	switch r.Kind {
	case KindSwap:
		if r.Swap == nil || r.Action != nil {
			return invalid("result")
		}
		return r.Swap.Validate()
	case KindPerpetual:
		if r.Swap != nil || r.Action == nil {
			return invalid("result")
		}
		a := r.Action
		if a.Venue == "" || a.Network == "" || !json.Valid(a.Payload) || a.PreparedAt <= 0 || a.ExpiresAt <= a.PreparedAt {
			return invalid("action")
		}
		if a.SubmitParams.PreparedToken == "" || a.SubmitParams.SignedPayload != nil || a.SubmitParams.OpenExecutionID != "" {
			return invalid("submit_params")
		}
		if err := a.SubmitParams.ValidateReference(); err != nil {
			return fmt.Errorf("failed to validate execution preparation result: %w", err)
		}
		return nil
	default:
		return invalid("kind")
	}
}
