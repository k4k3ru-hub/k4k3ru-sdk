package swap

import (
	"errors"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	k4k3ruSDKTradeHubExecution "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
)

func TestPrepareResultValidateReady(t *testing.T) {
	t.Parallel()
	result := validPrepareResult(PrepareStatusReady)
	result.AmountIn, result.AmountOut = "1000000", "999000000000000"
	if err := result.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestPrepareResultValidateApprovalRequired(t *testing.T) {
	t.Parallel()
	result := validPrepareResult(PrepareStatusApprovalRequired)
	result.Approval = &ApprovalRequirement{
		Token: "0xtoken", Owner: "0xowner", Spender: "0xspender",
		CurrentAllowance: "0", RequiredAllowance: "1000000", ApprovalAmount: "100000000",
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestPrepareResultValidateRejectsInconsistentResults(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		result PrepareResult
	}{
		{name: "unknown status", result: validPrepareResult(PrepareStatusUnknown)},
		{name: "ready without amounts", result: validPrepareResult(PrepareStatusReady)},
		{name: "approval without requirement", result: validPrepareResult(PrepareStatusApprovalRequired)},
		{name: "mismatched submit", result: func() PrepareResult {
			value := validPrepareResult(PrepareStatusReady)
			value.AmountIn, value.AmountOut = "1", "1"
			value.SubmitParams.PayloadDigest = "0xother"
			return value
		}()},
		{name: "approval below required", result: func() PrepareResult {
			value := validPrepareResult(PrepareStatusApprovalRequired)
			value.Approval = &ApprovalRequirement{Token: "t", Owner: "o", Spender: "s", CurrentAllowance: "0", RequiredAllowance: "2", ApprovalAmount: "1"}
			return value
		}()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.result.Validate(); !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func validPrepareResult(status PrepareStatus) PrepareResult {
	return PrepareResult{
		ExecutionID: "execution-1", Status: status, Chain: "base", Network: "sepolia",
		SigningPayload: &k4k3ruSDKTradeHubExecution.SigningPayload{ChainFamily: k4k3ruSDKTradeHubExecution.ChainFamilyEVM, Digest: "0xdigest", UnsignedTransaction: &k4k3ruSDKTradeHubExecution.EVMUnsignedTransaction{ChainID: "84532"}},
		SubmitParams:   &k4k3ruSDKTradeHubExecution.SubmitParams{ExecutionID: "execution-1", PayloadDigest: "0xdigest"},
		PreparedAt:     1_000, ExpiresAt: 2_000,
	}
}
