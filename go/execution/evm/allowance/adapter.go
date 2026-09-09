package allowance

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
	k4k3ruOnchainEVM "github.com/k4k3ru-hub/onchain/go/evm"
	k4k3ruOnchainERC20 "github.com/k4k3ru-hub/onchain/go/evm/erc20"
)

type EVMTokenReader interface {
	GetTokenBalance(context.Context, common.Address, common.Address, *big.Int) (*big.Int, error)
	GetTokenAllowance(context.Context, common.Address, common.Address, common.Address, *big.Int) (*big.Int, error)
}

type EVMExecutionClient interface {
	ChainID(context.Context) (k4k3ruOnchainEVM.ChainID, error)
	PendingNonce(context.Context, common.Address) (uint64, error)
	LatestFeeData(context.Context) (k4k3ruOnchainEVM.FeeData, error)
	EstimateGas(context.Context, ethereum.CallMsg) (uint64, error)
}

type EVMReaderAdapter struct {
	chain   k4k3ruOnchainCore.Chain
	network k4k3ruOnchainCore.Network
	reader  EVMTokenReader
}

// NewEVMReaderAdapter creates a chain-bound ERC20 reader adapter.
//
// Parameters:
//   - chain: EVM chain.
//   - network: EVM network.
//   - reader: onchain ERC20 reader.
//
// Returns:
//   - SDK allowance reader adapter.
//   - Construction error.
//
// Version:
//   - 2026-09-09: Added.
func NewEVMReaderAdapter(chain k4k3ruOnchainCore.Chain, network k4k3ruOnchainCore.Network, reader EVMTokenReader) (*EVMReaderAdapter, error) {
	if _, err := k4k3ruOnchainEVM.ResolveChainID(chain, network); err != nil {
		return nil, fmt.Errorf("failed to create evm allowance reader adapter: %w", err)
	}
	if reader == nil {
		return nil, fmt.Errorf("failed to create evm allowance reader adapter: reader=null")
	}
	return &EVMReaderAdapter{chain: chain, network: network, reader: reader}, nil
}

// Balance reads an ERC20 balance for the configured chain and network.
//
// Version:
//   - 2026-09-09: Added.
func (a *EVMReaderAdapter) Balance(ctx context.Context, chain k4k3ruOnchainCore.Chain, network k4k3ruOnchainCore.Network, token, owner string) (*big.Int, error) {
	if a == nil || a.reader == nil {
		return nil, fmt.Errorf("failed to read evm token balance: adapter=invalid")
	}
	if chain != a.chain || network != a.network || !isEVMAddress(token) || !isEVMAddress(owner) {
		return nil, fmt.Errorf("failed to read evm token balance: parameters=invalid")
	}
	value, err := a.reader.GetTokenBalance(ctx, common.HexToAddress(token), common.HexToAddress(owner), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read evm token balance: %w", err)
	}
	return value, nil
}

// Allowance reads an ERC20 allowance for the configured chain and network.
//
// Version:
//   - 2026-09-09: Added.
func (a *EVMReaderAdapter) Allowance(ctx context.Context, chain k4k3ruOnchainCore.Chain, network k4k3ruOnchainCore.Network, token, owner, spender string) (*big.Int, error) {
	if a == nil || a.reader == nil {
		return nil, fmt.Errorf("failed to read evm token allowance: adapter=invalid")
	}
	if chain != a.chain || network != a.network || !isEVMAddress(token) || !isEVMAddress(owner) || !isEVMAddress(spender) {
		return nil, fmt.Errorf("failed to read evm token allowance: parameters=invalid")
	}
	value, err := a.reader.GetTokenAllowance(ctx, common.HexToAddress(token), common.HexToAddress(owner), common.HexToAddress(spender), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read evm token allowance: %w", err)
	}
	return value, nil
}

type EVMApprovalTransactionBuilder struct {
	chain   k4k3ruOnchainCore.Chain
	network k4k3ruOnchainCore.Network
	client  EVMExecutionClient
}

// NewEVMApprovalTransactionBuilder creates a chain-bound approval transaction builder.
//
// Parameters:
//   - chain: EVM chain.
//   - network: EVM network.
//   - client: EVM execution RPC client.
//
// Returns:
//   - Approval transaction builder.
//   - Construction error.
//
// Version:
//   - 2026-09-09: Added.
func NewEVMApprovalTransactionBuilder(chain k4k3ruOnchainCore.Chain, network k4k3ruOnchainCore.Network, client EVMExecutionClient) (*EVMApprovalTransactionBuilder, error) {
	if _, err := k4k3ruOnchainEVM.ResolveChainID(chain, network); err != nil {
		return nil, fmt.Errorf("failed to create evm approval transaction builder: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("failed to create evm approval transaction builder: client=null")
	}
	return &EVMApprovalTransactionBuilder{chain: chain, network: network, client: client}, nil
}

// BuildApprovalTransaction builds an unsigned EIP-1559 ERC20 approval transaction.
//
// Version:
//   - 2026-09-09: Added.
func (b *EVMApprovalTransactionBuilder) BuildApprovalTransaction(ctx context.Context, params ApprovalTransactionParams) (*UnsignedTransaction, error) {
	if b == nil || b.client == nil {
		return nil, fmt.Errorf("failed to build evm approval transaction: builder=invalid")
	}
	if params.Chain != b.chain || params.Network != b.network || !isEVMAddress(params.Token) || !isEVMAddress(params.Owner) || !isEVMAddress(params.Spender) {
		return nil, fmt.Errorf("failed to build evm approval transaction: parameters=invalid")
	}
	amount, ok := new(big.Int).SetString(params.Amount, 10)
	if !ok || amount.Sign() < 0 || amount.BitLen() > 256 {
		return nil, fmt.Errorf("failed to build evm approval transaction: amount=invalid")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	expectedChainID, err := k4k3ruOnchainEVM.ResolveChainID(params.Chain, params.Network)
	if err != nil {
		return nil, fmt.Errorf("failed to build evm approval transaction: %w", err)
	}
	actualChainID, err := b.client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build evm approval transaction: %w", err)
	}
	if actualChainID != expectedChainID {
		return nil, fmt.Errorf("failed to build evm approval transaction: chain_id=invalid")
	}
	owner := common.HexToAddress(params.Owner)
	token := common.HexToAddress(params.Token)
	data, err := k4k3ruOnchainERC20.EncodeApprove(common.HexToAddress(params.Spender), amount)
	if err != nil {
		return nil, fmt.Errorf("failed to build evm approval transaction: %w", err)
	}
	call := ethereum.CallMsg{From: owner, To: &token, Data: data, Value: new(big.Int)}
	gasLimit, err := b.client.EstimateGas(ctx, call)
	if err != nil {
		return nil, fmt.Errorf("failed to build evm approval transaction: %w", err)
	}
	nonce, err := b.client.PendingNonce(ctx, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to build evm approval transaction: %w", err)
	}
	fees, err := b.client.LatestFeeData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build evm approval transaction: %w", err)
	}
	if fees.BaseFeePerGas == nil || fees.SuggestedPriorityFeePerGas == nil {
		return nil, fmt.Errorf("failed to build evm approval transaction: fee_data=invalid")
	}
	feeCap := new(big.Int).Add(new(big.Int).Mul(fees.BaseFeePerGas, big.NewInt(2)), fees.SuggestedPriorityFeePerGas)
	transaction, err := k4k3ruOnchainEVM.NewUnsignedDynamicFeeTransaction(k4k3ruOnchainEVM.DynamicFeeTransactionParams{
		ChainID: expectedChainID, Nonce: nonce, GasTipCap: fees.SuggestedPriorityFeePerGas, GasFeeCap: feeCap, GasLimit: gasLimit, To: token, Value: new(big.Int), Data: data,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build evm approval transaction: %w", err)
	}
	return &UnsignedTransaction{Chain: params.Chain, Network: params.Network, From: params.Owner, To: params.Token, Data: append([]byte(nil), data...), Value: "0", Payload: transaction}, nil
}
