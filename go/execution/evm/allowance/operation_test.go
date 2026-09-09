package allowance

import (
	"context"
	"math/big"
	"testing"

	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

const (
	testToken   = "0x0000000000000000000000000000000000000001"
	testOwner   = "0x0000000000000000000000000000000000000002"
	testSpender = "0x0000000000000000000000000000000000000003"
)

type readerStub struct {
	balance    *big.Int
	allowances []*big.Int
	reads      int
}

func (s *readerStub) Balance(context.Context, k4k3ruOnchainCore.Chain, k4k3ruOnchainCore.Network, string, string) (*big.Int, error) {
	return new(big.Int).Set(s.balance), nil
}

func (s *readerStub) Allowance(context.Context, k4k3ruOnchainCore.Chain, k4k3ruOnchainCore.Network, string, string, string) (*big.Int, error) {
	index := s.reads
	if index >= len(s.allowances) {
		index = len(s.allowances) - 1
	}
	s.reads++
	return new(big.Int).Set(s.allowances[index]), nil
}

type registryStub struct{ allowed bool }

func (s registryStub) Allows(k4k3ruOnchainCore.Chain, k4k3ruOnchainCore.Network, string, string) bool {
	return s.allowed
}

type builderStub struct{ params ApprovalTransactionParams }

func (s *builderStub) BuildApprovalTransaction(_ context.Context, params ApprovalTransactionParams) (*UnsignedTransaction, error) {
	s.params = params
	return &UnsignedTransaction{To: params.Token}, nil
}

type signerStub struct{ address string }

func (s signerStub) Address() string { return s.address }
func (s signerStub) SignTransaction(context.Context, *UnsignedTransaction) (*SignedTransaction, error) {
	return &SignedTransaction{RawTransaction: []byte{1}}, nil
}

type senderStub struct{ calls int }

func (s *senderStub) SendTransaction(context.Context, *SignedTransaction) (string, error) {
	s.calls++
	return "0xtransaction", nil
}

type waiterStub struct{ hash string }

func (s *waiterStub) WaitReceipt(_ context.Context, hash string) (*Receipt, error) {
	s.hash = hash
	return &Receipt{TransactionHash: hash, Success: true}, nil
}

func TestCheckOperationReportsRequirement(t *testing.T) {
	reader := &readerStub{balance: big.NewInt(20_000), allowances: []*big.Int{big.NewInt(9_999)}}
	operation, err := NewCheckOperation(reader, registryStub{allowed: true})
	if err != nil {
		t.Fatalf("NewCheckOperation() error = %v, want nil", err)
	}
	result, err := operation.Execute(t.Context(), validCheckParams())
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if !result.BalanceSufficient || !result.AllowanceRequired || result.CurrentAllowance != "9999" {
		t.Fatalf("Execute() result = %+v", result)
	}
}

func TestEnsureOperationApprovesAndVerifiesAllowance(t *testing.T) {
	reader := &readerStub{balance: big.NewInt(20_000), allowances: []*big.Int{big.NewInt(0), big.NewInt(20_000)}}
	check, err := NewCheckOperation(reader, registryStub{allowed: true})
	if err != nil {
		t.Fatalf("NewCheckOperation() error = %v, want nil", err)
	}
	builder := new(builderStub)
	sender := new(senderStub)
	waiter := new(waiterStub)
	operation, err := NewEnsureOperation(EnsureOperationDeps{
		CheckOperation: check, TransactionBuilder: builder, TransactionSigner: signerStub{address: testOwner}, TransactionSender: sender, ReceiptWaiter: waiter,
	})
	if err != nil {
		t.Fatalf("NewEnsureOperation() error = %v, want nil", err)
	}
	result, err := operation.Execute(t.Context(), EnsureParams{CheckParams: validCheckParams(), Policy: ApprovalPolicyFixed, FixedAmount: "20000"})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if !result.ApprovalSent || result.ApprovalAmount != "20000" || result.TransactionHash != "0xtransaction" || result.AllowanceRequired {
		t.Fatalf("Execute() result = %+v", result)
	}
	if builder.params.Amount != "20000" || sender.calls != 1 || waiter.hash != "0xtransaction" {
		t.Fatalf("workflow = params %+v sender_calls %d receipt_hash %q", builder.params, sender.calls, waiter.hash)
	}
}

func TestEnsureOperationSkipsSufficientAllowance(t *testing.T) {
	reader := &readerStub{balance: big.NewInt(20_000), allowances: []*big.Int{big.NewInt(10_000)}}
	check, err := NewCheckOperation(reader, registryStub{allowed: true})
	if err != nil {
		t.Fatalf("NewCheckOperation() error = %v, want nil", err)
	}
	sender := new(senderStub)
	operation, err := NewEnsureOperation(EnsureOperationDeps{
		CheckOperation: check, TransactionBuilder: new(builderStub), TransactionSigner: signerStub{address: testOwner}, TransactionSender: sender, ReceiptWaiter: new(waiterStub),
	})
	if err != nil {
		t.Fatalf("NewEnsureOperation() error = %v, want nil", err)
	}
	result, err := operation.Execute(t.Context(), EnsureParams{CheckParams: validCheckParams(), Policy: ApprovalPolicyExact})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if result.ApprovalSent || sender.calls != 0 {
		t.Fatalf("Execute() result = %+v sender_calls=%d", result, sender.calls)
	}
}

func TestCheckOperationRejectsUntrustedSpender(t *testing.T) {
	operation, err := NewCheckOperation(&readerStub{balance: big.NewInt(1), allowances: []*big.Int{big.NewInt(1)}}, registryStub{})
	if err != nil {
		t.Fatalf("NewCheckOperation() error = %v, want nil", err)
	}
	if _, err := operation.Execute(t.Context(), validCheckParams()); err == nil {
		t.Fatal("Execute() error = nil")
	}
}

func validCheckParams() CheckParams {
	return CheckParams{Chain: k4k3ruOnchainCore.ChainBase, Network: k4k3ruOnchainCore.NetworkSepolia, Token: testToken, Owner: testOwner, Spender: testSpender, RequiredAmount: "10000"}
}
