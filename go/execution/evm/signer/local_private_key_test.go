package signer

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	k4k3ruSDKAllowance "github.com/k4k3ru-hub/k4k3ru-sdk/go/execution/evm/allowance"
	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
	k4k3ruOnchainEVM "github.com/k4k3ru-hub/onchain/go/evm"
)

func TestLocalPrivateKeySignerSignsDynamicFeeTransaction(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	signer, err := NewLocalPrivateKeySigner(crypto.FromECDSA(privateKey))
	if err != nil {
		t.Fatalf("NewLocalPrivateKeySigner() error = %v, want nil", err)
	}
	t.Cleanup(func() { _ = signer.Close() })
	wantAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	if signer.Address() != wantAddress.Hex() {
		t.Fatalf("Address() = %q, want %q", signer.Address(), wantAddress.Hex())
	}

	result, err := signer.SignTransaction(t.Context(), &k4k3ruSDKAllowance.UnsignedTransaction{
		Chain: k4k3ruOnchainCore.ChainBase, Network: k4k3ruOnchainCore.NetworkSepolia,
		From: wantAddress.Hex(), Payload: newUnsignedTransaction(t, k4k3ruOnchainEVM.ChainIDBaseSepolia),
	})
	if err != nil {
		t.Fatalf("SignTransaction() error = %v, want nil", err)
	}
	if len(result.RawTransaction) == 0 {
		t.Fatal("SignTransaction() raw transaction is empty")
	}
	signed, ok := result.Payload.(*types.Transaction)
	if !ok {
		t.Fatalf("SignTransaction() payload type = %T", result.Payload)
	}
	recovered, err := types.Sender(types.LatestSignerForChainID(signed.ChainId()), signed)
	if err != nil {
		t.Fatalf("Sender() error = %v", err)
	}
	if recovered != wantAddress {
		t.Fatalf("Sender() = %s, want %s", recovered.Hex(), wantAddress.Hex())
	}
}

func TestLocalPrivateKeySignerRejectsInvalidInput(t *testing.T) {
	if _, err := NewLocalPrivateKeySigner(nil); err == nil {
		t.Fatal("NewLocalPrivateKeySigner(nil) error = nil")
	}
	if _, err := NewLocalPrivateKeySigner(make([]byte, 31)); err == nil {
		t.Fatal("NewLocalPrivateKeySigner(short) error = nil")
	}
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	signer, err := NewLocalPrivateKeySigner(crypto.FromECDSA(privateKey))
	if err != nil {
		t.Fatalf("NewLocalPrivateKeySigner() error = %v", err)
	}
	t.Cleanup(func() { _ = signer.Close() })
	tests := []struct {
		name     string
		ctx      context.Context
		unsigned *k4k3ruSDKAllowance.UnsignedTransaction
	}{
		{name: "nil context", unsigned: &k4k3ruSDKAllowance.UnsignedTransaction{}},
		{name: "nil transaction", ctx: t.Context()},
		{name: "mismatched from", ctx: t.Context(), unsigned: &k4k3ruSDKAllowance.UnsignedTransaction{From: common.Address{1}.Hex()}},
		{name: "invalid payload", ctx: t.Context(), unsigned: &k4k3ruSDKAllowance.UnsignedTransaction{Chain: k4k3ruOnchainCore.ChainBase, Network: k4k3ruOnchainCore.NetworkSepolia, From: signer.Address(), Payload: struct{}{}}},
		{name: "mismatched chain", ctx: t.Context(), unsigned: &k4k3ruSDKAllowance.UnsignedTransaction{Chain: k4k3ruOnchainCore.ChainBase, Network: k4k3ruOnchainCore.NetworkMainnet, From: signer.Address(), Payload: newUnsignedTransaction(t, k4k3ruOnchainEVM.ChainIDBaseSepolia)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := signer.SignTransaction(test.ctx, test.unsigned); err == nil {
				t.Fatal("SignTransaction() error = nil")
			}
		})
	}
}

func TestLocalPrivateKeySignerClose(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	signer, err := NewLocalPrivateKeySigner(crypto.FromECDSA(privateKey))
	if err != nil {
		t.Fatalf("NewLocalPrivateKeySigner() error = %v", err)
	}
	if err := signer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if signer.Address() != "" {
		t.Fatalf("Address() after Close = %q, want empty", signer.Address())
	}
	if _, err := signer.SignTransaction(t.Context(), &k4k3ruSDKAllowance.UnsignedTransaction{}); err == nil {
		t.Fatal("SignTransaction() after Close error = nil")
	}
}

func newUnsignedTransaction(t *testing.T, chainID k4k3ruOnchainEVM.ChainID) *k4k3ruOnchainEVM.UnsignedDynamicFeeTransaction {
	t.Helper()
	transaction, err := k4k3ruOnchainEVM.NewUnsignedDynamicFeeTransaction(k4k3ruOnchainEVM.DynamicFeeTransactionParams{
		ChainID: chainID, Nonce: 7, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2), GasLimit: 21_000,
		To: common.Address{2}, Value: big.NewInt(3), Data: []byte{4},
	})
	if err != nil {
		t.Fatalf("NewUnsignedDynamicFeeTransaction() error = %v", err)
	}
	return transaction
}
