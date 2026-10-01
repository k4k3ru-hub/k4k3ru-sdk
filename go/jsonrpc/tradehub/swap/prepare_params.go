package swap

import (
	"bytes"
	"encoding/json"
	"errors"
	sui "github.com/k4k3ru-hub/onchain/go/sui"
	"io"
	"math"
	"math/big"
	"strings"
	"time"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	k4k3ruSDKMarketHubArbitrage "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/arbitrage"
	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

type PrepareParams struct {
	ScalpingRun         *ScalpingRunReference                       `json:"scalpingRun,omitempty"`
	ScalpingExecutionID string                                      `json:"scalpingExecutionId,omitempty"`
	ScalpingRevision    string                                      `json:"scalpingRevision,omitempty"`
	Simulate            bool                                        `json:"simulate,omitempty"`
	AmountLimit         string                                      `json:"amountLimit,omitempty"`
	EVM                 *EVMPrepareParams                           `json:"evm,omitempty"`
	Chain               k4k3ruOnchainCore.Chain                     `json:"chain"`
	Network             k4k3ruOnchainCore.Network                   `json:"network"`
	Venue               k4k3ruSDKMarketHubArbitrage.Venue           `json:"venue"`
	PoolID              string                                      `json:"poolId"`
	TokenInAssetID      string                                      `json:"tokenInAssetId"`
	TokenOutAssetID     string                                      `json:"tokenOutAssetId"`
	Amount              string                                      `json:"amount"`
	Kind                Kind                                        `json:"kind"`
	MaximumSlippageBPS  *uint64                                     `json:"maximumSlippageBps,omitempty"`
	Signer              string                                      `json:"signer"`
	Recipient           string                                      `json:"recipient"`
	ApprovalAmount      string                                      `json:"approvalAmount,omitempty"`
	Sui                 *SuiPrepareParams                           `json:"sui,omitempty"`
	StateReference      *k4k3ruSDKMarketHubArbitrage.StateReference `json:"stateReference,omitempty"`
	ExecutionTTLMS      *uint64                                     `json:"executionTtlMs"`
	IdempotencyKey      string                                      `json:"idempotencyKey"`
}

// Normalize applies canonical formatting to swap preparation parameters.
//
// Returns:
//   - Normalized parameters.
//
// Version:
//   - 2026-10-02: Copy the optional per-order Run reference.
//   - 2026-09-29: Normalize the optional managed inventory reference.
//   - 2026-09-10: Added.
//   - 2026-09-24: Support explicit Sui coin and gas selections.
//   - 2026-09-27: Normalize explicit preparation limits.
func (p PrepareParams) Normalize() PrepareParams {
	if p.ScalpingRun != nil {
		n := p.ScalpingRun.Normalize()
		p.ScalpingRun = &n
	}
	p.ScalpingExecutionID = strings.TrimSpace(p.ScalpingExecutionID)
	p.ScalpingRevision = strings.TrimSpace(p.ScalpingRevision)
	quote := Params{
		Chain: p.Chain, Network: p.Network, Venue: p.Venue, PoolID: p.PoolID,
		TokenInAssetID: p.TokenInAssetID, TokenOutAssetID: p.TokenOutAssetID,
		Amount: p.Amount, Kind: p.Kind, MaximumSlippageBPS: p.MaximumSlippageBPS,
	}.Normalize()
	p.Chain, p.Network, p.Venue, p.PoolID = quote.Chain, quote.Network, quote.Venue, quote.PoolID
	p.TokenInAssetID, p.TokenOutAssetID, p.Amount, p.Kind = quote.TokenInAssetID, quote.TokenOutAssetID, quote.Amount, quote.Kind
	p.Signer = strings.TrimSpace(p.Signer)
	p.Recipient = strings.TrimSpace(p.Recipient)
	p.ApprovalAmount = strings.TrimSpace(p.ApprovalAmount)
	p.AmountLimit = strings.TrimSpace(p.AmountLimit)
	if limit, ok := new(big.Int).SetString(p.AmountLimit, 10); ok {
		p.AmountLimit = limit.String()
	}
	p.IdempotencyKey = strings.TrimSpace(p.IdempotencyKey)
	if p.Sui != nil {
		normalized := p.Sui.Normalize()
		p.Sui = &normalized
	}
	if p.Chain == k4k3ruOnchainCore.ChainSui {
		for _, field := range []*string{&p.PoolID, &p.Signer, &p.Recipient} {
			if a, err := sui.ParseAddress(*field); err == nil {
				*field = a.String()
			}
		}
		for _, field := range []*string{&p.TokenInAssetID, &p.TokenOutAssetID} {
			if t, err := sui.NormalizeMoveType(*field); err == nil {
				*field = t
			}
		}
	}

	return p
}

// Validate validates swap preparation parameters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-10-02: Validate mutually exclusive Run and legacy references.
//   - 2026-09-29: Bind managed Sui swaps to an observed inventory revision.
//   - 2026-09-10: Added.
//   - 2026-09-24: Support explicit Sui coin and gas selections.
//   - 2026-09-27: Require explicit limits when simulation is disabled by default.
func (p PrepareParams) Validate() error {
	p = p.Normalize()
	if p.ScalpingRun != nil {
		if p.ScalpingExecutionID != "" || p.ScalpingRevision != "" || p.Chain != k4k3ruOnchainCore.ChainSui || p.Kind != KindExactInput {
			return invalidPrepareParameter("scalping_run=unsupported")
		}
		if err := p.ScalpingRun.Validate(); err != nil {
			return err
		}
	}
	if (p.ScalpingExecutionID == "") != (p.ScalpingRevision == "") {
		return invalidPrepareParameter("scalping_reference=invalid")
	}
	if p.ScalpingExecutionID != "" {
		if p.Chain != k4k3ruOnchainCore.ChainSui || p.Kind != KindExactInput || len(p.ScalpingExecutionID) > 128 || len(p.ScalpingRevision) != 64 {
			return invalidPrepareParameter("scalping_reference=invalid")
		}
		for _, c := range p.ScalpingRevision {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
				return invalidPrepareParameter("scalping_revision=invalid")
			}
		}
	}
	quote := Params{
		Chain: p.Chain, Network: p.Network, Venue: p.Venue, PoolID: p.PoolID,
		TokenInAssetID: p.TokenInAssetID, TokenOutAssetID: p.TokenOutAssetID,
		Amount: p.Amount, Kind: p.Kind, MaximumSlippageBPS: p.MaximumSlippageBPS,
	}
	if !p.Simulate {
		if p.MaximumSlippageBPS != nil {
			return invalidPrepareParameter("maximum_slippage_bps=unsupported")
		}
		if p.StateReference != nil {
			return invalidPrepareParameter("state_reference=unsupported")
		}
		if p.AmountLimit == "" {
			return invalidPrepareParameter("amount_limit=empty")
		}
		zero := uint64(0)
		quote.MaximumSlippageBPS = &zero // Only validate the shared market and amount fields.
	}
	if p.AmountLimit != "" {
		limit, ok := new(big.Int).SetString(p.AmountLimit, 10)
		if !ok || limit.Sign() <= 0 || limit.BitLen() > 256 || (p.Chain == k4k3ruOnchainCore.ChainSui && !limit.IsUint64()) {
			return invalidPrepareParameter("amount_limit=out_of_range")
		}
	}
	if err := quote.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade hub swap preparation parameters: %w", err)
	}
	if p.Signer == "" {
		return invalidPrepareParameter("signer=empty")
	}
	if p.Recipient == "" {
		return invalidPrepareParameter("recipient=empty")
	}
	if p.Chain == k4k3ruOnchainCore.ChainSui {
		if p.EVM != nil {
			return invalidPrepareParameter("evm=unsupported")
		}
		if err := validateSuiPrepare(p); err != nil {
			return err
		}
	} else {
		if !p.Simulate && p.EVM == nil {
			return invalidPrepareParameter("evm=null")
		}
		if p.EVM != nil && p.EVM.GasLimit == 0 {
			return invalidPrepareParameter("gas_limit=empty")
		}
		if p.Sui != nil {
			return invalidPrepareParameter("sui=invalid")
		}
		approvalAmount, ok := new(big.Int).SetString(p.ApprovalAmount, 10)
		if !ok {
			return invalidPrepareParameter("approval_amount=invalid")
		}
		if approvalAmount.Sign() <= 0 || approvalAmount.BitLen() > 256 {
			return invalidPrepareParameter("approval_amount=out_of_range")
		}
	}
	if p.ExecutionTTLMS == nil {
		return invalidPrepareParameter("execution_ttl_ms=null")
	}
	if *p.ExecutionTTLMS > uint64(math.MaxInt64/int64(time.Millisecond)) {
		return invalidPrepareParameter("execution_ttl_ms=out_of_range")
	}
	if *p.ExecutionTTLMS == 0 {
		return invalidPrepareParameter("execution_ttl_ms=empty")
	}
	if p.IdempotencyKey == "" {
		return invalidPrepareParameter("idempotency_key=empty")
	}
	if len(p.IdempotencyKey) > 128 {
		return invalidPrepareParameter("idempotency_key=too_long")
	}
	return nil
}

// UnmarshalJSON decodes swap preparation parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded parameters.
//
// Version:
//   - 2026-09-10: Added.
func (p *PrepareParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalidPrepareParameter("destination=null")
	}
	type wire PrepareParams
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded wire
	if err := decoder.Decode(&decoded); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub swap preparation parameters: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing json value")
		}
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub swap preparation parameters: %w", err)
	}
	*p = PrepareParams(decoded)
	return nil
}

func invalidPrepareParameter(state string) error {
	return k4k3ruSDKAppError.Tracef("failed to validate trade hub swap preparation parameters: %w: %s", k4k3ruSDKAppError.InvalidParameter(), state)
}
