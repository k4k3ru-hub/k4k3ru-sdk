package swap

import (
	"math/big"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	k4k3ruSDKTradeHubExecution "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

type ApprovalRequirement struct {
	Token             string `json:"token"`
	Owner             string `json:"owner"`
	Spender           string `json:"spender"`
	CurrentAllowance  string `json:"currentAllowance"`
	RequiredAllowance string `json:"requiredAllowance"`
	ApprovalAmount    string `json:"approvalAmount"`
}

type PrepareResult struct {
	ExecutionID    string                                     `json:"executionId"`
	Status         PrepareStatus                              `json:"status"`
	Chain          k4k3ruOnchainCore.Chain                    `json:"chain"`
	Network        k4k3ruOnchainCore.Network                  `json:"network"`
	AmountIn       string                                     `json:"amountIn,omitempty"`
	AmountOut      string                                     `json:"amountOut,omitempty"`
	Approval       *ApprovalRequirement                       `json:"approval,omitempty"`
	SigningPayload *k4k3ruSDKTradeHubExecution.SigningPayload `json:"signingPayload"`
	SubmitParams   *k4k3ruSDKTradeHubExecution.SubmitParams   `json:"submitParams"`
	PreparedAt     int64                                      `json:"preparedAt"`
	ExpiresAt      int64                                      `json:"expiresAt"`
}

// Validate validates a prepared swap or prerequisite approval result.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-10: Added.
func (r PrepareResult) Validate() error {
	if strings.TrimSpace(r.ExecutionID) == "" {
		return invalidPrepareResult("execution_id=empty")
	}
	if err := r.Chain.Validate(); err != nil {
		return invalidPrepareResult("chain=invalid")
	}
	if err := r.Network.Validate(); err != nil {
		return invalidPrepareResult("network=invalid")
	}
	if r.Status != PrepareStatusReady && r.Status != PrepareStatusApprovalRequired {
		return invalidPrepareResult("status=invalid")
	}
	if r.SigningPayload == nil {
		return invalidPrepareResult("signing_payload=null")
	}
	if r.SigningPayload.ChainFamily == k4k3ruSDKTradeHubExecution.ChainFamilyUnknown || strings.TrimSpace(r.SigningPayload.Digest) == "" {
		return invalidPrepareResult("signing_payload=invalid")
	}
	if r.SubmitParams == nil {
		return invalidPrepareResult("submit_params=null")
	}
	if r.SubmitParams.ExecutionID != r.ExecutionID || r.SubmitParams.PayloadDigest != r.SigningPayload.Digest {
		return invalidPrepareResult("submit_params=mismatch")
	}
	if r.PreparedAt <= 0 {
		return invalidPrepareResult("prepared_at=out_of_range")
	}
	if r.ExpiresAt <= r.PreparedAt {
		return invalidPrepareResult("expires_at=out_of_range")
	}
	if r.Status == PrepareStatusReady {
		if r.Approval != nil {
			return invalidPrepareResult("approval=invalid")
		}
		if !positiveUint256(r.AmountIn) || !positiveUint256(r.AmountOut) {
			return invalidPrepareResult("swap_amounts=invalid")
		}
		return nil
	}
	if r.Approval == nil {
		return invalidPrepareResult("approval=null")
	}
	if r.AmountIn != "" || r.AmountOut != "" {
		return invalidPrepareResult("swap_amounts=invalid")
	}
	if strings.TrimSpace(r.Approval.Token) == "" || strings.TrimSpace(r.Approval.Owner) == "" || strings.TrimSpace(r.Approval.Spender) == "" {
		return invalidPrepareResult("approval=invalid")
	}
	if !nonNegativeUint256(r.Approval.CurrentAllowance) || !positiveUint256(r.Approval.RequiredAllowance) || !positiveUint256(r.Approval.ApprovalAmount) {
		return invalidPrepareResult("approval_amounts=invalid")
	}
	required, _ := new(big.Int).SetString(r.Approval.RequiredAllowance, 10)
	approval, _ := new(big.Int).SetString(r.Approval.ApprovalAmount, 10)
	if approval.Cmp(required) < 0 {
		return invalidPrepareResult("approval_amount=out_of_range")
	}
	return nil
}

func positiveUint256(value string) bool {
	parsed, ok := new(big.Int).SetString(strings.TrimSpace(value), 10)
	return ok && parsed.Sign() > 0 && parsed.BitLen() <= 256
}

func nonNegativeUint256(value string) bool {
	parsed, ok := new(big.Int).SetString(strings.TrimSpace(value), 10)
	return ok && parsed.Sign() >= 0 && parsed.BitLen() <= 256
}

func invalidPrepareResult(state string) error {
	return k4k3ruSDKAppError.Tracef("failed to validate trade hub swap preparation result: %w: %s", k4k3ruSDKAppError.InvalidParameter(), state)
}
