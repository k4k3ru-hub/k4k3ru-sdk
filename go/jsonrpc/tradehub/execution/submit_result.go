package execution

import (
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

type SubmitResult struct {
	ExecutionID   string      `json:"executionId"`
	ChainFamily   ChainFamily `json:"chainFamily"`
	TransactionID string      `json:"transactionId"`
	SubmittedAt   int64       `json:"submittedAt"`
}

// Validate validates an execution submission result.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-11: Added.
func (r SubmitResult) Validate() error {
	if strings.TrimSpace(r.ExecutionID) == "" {
		return invalidSubmitResult("execution_id=empty")
	}
	switch r.ChainFamily {
	case ChainFamilyEVM, ChainFamilySui, ChainFamilySolana:
	default:
		return invalidSubmitResult("chain_family=invalid")
	}
	if strings.TrimSpace(r.TransactionID) == "" {
		return invalidSubmitResult("transaction_id=empty")
	}
	if r.SubmittedAt <= 0 {
		return invalidSubmitResult("submitted_at=out_of_range")
	}
	return nil
}

func invalidSubmitResult(reason string) error {
	return k4k3ruSDKAppError.Tracef("failed to validate trade hub execution submission result: %w: %s", k4k3ruSDKAppError.InvalidParameter(), reason)
}
