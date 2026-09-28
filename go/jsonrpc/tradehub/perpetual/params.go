// Package perpetual defines the authenticated manual perpetual trading RPCs.
package perpetual

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

type Scope struct {
	Venue          string `json:"venue"`
	Network        string `json:"network"`
	Symbol         string `json:"symbol"`
	AccountAddress string `json:"accountAddress"`
}
type AccountParams struct{ Scope }
type PrepareParams struct {
	Scope
	SignerAddress string          `json:"signerAddress"`
	Kind          string          `json:"kind"`
	Nonce         uint64          `json:"nonce"`
	Order         *OrderIntent    `json:"order,omitempty"`
	Leverage      *LeverageIntent `json:"leverage,omitempty"`
}
type OrderIntent struct {
	Side          string `json:"side"`
	Quantity      string `json:"quantity"`
	LimitPrice    string `json:"limitPrice"`
	TimeInForce   string `json:"timeInForce"`
	ReduceOnly    bool   `json:"reduceOnly"`
	ClientOrderID string `json:"clientOrderId"`
}
type LeverageIntent struct {
	Value      uint32 `json:"value"`
	MarginMode string `json:"marginMode"`
}
type Signature struct {
	R string `json:"r"`
	S string `json:"s"`
	V uint8  `json:"v"`
}
type SubmitParams struct {
	PreparationID string    `json:"preparationId"`
	Digest        string    `json:"digest"`
	PreparedToken string    `json:"preparedToken"`
	Signature     Signature `json:"signature"`
}
type OrderParams struct {
	Scope
	OrderID       string `json:"orderId,omitempty"`
	ClientOrderID string `json:"clientOrderId,omitempty"`
}

func invalid(field string) error {
	return validation.Invalid("validate perpetual parameters", field, "invalid")
}

func hexValue(value string, size int) bool {
	if len(value) != 2+size*2 || !strings.HasPrefix(value, "0x") {
		return false
	}
	_, err := hex.DecodeString(value[2:])
	return err == nil
}

func validateHex(field, value string, size int) error {
	if err := validation.Text("validate perpetual parameters", field, value, 2+size*2); err != nil {
		return err
	}
	if !hexValue(value, size) {
		return invalid(field)
	}
	return nil
}

// Validate validates the initially supported venue, network, market and trading address.
//
// Version:
//   - 2026-09-28: Added.
func (p Scope) Validate() error {
	for _, f := range []struct{ name, value string }{{"venue", p.Venue}, {"network", p.Network}, {"symbol", p.Symbol}} {
		if err := validation.Text("validate perpetual scope", f.name, f.value, 64); err != nil {
			return err
		}
	}
	if p.Venue != "hyperliquid" || p.Network != "testnet" || p.Symbol != "SUI/USDC" {
		return invalid("scope")
	}
	if err := validateHex("account_address", p.AccountAddress, 20); err != nil {
		return err
	}
	if strings.EqualFold(p.AccountAddress, "0x"+strings.Repeat("0", 40)) {
		return invalid("account_address")
	}
	return nil
}

// Validate validates an order or a separate leverage update without allocating a nonce.
//
// Version:
//   - 2026-09-28: Added.
func (p PrepareParams) Validate() error {
	if err := p.Scope.Validate(); err != nil {
		return fmt.Errorf("failed to validate preparation: %w", err)
	}
	if err := validateHex("signer_address", p.SignerAddress, 20); err != nil {
		return err
	}
	if strings.EqualFold(p.SignerAddress, "0x"+strings.Repeat("0", 40)) {
		return invalid("signer_address")
	}
	if p.Nonce == 0 {
		return validation.Invalid("validate preparation", "nonce", "empty")
	}
	switch p.Kind {
	case "order":
		if p.Order == nil || p.Leverage != nil {
			return invalid("intent")
		}
		o := p.Order
		if err := validateHex("client_order_id", o.ClientOrderID, 16); err != nil {
			return err
		}
		if (o.Side != "buy" && o.Side != "sell") || o.TimeInForce != "ioc" {
			return invalid("order")
		}
		for _, f := range []struct{ name, value string }{{"quantity", o.Quantity}, {"limit_price", o.LimitPrice}} {
			n, err := validation.Number("validate order", f.name, f.value, false, false)
			if err != nil {
				return fmt.Errorf("failed to validate preparation: %w", err)
			}
			if n.Sign() <= 0 {
				return invalid(f.name)
			}
		}
	case "leverage":
		if p.Leverage == nil || p.Order != nil {
			return invalid("intent")
		}
		if p.Leverage.Value == 0 || (p.Leverage.MarginMode != "cross" && p.Leverage.MarginMode != "isolated") {
			return invalid("leverage")
		}
	default:
		return invalid("kind")
	}
	return nil
}

// Validate validates token and signature encoding; cryptographic checks occur at Submit.
//
// Version:
//   - 2026-09-28: Added.
func (p SubmitParams) Validate() error {
	if err := validateHex("preparation_id", p.PreparationID, 16); err != nil {
		return err
	}
	if err := validateHex("digest", p.Digest, 32); err != nil {
		return err
	}
	if err := validation.Text("validate submission", "prepared_token", p.PreparedToken, 256<<10); err != nil {
		return fmt.Errorf("failed to validate submission: %w", err)
	}
	for _, v := range []string{p.Signature.R, p.Signature.S} {
		if err := validation.Text("validate submission", "signature_scalar", v, 66); err != nil {
			return err
		}
		if len(v) < 3 || len(v) > 66 || !strings.HasPrefix(v, "0x") {
			return invalid("signature")
		}
		for _, c := range v[2:] {
			if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
				return invalid("signature")
			}
		}
	}
	if p.Signature.V != 27 && p.Signature.V != 28 {
		return invalid("signature")
	}
	return nil
}

// Validate requires exactly one venue or client order identifier.
//
// Version:
//   - 2026-09-28: Added.
func (p OrderParams) Validate() error {
	if err := p.Scope.Validate(); err != nil {
		return fmt.Errorf("failed to validate order query: %w", err)
	}
	if (p.OrderID == "") == (p.ClientOrderID == "") {
		return invalid("order_identifier")
	}
	if p.ClientOrderID != "" && !hexValue(p.ClientOrderID, 16) {
		return invalid("client_order_id")
	}
	if p.OrderID != "" {
		n, err := strconv.ParseUint(p.OrderID, 10, 64)
		if err != nil || n == 0 || strconv.FormatUint(n, 10) != p.OrderID {
			return invalid("order_id")
		}
	}
	return nil
}

// DecodeError preserves the error chain without exposing submitted tokens or values in logs.
type DecodeError struct{ cause error }

// Error returns a safe description of invalid request JSON.
//
// Version:
//   - 2026-09-28: Added.
func (e *DecodeError) Error() string { return "failed to decode perpetual parameters: json=invalid" }

// Unwrap returns the inspectable decoding error.
//
// Version:
//   - 2026-09-28: Added.
func (e *DecodeError) Unwrap() error { return e.cause }

func decode(data []byte, destination any, required ...string) error {
	if err := validation.Decode(data, destination, required...); err != nil {
		return &DecodeError{cause: err}
	}
	return nil
}

// UnmarshalJSON decodes a strict account query.
//
// Version:
//   - 2026-09-28: Added.
func (p *AccountParams) UnmarshalJSON(data []byte) error {
	type plain AccountParams
	return decode(data, (*plain)(p), "venue", "network", "symbol", "accountAddress")
}

// UnmarshalJSON decodes a strict preparation intent.
//
// Version:
//   - 2026-09-28: Added.
func (p *PrepareParams) UnmarshalJSON(data []byte) error {
	type plain PrepareParams
	return decode(data, (*plain)(p), "venue", "network", "symbol", "accountAddress", "signerAddress", "kind", "nonce")
}

// UnmarshalJSON requires an explicit reduce-only choice and IOC constraints.
//
// Version:
//   - 2026-09-28: Added.
func (p *OrderIntent) UnmarshalJSON(data []byte) error {
	type plain OrderIntent
	return decode(data, (*plain)(p), "side", "quantity", "limitPrice", "timeInForce", "reduceOnly", "clientOrderId")
}

// UnmarshalJSON requires an explicit leverage and margin mode.
//
// Version:
//   - 2026-09-28: Added.
func (p *LeverageIntent) UnmarshalJSON(data []byte) error {
	type plain LeverageIntent
	return decode(data, (*plain)(p), "value", "marginMode")
}

// UnmarshalJSON decodes a strict signature without logging its values.
//
// Version:
//   - 2026-09-28: Added.
func (p *Signature) UnmarshalJSON(data []byte) error {
	type plain Signature
	return decode(data, (*plain)(p), "r", "s", "v")
}

// UnmarshalJSON decodes a strict signed submission.
//
// Version:
//   - 2026-09-28: Added.
func (p *SubmitParams) UnmarshalJSON(data []byte) error {
	type plain SubmitParams
	return decode(data, (*plain)(p), "preparationId", "digest", "preparedToken", "signature")
}

// UnmarshalJSON decodes a strict order query.
//
// Version:
//   - 2026-09-28: Added.
func (p *OrderParams) UnmarshalJSON(data []byte) error {
	type plain OrderParams
	return decode(data, (*plain)(p), "venue", "network", "symbol", "accountAddress")
}

var _ json.Unmarshaler = (*PrepareParams)(nil)
