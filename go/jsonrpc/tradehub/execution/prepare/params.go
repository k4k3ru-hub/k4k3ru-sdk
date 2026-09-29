// Package prepare owns the public TradeHub.Execution.Prepare contract.
// Spread's existing internal execution parameters remain in package execution.
package prepare

import (
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/perpetual"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/swap"
)

type Kind string

const (
	KindSwap      Kind = "swap"
	KindPerpetual Kind = "perpetual"
)

type Params struct {
	Kind      Kind                `json:"kind"`
	Swap      *swap.PrepareParams `json:"swap,omitempty"`
	Perpetual *PerpetualParams    `json:"perpetual,omitempty"`
}

// PerpetualParams separates trading intent from venue-specific signing inputs.
// Accepted venues, networks and order capabilities are enforced by the selected adapter.
type PerpetualParams struct {
	Venue          string                    `json:"venue"`
	Network        string                    `json:"network"`
	Symbol         string                    `json:"symbol"`
	AccountAddress string                    `json:"accountAddress"`
	SignerAddress  string                    `json:"signerAddress"`
	Kind           string                    `json:"kind"`
	Order          *perpetual.OrderIntent    `json:"order,omitempty"`
	Leverage       *perpetual.LeverageIntent `json:"leverage,omitempty"`
	Signing        SigningContext            `json:"signing"`
}

type SigningContext struct {
	Hyperliquid *HyperliquidSigning `json:"hyperliquid,omitempty"`
}
type HyperliquidSigning struct {
	Nonce uint64 `json:"nonce"`
}

// Validate validates the discriminated intent; adapters apply venue capabilities.
//
// Version:
//   - 2026-09-29: Added.
func (p Params) Validate() error {
	switch p.Kind {
	case KindSwap:
		if p.Swap == nil || p.Perpetual != nil {
			return invalid("intent")
		}
		if err := p.Swap.Validate(); err != nil {
			return fmt.Errorf("failed to validate execution preparation: %w", err)
		}
	case KindPerpetual:
		if p.Perpetual == nil || p.Swap != nil {
			return invalid("intent")
		}
		v := p.Perpetual
		for _, f := range []struct{ name, value string }{{"venue", v.Venue}, {"network", v.Network}, {"symbol", v.Symbol}, {"account_address", v.AccountAddress}, {"signer_address", v.SignerAddress}} {
			if err := validation.Text("validate execution preparation", f.name, f.value, 256); err != nil {
				return err
			}
		}
		switch v.Kind {
		case "order":
			if v.Order == nil || v.Leverage != nil {
				return invalid("intent")
			}
		case "leverage":
			if v.Leverage == nil || v.Order != nil {
				return invalid("intent")
			}
		default:
			return invalid("perpetual_kind")
		}
		if v.Signing.Hyperliquid != nil && (v.Venue != "hyperliquid" || v.Signing.Hyperliquid.Nonce == 0) {
			return invalid("signing")
		}
	default:
		return invalid("kind")
	}
	return nil
}

// UnmarshalJSON rejects unknown, duplicate, null and mixed preparation variants.
//
// Version:
//   - 2026-09-29: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination")
	}
	type plain Params
	var decoded plain
	if err := validation.Decode(data, &decoded, "kind"); err != nil {
		return &decodeError{cause: err}
	}
	*p = Params(decoded)
	return p.Validate()
}

func invalid(field string) error {
	return validation.Invalid("validate execution preparation", field, "invalid")
}

type decodeError struct{ cause error }

// Error describes invalid JSON without disclosing input values.
//
// Version:
//   - 2026-09-29: Added.
func (e *decodeError) Error() string { return "failed to decode execution preparation: json=invalid" }

// Unwrap preserves the parsing error for inspection.
//
// Version:
//   - 2026-09-29: Added.
func (e *decodeError) Unwrap() error { return e.cause }
