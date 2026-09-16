package execution

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"

	apperror "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

type SubscribeParams struct {
	ExecutionID string `json:"executionId"`
}
type SubscribeResult struct {
	ExecutionID     string `json:"executionId"`
	SubscriptionKey string `json:"subscriptionKey"`
}
type UnsubscribeParams struct {
	ExecutionID     string `json:"executionId"`
	SubscriptionKey string `json:"subscriptionKey"`
}
type ObservationStatus string

const (
	ObservationStatusPending ObservationStatus = "pending"
	ObservationStatusSuccess ObservationStatus = "success"
	ObservationStatusFailed  ObservationStatus = "failed"
)

type ExecutionSnapshot struct {
	Status     ObservationStatus `json:"status"`
	ObservedAt int64             `json:"observedAt"`
	Onchain    *OnchainExecution `json:"onchain,omitempty"`
	Failure    *ExecutionFailure `json:"failure,omitempty"`
}
type OnchainExecution struct {
	ChainFamily   ChainFamily `json:"chainFamily"`
	Chain         string      `json:"chain"`
	Network       string      `json:"network"`
	TransactionID string      `json:"transactionId"`
	BlockNumber   *uint64     `json:"blockNumber,omitempty"`
	BlockHash     string      `json:"blockHash,omitempty"`
}
type ExecutionFailure struct {
	Code string `json:"code"`
}
type ObservationError struct {
	Code      string `json:"code"`
	Retryable bool   `json:"retryable"`
}
type ExecutionEventKind string

const (
	ExecutionEventSnapshot ExecutionEventKind = "snapshot"
	ExecutionEventError    ExecutionEventKind = "error"
)

type SubscriptionEvent struct {
	ExecutionID     string             `json:"executionId"`
	SubscriptionKey string             `json:"subscriptionKey"`
	Sequence        uint64             `json:"sequence"`
	Kind            ExecutionEventKind `json:"kind"`
	Snapshot        *ExecutionSnapshot `json:"snapshot,omitempty"`
	Error           *ObservationError  `json:"error,omitempty"`
}

// Normalize trims the execution identifier without changing its case.
//
// Version:
//   - 2026-09-16: Added.
func (p SubscribeParams) Normalize() SubscribeParams {
	p.ExecutionID = strings.TrimSpace(p.ExecutionID)
	return p
}

// Validate validates a service-issued execution identifier.
//
// Version:
//   - 2026-09-16: Added.
func (p SubscribeParams) Validate() error {
	return validateObservationID(strings.TrimSpace(p.ExecutionID), "execution_id")
}

// Normalize trims the watch identifiers without changing their case.
//
// Version:
//   - 2026-09-16: Added.
func (p UnsubscribeParams) Normalize() UnsubscribeParams {
	p.ExecutionID = strings.TrimSpace(p.ExecutionID)
	p.SubscriptionKey = strings.TrimSpace(p.SubscriptionKey)
	return p
}

// Validate validates both watch identifiers.
//
// Version:
//   - 2026-09-16: Added.
func (p UnsubscribeParams) Validate() error {
	p = p.Normalize()
	if err := validateObservationID(p.ExecutionID, "execution_id"); err != nil {
		return err
	}
	return validateObservationID(p.SubscriptionKey, "subscription_key")
}

// Validate validates a subscription acknowledgement.
//
// Version:
//   - 2026-09-16: Added.
func (p SubscribeResult) Validate() error { return (UnsubscribeParams(p)).Validate() }

// Terminal reports whether receipt inclusion has completed this observation.
// It does not imply chain finality.
//
// Version:
//   - 2026-09-16: Added.
func (s ObservationStatus) Terminal() bool {
	return s == ObservationStatusSuccess || s == ObservationStatusFailed
}

// Validate validates an initial single-transaction EVM observation event.
//
// Version:
//   - 2026-09-16: Added.
func (e SubscriptionEvent) Validate() error {
	if err := (UnsubscribeParams{e.ExecutionID, e.SubscriptionKey}).Validate(); err != nil {
		return err
	}
	if e.Sequence == 0 {
		return observationInvalid("sequence=empty")
	}
	if e.Kind == ExecutionEventError {
		if e.Snapshot != nil || e.Error == nil || e.Error.Code != "observation_unavailable" {
			return observationInvalid("error=invalid")
		}
		return nil
	}
	if e.Kind != ExecutionEventSnapshot || e.Snapshot == nil || e.Error != nil {
		return observationInvalid("event=invalid")
	}
	s := e.Snapshot
	if s.ObservedAt <= 0 || s.Onchain == nil {
		return observationInvalid("snapshot=invalid")
	}
	o := s.Onchain
	if o.ChainFamily != ChainFamilyEVM || o.Chain != "base" || (o.Network != "mainnet" && o.Network != "sepolia") || !observationHash(o.TransactionID) {
		return observationInvalid("onchain=invalid")
	}
	if s.Status == ObservationStatusPending {
		if o.BlockNumber != nil || o.BlockHash != "" || s.Failure != nil {
			return observationInvalid("pending=invalid")
		}
		return nil
	}
	if !s.Status.Terminal() || o.BlockNumber == nil || !observationHash(o.BlockHash) {
		return observationInvalid("receipt=invalid")
	}
	if s.Status == ObservationStatusFailed {
		if s.Failure == nil || s.Failure.Code != "transaction_reverted" {
			return observationInvalid("failure=invalid")
		}
	} else if s.Failure != nil {
		return observationInvalid("failure=invalid")
	}
	return nil
}

func validateObservationID(value, field string) error {
	if value == "" {
		return observationInvalid(field + "=empty")
	}
	if len(value) > 64 {
		return observationInvalid(field + "=too_long")
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return observationInvalid(field + "=invalid")
		}
	}
	return nil
}
func observationHash(value string) bool {
	if len(value) != 66 || !strings.HasPrefix(value, "0x") {
		return false
	}
	_, err := hex.DecodeString(value[2:])
	return err == nil
}
func observationInvalid(detail string) error {
	return apperror.Tracef("failed to validate execution observation: %w: %s", apperror.InvalidParameter(), detail)
}
func decodeObservation(data []byte, result any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(result); err != nil {
		return observationInvalid("json=invalid")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return observationInvalid("json=invalid")
	}
	return nil
}

// UnmarshalJSON decodes subscription parameters and rejects unknown fields.
//
// Version:
//   - 2026-09-16: Added.
func (p *SubscribeParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return observationInvalid("destination=null")
	}
	type wire SubscribeParams
	var v wire
	if err := decodeObservation(data, &v); err != nil {
		return err
	}
	*p = SubscribeParams(v)
	return p.Validate()
}

// UnmarshalJSON decodes unsubscription parameters and rejects unknown fields.
//
// Version:
//   - 2026-09-16: Added.
func (p *UnsubscribeParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return observationInvalid("destination=null")
	}
	type wire UnsubscribeParams
	var v wire
	if err := decodeObservation(data, &v); err != nil {
		return err
	}
	*p = UnsubscribeParams(v)
	return p.Validate()
}

// UnmarshalJSON decodes an observation event and rejects unknown fields.
//
// Version:
//   - 2026-09-16: Added.
func (p *SubscriptionEvent) UnmarshalJSON(data []byte) error {
	if p == nil {
		return observationInvalid("destination=null")
	}
	type wire SubscriptionEvent
	var v wire
	if err := decodeObservation(data, &v); err != nil {
		return err
	}
	*p = SubscriptionEvent(v)
	return p.Validate()
}

// UnmarshalJSON decodes a subscription acknowledgement and rejects unknown fields.
//
// Version:
//   - 2026-09-16: Added.
func (p *SubscribeResult) UnmarshalJSON(data []byte) error {
	if p == nil {
		return observationInvalid("destination=null")
	}
	type wire SubscribeResult
	var v wire
	if err := decodeObservation(data, &v); err != nil {
		return err
	}
	*p = SubscribeResult(v)
	return p.Validate()
}
