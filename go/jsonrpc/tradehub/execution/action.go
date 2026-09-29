package execution

import (
	"encoding/json"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// SignedAction carries a venue signature; the authenticated prepared token owns the action.
type SignedAction struct {
	Venue     string          `json:"venue"`
	Network   string          `json:"network"`
	Signature json.RawMessage `json:"signature"`
}

type ActionReceipt struct {
	Venue                  string `json:"venue"`
	Network                string `json:"network"`
	Kind                   string `json:"kind"`
	Status                 string `json:"status"`
	OrderID                string `json:"orderId,omitempty"`
	ClientOrderID          string `json:"clientOrderId,omitempty"`
	RequestedSize          string `json:"requestedSize,omitempty"`
	FilledSize             string `json:"filledSize,omitempty"`
	AveragePrice           string `json:"averagePrice,omitempty"`
	ReconciliationRequired bool   `json:"reconciliationRequired"`
}

// Validate validates the signature envelope before a venue adapter checks its protocol.
//
// Version:
//   - 2026-09-29: Added.
func (a SignedAction) Validate() error {
	for _, f := range []struct{ name, value string }{{"venue", a.Venue}, {"network", a.Network}} {
		if err := validation.Text("validate signed action", f.name, f.value, 64); err != nil {
			return err
		}
	}
	if len(a.Signature) > 4096 {
		return invalidSignedPayloadParameterError("signature=too_long")
	}
	var fields map[string]json.RawMessage
	if err := validation.Decode(a.Signature, &fields); err != nil {
		return &DecodeError{cause: err}
	}
	if len(fields) == 0 {
		return invalidSignedPayloadParameterError("signature=empty")
	}
	return nil
}

// DecodeError retains parsing errors without disclosing signed payloads or tokens.
type DecodeError struct{ cause error }

// Error returns a safe decoding diagnostic.
//
// Version:
//   - 2026-09-29: Added.
func (e *DecodeError) Error() string { return "failed to decode execution parameters: json=invalid" }

// Unwrap preserves the underlying error for inspection.
//
// Version:
//   - 2026-09-29: Added.
func (e *DecodeError) Unwrap() error { return e.cause }
