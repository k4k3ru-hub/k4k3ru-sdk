package signer

import (
	"context"
	"crypto/ecdsa"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	k4k3ruSDKAllowance "github.com/k4k3ru-hub/k4k3ru-sdk/go/execution/evm/allowance"
	k4k3ruOnchainEVM "github.com/k4k3ru-hub/onchain/go/evm"
)

type LocalPrivateKeySigner struct {
	mu         sync.RWMutex
	privateKey *ecdsa.PrivateKey
	address    common.Address
}

var _ k4k3ruSDKAllowance.TransactionSigner = (*LocalPrivateKeySigner)(nil)

// NewLocalPrivateKeySigner creates an EVM signer backed by an in-process private key.
//
// Parameters:
//   - privateKeyBytes: 32-byte secp256k1 private key copied by the signer
//
// Returns:
//   - Local EVM transaction signer.
//   - Construction error.
//
// Version:
//   - 2026-09-09: Added.
func NewLocalPrivateKeySigner(privateKeyBytes []byte) (*LocalPrivateKeySigner, error) {
	if len(privateKeyBytes) == 0 {
		return nil, k4k3ruSDKAppError.Tracef("failed to create local private key signer: %w: private_key=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if len(privateKeyBytes) != 32 {
		return nil, k4k3ruSDKAppError.Tracef("failed to create local private key signer: %w: private_key=invalid actual_length=%d expected_length=%d", k4k3ruSDKAppError.InvalidParameter(), len(privateKeyBytes), 32)
	}

	keyCopy := append([]byte(nil), privateKeyBytes...)
	defer clear(keyCopy)
	privateKey, err := crypto.ToECDSA(keyCopy)
	if err != nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to create local private key signer: invalid secp256k1 private key: %w: private_key=invalid", err)
	}

	return &LocalPrivateKeySigner{privateKey: privateKey, address: crypto.PubkeyToAddress(privateKey.PublicKey)}, nil
}

// Address returns the EVM address controlled by the local private key.
//
// Returns:
//   - Checksummed EVM address, or an empty string after Close.
//
// Version:
//   - 2026-09-09: Added.
func (s *LocalPrivateKeySigner) Address() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.privateKey == nil {
		return ""
	}
	return s.address.Hex()
}

// SignTransaction signs an SDK unsigned transaction as an EIP-1559 transaction.
//
// Parameters:
//   - ctx: Signing context.
//   - unsigned: Transaction whose payload is an onchain EVM dynamic-fee transaction.
//
// Returns:
//   - Signed raw transaction and underlying signed transaction.
//   - Validation or signing error.
//
// Version:
//   - 2026-09-09: Added.
func (s *LocalPrivateKeySigner) SignTransaction(ctx context.Context, unsigned *k4k3ruSDKAllowance.UnsignedTransaction) (*k4k3ruSDKAllowance.SignedTransaction, error) {
	if s == nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: %w: signer=null", k4k3ruSDKAppError.InvalidParameter())
	}
	if ctx == nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: %w: context=null", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := ctx.Err(); err != nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: %w", err)
	}
	if unsigned == nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: %w: unsigned_transaction=null", k4k3ruSDKAppError.InvalidParameter())
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.privateKey == nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: signer is closed: %w", k4k3ruSDKAppError.InvalidParameter())
	}
	if !common.IsHexAddress(unsigned.From) || common.HexToAddress(unsigned.From) != s.address {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: %w: from=mismatch", k4k3ruSDKAppError.InvalidParameter())
	}

	expectedChainID, err := k4k3ruOnchainEVM.ResolveChainID(unsigned.Chain, unsigned.Network)
	if err != nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: %w", err)
	}
	payload, ok := unsigned.Payload.(*k4k3ruOnchainEVM.UnsignedDynamicFeeTransaction)
	if !ok || payload == nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: %w: payload=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	transaction, err := payload.Transaction()
	if err != nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: %w", err)
	}
	if transaction.ChainId() == nil || transaction.ChainId().Uint64() != uint64(expectedChainID) {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: %w: chain_id=mismatch", k4k3ruSDKAppError.InvalidParameter())
	}

	chainSigner := types.LatestSignerForChainID(transaction.ChainId())
	signedTransaction, err := types.SignTx(transaction, chainSigner, s.privateKey)
	if err != nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: %w", err)
	}
	recoveredAddress, err := types.Sender(chainSigner, signedTransaction)
	if err != nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: failed to recover signer: %w", err)
	}
	if recoveredAddress != s.address {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: %w: signer_address=mismatch", k4k3ruSDKAppError.InvalidParameter())
	}
	rawTransaction, err := signedTransaction.MarshalBinary()
	if err != nil {
		return nil, k4k3ruSDKAppError.Tracef("failed to sign local evm transaction: failed to serialize signed transaction: %w", err)
	}

	return &k4k3ruSDKAllowance.SignedTransaction{RawTransaction: append([]byte(nil), rawTransaction...), Payload: signedTransaction}, nil
}

// Close clears the in-process private key and prevents further signing.
//
// Returns:
//   - Nil. Close is idempotent.
//
// Version:
//   - 2026-09-09: Added.
func (s *LocalPrivateKeySigner) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.privateKey != nil {
		s.privateKey.D.SetInt64(0)
		s.privateKey = nil
	}
	s.address = common.Address{}
	return nil
}
