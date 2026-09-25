package market

import (
	"math/big"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// AMMStateReferenceType identifies the chain-specific ordering reference for an AMM swap.
type AMMStateReferenceType string

const (
	AMMStateReferenceTypeBlockNumber AMMStateReferenceType = "block_number"
	AMMStateReferenceTypeSlot        AMMStateReferenceType = "slot"
	AMMStateReferenceTypeCheckpoint  AMMStateReferenceType = "checkpoint"
)

// AMMSwap represents one normalized swap executed against an AMM pool.
// Side, price, and quantities are expressed relative to the canonical symbol:
// buy acquires the base asset and sell disposes of the base asset.
type AMMSwap struct {
	AssetClass          AssetClass            `json:"ac"`
	MarketType          MarketType            `json:"mt"`
	Symbol              Symbol                `json:"s"`
	Venue               Venue                 `json:"v"`
	Chain               Chain                 `json:"chain"`
	PoolID              string                `json:"pool_id"`
	SwapID              string                `json:"swap_id"`
	TransactionID       string                `json:"transaction_id"`
	EventIndex          string                `json:"event_index"`
	StateReferenceType  AMMStateReferenceType `json:"state_reference_type"`
	StateReferenceValue string                `json:"state_reference_value"`
	Side                TradeSide             `json:"side"`
	Price               string                `json:"p"`
	BaseQuantity        string                `json:"base_quantity"`
	QuoteQuantity       string                `json:"quote_quantity"`
	EffectiveFeeRate    *string               `json:"effective_fee_rate,omitempty"`
	Timestamp           int64                 `json:"ts"`
}

// Validate validates a normalized AMM swap.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-09-07: Accept block-number references for Robinhood Chain swaps.
//   - 2026-08-29: Added.
func (s AMMSwap) Validate() error {
	if !isValidBBOAssetClass(s.AssetClass) {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: asset_class=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if s.MarketType != MarketTypeSpot {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: market_type=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := s.Symbol.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w", err)
	}
	if err := s.Venue.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w", err)
	}
	if err := s.Chain.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w", err)
	}
	if s.Chain.Normalize() == ChainNone {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: chain=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	for field, value := range map[string]string{
		"pool_id": s.PoolID, "swap_id": s.SwapID, "transaction_id": s.TransactionID,
		"event_index": s.EventIndex, "state_reference_value": s.StateReferenceValue,
	} {
		if strings.TrimSpace(value) == "" {
			return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: %s=empty", k4k3ruSDKAppError.InvalidParameter(), field)
		}
	}
	if !s.StateReferenceType.IsValid() {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: state_reference_type=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if !s.StateReferenceType.supportsChain(s.Chain.Normalize()) {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: state_reference_type=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if s.Side != TradeSideBuy && s.Side != TradeSideSell {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: side=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	for field, value := range map[string]string{"price": s.Price, "base_quantity": s.BaseQuantity, "quote_quantity": s.QuoteQuantity} {
		if err := validateAMMSwapDecimal(field, value, true); err != nil {
			return err
		}
	}
	if s.EffectiveFeeRate != nil {
		if err := validateAMMSwapDecimal("effective_fee_rate", *s.EffectiveFeeRate, false); err != nil {
			return err
		}
	}
	if s.Timestamp <= 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: timestamp=out_of_range", k4k3ruSDKAppError.InvalidParameter())
	}
	return nil
}

// IsValid reports whether the state-reference type is supported.
//
// Returns:
//   - True when the state-reference type is supported.
//
// Version:
//   - 2026-09-25: Copied to the SDK finance package.
//   - 2026-08-29: Added.
func (t AMMStateReferenceType) IsValid() bool {
	switch t {
	case AMMStateReferenceTypeBlockNumber, AMMStateReferenceTypeSlot, AMMStateReferenceTypeCheckpoint:
		return true
	default:
		return false
	}
}

func (t AMMStateReferenceType) supportsChain(chain Chain) bool {
	switch chain {
	case ChainEthereum, ChainBase, ChainBNB, ChainRobinhood:
		return t == AMMStateReferenceTypeBlockNumber
	case ChainSolana:
		return t == AMMStateReferenceTypeSlot
	case ChainSui:
		return t == AMMStateReferenceTypeCheckpoint
	default:
		return false
	}
}

func validateAMMSwapDecimal(field, value string, positive bool) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: %s=empty", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	if strings.ContainsAny(value, "/eE") {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: %s=invalid", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	decimal := new(big.Rat)
	if _, ok := decimal.SetString(value); !ok {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: %s=invalid", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	if decimal.Sign() < 0 || positive && decimal.Sign() == 0 {
		return k4k3ruSDKAppError.Tracef("failed to validate amm swap: %w: %s=out_of_range", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	return nil
}
