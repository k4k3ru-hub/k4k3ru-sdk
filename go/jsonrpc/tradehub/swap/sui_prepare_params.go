package swap

import (
	"strconv"
	"strings"

	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	execution "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
	sui "github.com/k4k3ru-hub/onchain/go/sui"
)

type SuiPrepareParams struct {
	InputCoins []execution.SuiObjectRef `json:"inputCoins,omitempty"`
	GasPayment []execution.SuiObjectRef `json:"gasPayment"`
	GasBudget  string                   `json:"gasBudget"`
}

// Normalize copies coin selections and canonicalizes their references and gas budget.
//
// Version:
//   - 2026-09-24: Added.
func (p SuiPrepareParams) Normalize() SuiPrepareParams {
	p.InputCoins = append([]execution.SuiObjectRef(nil), p.InputCoins...)
	p.GasPayment = append([]execution.SuiObjectRef(nil), p.GasPayment...)
	for i := range p.InputCoins {
		p.InputCoins[i] = p.InputCoins[i].Normalize()
	}
	for i := range p.GasPayment {
		p.GasPayment[i] = p.GasPayment[i].Normalize()
	}
	p.GasBudget = strings.TrimSpace(p.GasBudget)
	if value, err := strconv.ParseUint(p.GasBudget, 10, 64); err == nil {
		p.GasBudget = strconv.FormatUint(value, 10)
	}
	return p
}

// Validate checks explicit gas payment and non-overlapping input coin references.
//
// Version:
//   - 2026-09-24: Added.
func (p SuiPrepareParams) Validate() error {
	p = p.Normalize()
	if len(p.GasPayment) == 0 {
		return invalidPrepareParameter("gas_payment=empty")
	}
	if len(p.GasPayment) > 256 || len(p.InputCoins) > 256 {
		return invalidPrepareParameter("coins=too_long")
	}
	budget, err := strconv.ParseUint(p.GasBudget, 10, 64)
	if err != nil || budget == 0 {
		return invalidPrepareParameter("gas_budget=invalid")
	}
	seen := make(map[string]bool)
	for _, refs := range [][]execution.SuiObjectRef{p.InputCoins, p.GasPayment} {
		for _, ref := range refs {
			if err := ref.Validate(); err != nil {
				return app.Tracef("failed to validate sui swap parameters: %w", err)
			}
			if seen[ref.ObjectID] {
				return invalidPrepareParameter("coin_reference=duplicate")
			}
			seen[ref.ObjectID] = true
		}
	}
	return nil
}

func validateSuiPrepare(p PrepareParams) error {
	if p.Sui == nil {
		return invalidPrepareParameter("sui=null")
	}
	if p.ApprovalAmount != "" {
		return invalidPrepareParameter("approval_amount=invalid")
	}
	if p.StateReference != nil {
		return invalidPrepareParameter("state_reference=unsupported")
	}
	if p.Kind != KindExactInput {
		return invalidPrepareParameter("kind=unsupported")
	}
	amount, err := strconv.ParseUint(p.Amount, 10, 64)
	if err != nil || amount == 0 {
		return invalidPrepareParameter("amount=out_of_range")
	}
	for _, value := range []string{p.PoolID, p.Signer, p.Recipient} {
		address, err := sui.ParseAddress(value)
		if err != nil || address.IsZero() {
			return invalidPrepareParameter("address=invalid")
		}
	}
	in, err := sui.NormalizeMoveType(p.TokenInAssetID)
	if err != nil {
		return invalidPrepareParameter("token_in_asset_id=invalid")
	}
	out, err := sui.NormalizeMoveType(p.TokenOutAssetID)
	if err != nil || in == out {
		return invalidPrepareParameter("token_out_asset_id=invalid")
	}
	native, err := sui.NormalizeMoveType("0x2::sui::SUI")
	if err != nil {
		return app.Tracef("failed to validate sui swap parameters: %w", err)
	}
	if (in == native && len(p.Sui.InputCoins) != 0) || (in != native && len(p.Sui.InputCoins) == 0) {
		return invalidPrepareParameter("input_coins=invalid")
	}
	return p.Sui.Validate()
}
