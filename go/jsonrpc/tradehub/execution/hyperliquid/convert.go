// Package hyperliquid translates Hyperliquid signing data to the common execution RPCs.
package hyperliquid

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution/prepare"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/perpetual"
)

// PrepareParams wraps a legacy intent with venue-specific signing inputs.
//
// Version:
//   - 2026-09-29: Added.
func PrepareParams(p perpetual.PrepareParams) prepare.Params {
	return prepare.Params{Kind: prepare.KindPerpetual, Perpetual: &prepare.PerpetualParams{
		Venue: p.Venue, Network: p.Network, Symbol: p.Symbol, AccountAddress: p.AccountAddress,
		SignerAddress: p.SignerAddress, Kind: p.Kind, Order: p.Order, Leverage: p.Leverage,
		Signing: prepare.SigningContext{Hyperliquid: &prepare.HyperliquidSigning{Nonce: p.Nonce}},
	}}
}

// Intent checks the supported Hyperliquid scope and converts the common intent.
//
// Version:
//   - 2026-09-29: Added.
func Intent(p prepare.PerpetualParams) (perpetual.PrepareParams, error) {
	if err := (prepare.Params{Kind: prepare.KindPerpetual, Perpetual: &p}).Validate(); err != nil {
		return perpetual.PrepareParams{}, fmt.Errorf("failed to convert hyperliquid intent: %w", err)
	}
	if p.Signing.Hyperliquid == nil {
		return perpetual.PrepareParams{}, invalid("signing")
	}
	v := perpetual.PrepareParams{Scope: perpetual.Scope{Venue: p.Venue, Network: p.Network, Symbol: p.Symbol, AccountAddress: p.AccountAddress}, SignerAddress: p.SignerAddress, Kind: p.Kind, Nonce: p.Signing.Hyperliquid.Nonce, Order: p.Order, Leverage: p.Leverage}
	if err := v.Validate(); err != nil {
		return perpetual.PrepareParams{}, fmt.Errorf("failed to convert hyperliquid intent: %w", err)
	}
	return v, nil
}

// PrepareResult wraps verified venue data with common submission references and microsecond times.
//
// Version:
//   - 2026-09-29: Added.
func PrepareResult(r *perpetual.PrepareResult) (*prepare.Result, error) {
	if r == nil || r.Prepared.ExpiresAfter > math.MaxInt64/1000 || r.Prepared.PreparedAt > math.MaxInt64/1000 {
		return nil, invalid("preparation")
	}
	payload, err := json.Marshal(r.Prepared)
	if err != nil {
		return nil, fmt.Errorf("failed to encode hyperliquid preparation: %w", err)
	}
	v := &prepare.Result{Kind: prepare.KindPerpetual, Action: &prepare.ActionPreparation{
		Venue: r.Prepared.Intent.Venue, Network: r.Prepared.Intent.Network, Payload: payload,
		SubmitParams: execution.SubmitParams{PreparedToken: r.PreparedToken, ExecutionID: r.Prepared.PreparationID, PayloadDigest: r.Prepared.Digest},
		PreparedAt:   int64(r.Prepared.PreparedAt) * 1000, ExpiresAt: int64(r.Prepared.ExpiresAfter) * 1000,
	}}
	if err := v.Validate(); err != nil {
		return nil, fmt.Errorf("failed to encode hyperliquid preparation: %w", err)
	}
	return v, nil
}

// Prepared checks envelope bindings before returning data for local cryptographic verification.
//
// Version:
//   - 2026-09-29: Added.
func Prepared(r *prepare.Result) (*perpetual.PrepareResult, error) {
	if r == nil {
		return nil, invalid("result")
	}
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("failed to decode hyperliquid preparation: %w", err)
	}
	if r.Kind != prepare.KindPerpetual || r.Action.Venue != "hyperliquid" || r.Action.Network != "testnet" {
		return nil, invalid("scope")
	}
	a := r.Action
	var p perpetual.Prepared
	if err := json.Unmarshal(a.Payload, &p); err != nil {
		return nil, &payloadDecodeError{cause: fmt.Errorf("failed to decode hyperliquid preparation: %w: %w", apperror.InvalidParameter(), err)}
	}
	if p.Intent.Venue != a.Venue || p.Intent.Network != a.Network || p.PreparationID != a.SubmitParams.ExecutionID || p.Digest != a.SubmitParams.PayloadDigest || p.PreparedAt > math.MaxInt64/1000 || p.ExpiresAfter > math.MaxInt64/1000 || int64(p.PreparedAt)*1000 != a.PreparedAt || int64(p.ExpiresAfter)*1000 != a.ExpiresAt {
		return nil, invalid("binding")
	}
	return &perpetual.PrepareResult{Prepared: p, PreparedToken: a.SubmitParams.PreparedToken}, nil
}

// SubmitParams converts a persisted signature without rebuilding or resigning the action.
//
// Version:
//   - 2026-09-29: Added.
func SubmitParams(p perpetual.SubmitParams) (execution.SubmitParams, error) {
	if err := p.Validate(); err != nil {
		return execution.SubmitParams{}, fmt.Errorf("failed to convert hyperliquid submission: %w", err)
	}
	signature, err := json.Marshal(p.Signature)
	if err != nil {
		return execution.SubmitParams{}, fmt.Errorf("failed to encode hyperliquid signature: %w", err)
	}
	return execution.SubmitParams{ExecutionID: p.PreparationID, PayloadDigest: p.Digest, PreparedToken: p.PreparedToken,
		SignedPayload: &execution.SignedPayload{Action: &execution.SignedAction{Venue: "hyperliquid", Network: "testnet", Signature: signature}}}, nil
}

// Submission validates the envelope before the service authenticates its token and signature.
//
// Version:
//   - 2026-09-29: Added.
func Submission(p execution.SubmitParams) (perpetual.SubmitParams, error) {
	if err := p.Validate(); err != nil {
		return perpetual.SubmitParams{}, fmt.Errorf("failed to decode hyperliquid submission: %w", err)
	}
	a := p.SignedPayload.Action
	if a == nil || a.Venue != "hyperliquid" || a.Network != "testnet" {
		return perpetual.SubmitParams{}, invalid("scope")
	}
	v := perpetual.SubmitParams{PreparationID: p.ExecutionID, Digest: p.PayloadDigest, PreparedToken: p.PreparedToken}
	if err := json.Unmarshal(a.Signature, &v.Signature); err != nil {
		return perpetual.SubmitParams{}, fmt.Errorf("failed to decode hyperliquid signature: %w", err)
	}
	if err := v.Validate(); err != nil {
		return perpetual.SubmitParams{}, fmt.Errorf("failed to decode hyperliquid submission: %w", err)
	}
	return v, nil
}

// SubmitResult converts an action acknowledgment without inventing a transaction identifier.
//
// Version:
//   - 2026-09-29: Added.
func SubmitResult(r *perpetual.SubmitResult, submittedAt int64) (*execution.SubmitResult, error) {
	if r == nil {
		return nil, invalid("receipt")
	}
	v := &execution.SubmitResult{ExecutionID: r.PreparationID, SubmittedAt: submittedAt, Action: &execution.ActionReceipt{
		Venue: "hyperliquid", Network: "testnet", Kind: r.Kind, Status: r.Status, OrderID: r.OrderID, ClientOrderID: r.ClientOrderID,
		RequestedSize: r.RequestedSize, FilledSize: r.FilledSize, AveragePrice: r.AveragePrice, ReconciliationRequired: r.ReconciliationRequired,
	}}
	if err := v.Validate(); err != nil {
		return nil, fmt.Errorf("failed to encode hyperliquid receipt: %w", err)
	}
	return v, nil
}

// Receipt converts a common acknowledgment for the existing journal and reconciliation logic.
//
// Version:
//   - 2026-09-29: Added.
func Receipt(r *execution.SubmitResult) (*perpetual.SubmitResult, error) {
	if r == nil {
		return nil, invalid("receipt")
	}
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("failed to decode hyperliquid receipt: %w", err)
	}
	a := r.Action
	if a == nil || a.Venue != "hyperliquid" || a.Network != "testnet" {
		return nil, invalid("scope")
	}
	return &perpetual.SubmitResult{PreparationID: r.ExecutionID, Kind: a.Kind, Status: a.Status, OrderID: a.OrderID, ClientOrderID: a.ClientOrderID,
		RequestedSize: a.RequestedSize, FilledSize: a.FilledSize, AveragePrice: a.AveragePrice, ReconciliationRequired: a.ReconciliationRequired}, nil
}

func invalid(field string) error {
	return fmt.Errorf("failed to convert hyperliquid execution: %w: %s=invalid", apperror.InvalidParameter(), field)
}

type payloadDecodeError struct{ cause error }

// Error describes an invalid preparation without exposing its payload.
//
// Version:
//   - 2026-09-29: Added.
func (e *payloadDecodeError) Error() string {
	return "failed to decode hyperliquid preparation: payload=invalid"
}

// Unwrap preserves the inspectable parsing error.
//
// Version:
//   - 2026-09-29: Added.
func (e *payloadDecodeError) Unwrap() error { return e.cause }
