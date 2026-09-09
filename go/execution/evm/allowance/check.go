package allowance

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

type CheckOperation struct {
	reader          Reader
	spenderRegistry SpenderRegistry
}

// NewCheckOperation creates an allowance check operation.
//
// Parameters:
//   - reader: ERC20 balance and allowance reader.
//   - spenderRegistry: trusted spender registry.
//
// Returns:
//   - Allowance check operation.
//   - Construction error.
//
// Version:
//   - 2026-09-09: Added.
func NewCheckOperation(reader Reader, spenderRegistry SpenderRegistry) (*CheckOperation, error) {
	if reader == nil {
		return nil, fmt.Errorf("failed to create allowance check operation: reader=null")
	}
	if spenderRegistry == nil {
		return nil, fmt.Errorf("failed to create allowance check operation: spender_registry=null")
	}
	return &CheckOperation{reader: reader, spenderRegistry: spenderRegistry}, nil
}

// Execute checks an owner's token balance and allowance without changing state.
//
// Parameters:
//   - ctx: request context; nil uses context.Background.
//   - params: allowance check parameters.
//
// Returns:
//   - Current balance and allowance state.
//   - Check error.
//
// Version:
//   - 2026-09-09: Added.
func (o *CheckOperation) Execute(ctx context.Context, params CheckParams) (*CheckResult, error) {
	if o == nil || o.reader == nil || o.spenderRegistry == nil {
		return nil, fmt.Errorf("failed to check token allowance: operation=invalid")
	}
	params = normalizeCheckParams(params)
	required, err := validateCheckParams(params)
	if err != nil {
		return nil, fmt.Errorf("failed to check token allowance: %w", err)
	}
	if !o.spenderRegistry.Allows(params.Chain, params.Network, params.Token, params.Spender) {
		return nil, fmt.Errorf("failed to check token allowance: spender is not allowlisted: spender=%q", params.Spender)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	balance, err := o.reader.Balance(ctx, params.Chain, params.Network, params.Token, params.Owner)
	if err != nil {
		return nil, fmt.Errorf("failed to check token allowance: failed to get token balance: %w", err)
	}
	if balance == nil || balance.Sign() < 0 {
		return nil, fmt.Errorf("failed to check token allowance: balance=invalid")
	}
	current, err := o.reader.Allowance(ctx, params.Chain, params.Network, params.Token, params.Owner, params.Spender)
	if err != nil {
		return nil, fmt.Errorf("failed to check token allowance: failed to get current allowance: %w", err)
	}
	if current == nil || current.Sign() < 0 {
		return nil, fmt.Errorf("failed to check token allowance: current_allowance=invalid")
	}
	return &CheckResult{
		Balance: balance.String(), CurrentAllowance: current.String(), RequiredAmount: required.String(),
		BalanceSufficient: balance.Cmp(required) >= 0, AllowanceRequired: current.Cmp(required) < 0,
	}, nil
}

func normalizeCheckParams(params CheckParams) CheckParams {
	params.Chain = k4k3ruOnchainCore.Chain(strings.ToLower(strings.TrimSpace(string(params.Chain))))
	params.Network = k4k3ruOnchainCore.Network(strings.ToLower(strings.TrimSpace(string(params.Network))))
	params.Token = strings.ToLower(strings.TrimSpace(params.Token))
	params.Owner = strings.ToLower(strings.TrimSpace(params.Owner))
	params.Spender = strings.ToLower(strings.TrimSpace(params.Spender))
	params.RequiredAmount = strings.TrimSpace(params.RequiredAmount)
	return params
}

func validateCheckParams(params CheckParams) (*big.Int, error) {
	if params.Chain == "" {
		return nil, fmt.Errorf("failed to validate allowance check parameters: chain=empty")
	}
	if params.Network == "" {
		return nil, fmt.Errorf("failed to validate allowance check parameters: network=empty")
	}
	for name, value := range map[string]string{"token": params.Token, "owner": params.Owner, "spender": params.Spender} {
		if !isEVMAddress(value) {
			return nil, fmt.Errorf("failed to validate allowance check parameters: %s=invalid", name)
		}
	}
	required, ok := new(big.Int).SetString(params.RequiredAmount, 10)
	if !ok || required.Sign() <= 0 || required.BitLen() > 256 {
		return nil, fmt.Errorf("failed to validate allowance check parameters: required_amount=invalid")
	}
	return required, nil
}

func isEVMAddress(value string) bool {
	if len(value) != 42 || !strings.HasPrefix(value, "0x") {
		return false
	}
	decoded, err := hex.DecodeString(value[2:])
	return err == nil && len(decoded) == 20
}
