package allowance

import (
	"context"
	"math/big"

	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

type ApprovalPolicy string

const (
	ApprovalPolicyExact ApprovalPolicy = "exact"
	ApprovalPolicyFixed ApprovalPolicy = "fixed"
	ApprovalPolicyMax   ApprovalPolicy = "max"
)

type CheckParams struct {
	Chain          k4k3ruOnchainCore.Chain
	Network        k4k3ruOnchainCore.Network
	Token          string
	Owner          string
	Spender        string
	RequiredAmount string
}

type CheckResult struct {
	Balance           string
	CurrentAllowance  string
	RequiredAmount    string
	BalanceSufficient bool
	AllowanceRequired bool
}

type EnsureParams struct {
	CheckParams
	Policy      ApprovalPolicy
	FixedAmount string
}

type EnsureResult struct {
	CheckResult
	ApprovalAmount  string
	ApprovalSent    bool
	TransactionHash string
}

type ApprovalTransactionParams struct {
	Chain   k4k3ruOnchainCore.Chain
	Network k4k3ruOnchainCore.Network
	Token   string
	Owner   string
	Spender string
	Amount  string
}

type UnsignedTransaction struct {
	Chain   k4k3ruOnchainCore.Chain
	Network k4k3ruOnchainCore.Network
	From    string
	To      string
	Data    []byte
	Value   string
	Payload any
}

type SignedTransaction struct {
	RawTransaction []byte
	Payload        any
}

type Receipt struct {
	TransactionHash string
	Success         bool
}

type Reader interface {
	Balance(context.Context, k4k3ruOnchainCore.Chain, k4k3ruOnchainCore.Network, string, string) (*big.Int, error)
	Allowance(context.Context, k4k3ruOnchainCore.Chain, k4k3ruOnchainCore.Network, string, string, string) (*big.Int, error)
}

type SpenderRegistry interface {
	Allows(k4k3ruOnchainCore.Chain, k4k3ruOnchainCore.Network, string, string) bool
}

type ApprovalTransactionBuilder interface {
	BuildApprovalTransaction(context.Context, ApprovalTransactionParams) (*UnsignedTransaction, error)
}

type TransactionSigner interface {
	Address() string
	SignTransaction(context.Context, *UnsignedTransaction) (*SignedTransaction, error)
}

type TransactionSender interface {
	SendTransaction(context.Context, *SignedTransaction) (string, error)
}

type ReceiptWaiter interface {
	WaitReceipt(context.Context, string) (*Receipt, error)
}
