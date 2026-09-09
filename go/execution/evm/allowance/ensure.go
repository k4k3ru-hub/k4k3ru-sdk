package allowance

import (
	"context"
	"fmt"
	"math/big"
	"strings"
)

type EnsureOperationDeps struct {
	CheckOperation     *CheckOperation
	TransactionBuilder ApprovalTransactionBuilder
	TransactionSigner  TransactionSigner
	TransactionSender  TransactionSender
	ReceiptWaiter      ReceiptWaiter
}

type EnsureOperation struct {
	deps EnsureOperationDeps
}

// NewEnsureOperation creates a state-changing allowance operation.
//
// Parameters:
//   - deps: explicit allowance workflow dependencies.
//
// Returns:
//   - Allowance ensure operation.
//   - Construction error.
//
// Version:
//   - 2026-09-09: Added.
func NewEnsureOperation(deps EnsureOperationDeps) (*EnsureOperation, error) {
	if deps.CheckOperation == nil {
		return nil, fmt.Errorf("failed to create allowance ensure operation: check_operation=null")
	}
	if deps.TransactionBuilder == nil {
		return nil, fmt.Errorf("failed to create allowance ensure operation: transaction_builder=null")
	}
	if deps.TransactionSigner == nil {
		return nil, fmt.Errorf("failed to create allowance ensure operation: transaction_signer=null")
	}
	if deps.TransactionSender == nil {
		return nil, fmt.Errorf("failed to create allowance ensure operation: transaction_sender=null")
	}
	if deps.ReceiptWaiter == nil {
		return nil, fmt.Errorf("failed to create allowance ensure operation: receipt_waiter=null")
	}
	return &EnsureOperation{deps: deps}, nil
}

// Execute ensures sufficient allowance and sends an approval only when required.
//
// Parameters:
//   - ctx: request context; nil uses context.Background.
//   - params: allowance requirement and approval policy.
//
// Returns:
//   - Final allowance state and optional approval transaction hash.
//   - Workflow error.
//
// Version:
//   - 2026-09-09: Added.
func (o *EnsureOperation) Execute(ctx context.Context, params EnsureParams) (*EnsureResult, error) {
	if o == nil {
		return nil, fmt.Errorf("failed to ensure token allowance: operation=null")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	params.CheckParams = normalizeCheckParams(params.CheckParams)
	if strings.ToLower(strings.TrimSpace(o.deps.TransactionSigner.Address())) != params.Owner {
		return nil, fmt.Errorf("failed to ensure token allowance: signer_address=invalid")
	}
	initial, err := o.deps.CheckOperation.Execute(ctx, params.CheckParams)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure token allowance: %w", err)
	}
	result := &EnsureResult{CheckResult: *initial}
	if !initial.BalanceSufficient {
		return result, fmt.Errorf("failed to ensure token allowance: insufficient token balance")
	}
	if !initial.AllowanceRequired {
		return result, nil
	}
	approvalAmount, err := resolveApprovalAmount(params)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure token allowance: %w", err)
	}
	result.ApprovalAmount = approvalAmount.String()
	unsigned, err := o.deps.TransactionBuilder.BuildApprovalTransaction(ctx, ApprovalTransactionParams{
		Chain: params.Chain, Network: params.Network, Token: params.Token, Owner: params.Owner, Spender: params.Spender, Amount: result.ApprovalAmount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to ensure token allowance: failed to build approval transaction: %w", err)
	}
	if unsigned == nil {
		return nil, fmt.Errorf("failed to ensure token allowance: unsigned_transaction=null")
	}
	signed, err := o.deps.TransactionSigner.SignTransaction(ctx, unsigned)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure token allowance: failed to sign approval transaction: %w", err)
	}
	if signed == nil {
		return nil, fmt.Errorf("failed to ensure token allowance: signed_transaction=null")
	}
	hash, err := o.deps.TransactionSender.SendTransaction(ctx, signed)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure token allowance: failed to send approval transaction: %w", err)
	}
	if strings.TrimSpace(hash) == "" {
		return nil, fmt.Errorf("failed to ensure token allowance: transaction_hash=empty")
	}
	result.ApprovalSent = true
	result.TransactionHash = hash
	receipt, err := o.deps.ReceiptWaiter.WaitReceipt(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure token allowance: failed to wait for approval receipt: %w", err)
	}
	if receipt == nil || !receipt.Success {
		return nil, fmt.Errorf("failed to ensure token allowance: approval_receipt=invalid")
	}
	final, err := o.deps.CheckOperation.Execute(ctx, params.CheckParams)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure token allowance: failed to verify approval: %w", err)
	}
	if final.AllowanceRequired {
		return nil, fmt.Errorf("failed to ensure token allowance: approved allowance is insufficient")
	}
	result.CheckResult = *final
	return result, nil
}

func resolveApprovalAmount(params EnsureParams) (*big.Int, error) {
	required, ok := new(big.Int).SetString(params.RequiredAmount, 10)
	if !ok || required.Sign() <= 0 || required.BitLen() > 256 {
		return nil, fmt.Errorf("failed to resolve approval amount: required_amount=invalid")
	}
	switch params.Policy {
	case ApprovalPolicyExact:
		return required, nil
	case ApprovalPolicyFixed:
		fixed, ok := new(big.Int).SetString(strings.TrimSpace(params.FixedAmount), 10)
		if !ok || fixed.Sign() <= 0 || fixed.BitLen() > 256 || fixed.Cmp(required) < 0 {
			return nil, fmt.Errorf("failed to resolve approval amount: fixed_amount=invalid")
		}
		return fixed, nil
	case ApprovalPolicyMax:
		return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)), nil
	default:
		return nil, fmt.Errorf("failed to resolve approval amount: approval_policy=invalid")
	}
}
