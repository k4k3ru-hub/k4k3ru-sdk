package allowance

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
	k4k3ruOnchainEVM "github.com/k4k3ru-hub/onchain/go/evm"
)

type evmTokenReaderStub struct{}

func (evmTokenReaderStub) GetTokenBalance(context.Context, common.Address, common.Address, *big.Int) (*big.Int, error) {
	return big.NewInt(20_000), nil
}

func (evmTokenReaderStub) GetTokenAllowance(context.Context, common.Address, common.Address, common.Address, *big.Int) (*big.Int, error) {
	return big.NewInt(10_000), nil
}

type evmExecutionClientStub struct {
	call ethereum.CallMsg
}

func (s *evmExecutionClientStub) ChainID(context.Context) (k4k3ruOnchainEVM.ChainID, error) {
	return k4k3ruOnchainEVM.ChainIDBaseSepolia, nil
}

func (s *evmExecutionClientStub) PendingNonce(context.Context, common.Address) (uint64, error) {
	return 7, nil
}

func (s *evmExecutionClientStub) LatestFeeData(context.Context) (k4k3ruOnchainEVM.FeeData, error) {
	return k4k3ruOnchainEVM.FeeData{BaseFeePerGas: big.NewInt(10), SuggestedPriorityFeePerGas: big.NewInt(2)}, nil
}

func (s *evmExecutionClientStub) EstimateGas(_ context.Context, call ethereum.CallMsg) (uint64, error) {
	s.call = call
	return 50_000, nil
}

func TestEVMReaderAdapter(t *testing.T) {
	adapter, err := NewEVMReaderAdapter(k4k3ruOnchainCore.ChainBase, k4k3ruOnchainCore.NetworkSepolia, evmTokenReaderStub{})
	if err != nil {
		t.Fatalf("NewEVMReaderAdapter() error = %v, want nil", err)
	}
	balance, err := adapter.Balance(t.Context(), k4k3ruOnchainCore.ChainBase, k4k3ruOnchainCore.NetworkSepolia, testToken, testOwner)
	if err != nil || balance.Cmp(big.NewInt(20_000)) != 0 {
		t.Fatalf("Balance() = %v, %v", balance, err)
	}
	value, err := adapter.Allowance(t.Context(), k4k3ruOnchainCore.ChainBase, k4k3ruOnchainCore.NetworkSepolia, testToken, testOwner, testSpender)
	if err != nil || value.Cmp(big.NewInt(10_000)) != 0 {
		t.Fatalf("Allowance() = %v, %v", value, err)
	}
}

func TestEVMApprovalTransactionBuilder(t *testing.T) {
	client := new(evmExecutionClientStub)
	builder, err := NewEVMApprovalTransactionBuilder(k4k3ruOnchainCore.ChainBase, k4k3ruOnchainCore.NetworkSepolia, client)
	if err != nil {
		t.Fatalf("NewEVMApprovalTransactionBuilder() error = %v, want nil", err)
	}
	result, err := builder.BuildApprovalTransaction(t.Context(), ApprovalTransactionParams{
		Chain: k4k3ruOnchainCore.ChainBase, Network: k4k3ruOnchainCore.NetworkSepolia, Token: testToken, Owner: testOwner, Spender: testSpender, Amount: "10000",
	})
	if err != nil {
		t.Fatalf("BuildApprovalTransaction() error = %v, want nil", err)
	}
	transaction, ok := result.Payload.(*k4k3ruOnchainEVM.UnsignedDynamicFeeTransaction)
	if !ok || transaction == nil {
		t.Fatalf("BuildApprovalTransaction() payload = %T", result.Payload)
	}
	value, err := transaction.Transaction()
	if err != nil {
		t.Fatalf("Transaction() error = %v, want nil", err)
	}
	if value.Nonce() != 7 || value.Gas() != 50_000 || value.GasTipCap().Cmp(big.NewInt(2)) != 0 || value.GasFeeCap().Cmp(big.NewInt(22)) != 0 {
		t.Fatalf("Transaction() = %+v", value)
	}
	if client.call.From != common.HexToAddress(testOwner) || client.call.To == nil || *client.call.To != common.HexToAddress(testToken) {
		t.Fatalf("EstimateGas() call = %+v", client.call)
	}
}
