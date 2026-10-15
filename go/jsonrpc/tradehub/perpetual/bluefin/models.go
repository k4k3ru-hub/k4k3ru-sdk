// Package bluefin owns TradeHub's Bluefin Pro manual perpetual request and result models.
package bluefin

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution/prepare"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
	perpetual "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/perpetual"
	"github.com/k4k3ru-hub/onchain/go/sui"
)

type Scope struct {
	Venue          string `json:"venue"`
	Network        string `json:"network"`
	Symbol         string `json:"symbol"`
	AccountAddress string `json:"accountAddress"`
}

// Validate checks the supported product and a canonical, nonzero Sui trading account.
// Network enablement is checked separately by TradeHub.
//
// Version:
//   - 2026-10-14: Added.
func (p Scope) Validate() error {
	if p.Venue != "bluefin" || (p.Network != "testnet" && p.Network != "mainnet") || p.Symbol != "SUI/USDC" {
		return invalid("scope", "invalid")
	}
	return validateAddress(p.AccountAddress)
}
func validateAddress(value string) error {
	a, err := sui.ParseAddress(value)
	if err != nil || a.IsZero() || a.String() != value {
		return invalid("account_address", "invalid")
	}
	return nil
}
func invalid(field, state string) error {
	return validation.Invalid("validate bluefin perpetual parameters", field, state)
}

type SessionCreateParams struct {
	Scope
	SignerAddress string `json:"signerAddress"`
	SignedAtMS    int64  `json:"signedAtMillis"`
	Signature     string `json:"signature"`
}
type SessionResult struct {
	Scope
	SessionID     string `json:"sessionId"`
	SignerAddress string `json:"signerAddress"`
	ExpiresAtMS   int64  `json:"expiresAtMillis"`
}
type SessionDeleteParams struct {
	SessionID string `json:"sessionId"`
}
type SessionDeleteResult struct {
	Deleted bool `json:"deleted"`
}

// Validate checks an owner-signed login reference without exposing signature contents.
//
// Version:
//   - 2026-10-14: Added.
func (p SessionCreateParams) Validate() error {
	if err := p.Scope.Validate(); err != nil {
		return err
	}
	if p.SignerAddress != p.AccountAddress || p.SignedAtMS <= 0 {
		return invalid("login", "invalid")
	}
	if len(p.Signature) != base64.StdEncoding.EncodedLen(97) {
		return invalid("signature", "invalid")
	}
	raw, err := base64.StdEncoding.Strict().DecodeString(p.Signature)
	if err != nil || len(raw) != 97 || raw[0] != 0 {
		return invalid("signature", "invalid")
	}
	return nil
}

// Validate bounds a private, owner-scoped session reference.
//
// Version:
//   - 2026-10-14: Added.
func (p SessionDeleteParams) Validate() error {
	return validation.Text("validate bluefin session", "session_id", p.SessionID, 128)
}

type AccountParams struct {
	Scope
	SessionID string `json:"sessionId"`
}
type OrderParams struct {
	Scope
	SessionID   string `json:"sessionId"`
	ExecutionID string `json:"executionId"`
}

// Validate checks an authenticated account query.
//
// Version:
//   - 2026-10-14: Added.
func (p AccountParams) Validate() error {
	if err := p.Scope.Validate(); err != nil {
		return err
	}
	return (SessionDeleteParams{p.SessionID}).Validate()
}

// Validate checks the owner-scoped durable execution reference used for reconciliation.
//
// Version:
//   - 2026-10-14: Added.
func (p OrderParams) Validate() error {
	if err := (AccountParams{p.Scope, p.SessionID}).Validate(); err != nil {
		return err
	}
	return validation.Text("validate bluefin order query", "execution_id", p.ExecutionID, 128)
}

type Margin struct {
	Leverage uint32 `json:"leverage"`
	Mode     string `json:"mode"`
}
type OrderIntent struct {
	Scope
	SignerAddress  string                `json:"signerAddress"`
	Order          perpetual.OrderIntent `json:"order"`
	Margin         Margin                `json:"margin"`
	Salt           string                `json:"salt"`
	SignedAtMS     int64                 `json:"signedAtMillis"`
	ExecutionTTLMS uint64                `json:"executionTtlMs"`
}

// Validate checks a stable trading intent independent of transient authentication sessions.
//
// Version:
//   - 2026-10-14: Added.
func (p OrderIntent) Validate() error {
	if err := p.Scope.Validate(); err != nil {
		return err
	}
	if p.SignerAddress != p.AccountAddress || p.SignedAtMS <= 0 || p.ExecutionTTLMS == 0 || p.ExecutionTTLMS > perpetual.MaximumExecutionTTLMS {
		return invalid("signing", "invalid")
	}
	if p.Margin.Leverage == 0 || (p.Margin.Mode != "cross" && p.Margin.Mode != "isolated") {
		return invalid("margin", "invalid")
	}
	if p.Order.TimeInForce != "ioc" || (p.Order.Side != "buy" && p.Order.Side != "sell") {
		return invalid("order", "invalid")
	}
	for _, value := range []string{p.Order.Quantity, p.Order.LimitPrice} {
		n, err := validation.Number("validate bluefin order", "amount", value, false, false)
		if err != nil {
			return err
		}
		if n.Sign() <= 0 {
			return invalid("amount", "out_of_range")
		}
	}
	if err := validation.Text("validate bluefin order", "client_order_id", p.Order.ClientOrderID, 128); err != nil {
		return err
	}
	if _, err := validation.Number("validate bluefin order", "salt", p.Salt, true, false); err != nil {
		return err
	}
	n, err := strconv.ParseUint(p.Salt, 10, 64)
	if err != nil || n == 0 || strconv.FormatUint(n, 10) != p.Salt {
		return invalid("salt", "out_of_range")
	}
	return nil
}

// PrepareParams wraps the stable intent in the common execution API with a current authentication reference.
//
// Version:
//   - 2026-10-14: Added.
func (p OrderIntent) PrepareParams(sessionID string) (prepare.Params, error) {
	if err := p.Validate(); err != nil {
		return prepare.Params{}, err
	}
	ttl := p.ExecutionTTLMS
	order := p.Order
	r := prepare.Params{Kind: prepare.KindPerpetual, Perpetual: &prepare.PerpetualParams{Venue: p.Venue, Network: p.Network, Symbol: p.Symbol, AccountAddress: p.AccountAddress, SignerAddress: p.SignerAddress, Kind: "order", Order: &order, Margin: &prepare.MarginIntent{Leverage: p.Margin.Leverage, Mode: p.Margin.Mode}, ExecutionTTLMS: &ttl, Signing: prepare.SigningContext{Bluefin: &prepare.BluefinSigning{SessionID: sessionID, Salt: p.Salt, SignedAtMS: p.SignedAtMS}}}}
	if err := r.Validate(); err != nil {
		return prepare.Params{}, fmt.Errorf("failed to create bluefin preparation parameters: %w", err)
	}
	return r, nil
}

// Intent extracts a validated stable Bluefin intent from the common preparation contract.
//
// Version:
//   - 2026-10-14: Added.
func Intent(p prepare.PerpetualParams) (OrderIntent, error) {
	if err := (prepare.Params{Kind: prepare.KindPerpetual, Perpetual: &p}).Validate(); err != nil {
		return OrderIntent{}, err
	}
	if p.Venue != "bluefin" || p.Order == nil || p.Margin == nil || p.Signing.Bluefin == nil {
		return OrderIntent{}, invalid("intent", "invalid")
	}
	ttl := perpetual.DefaultExecutionTTLMS
	if p.ExecutionTTLMS != nil {
		ttl = *p.ExecutionTTLMS
	}
	r := OrderIntent{Scope: Scope{p.Venue, p.Network, p.Symbol, p.AccountAddress}, SignerAddress: p.SignerAddress, Order: *p.Order, Margin: Margin{p.Margin.Leverage, p.Margin.Mode}, Salt: p.Signing.Bluefin.Salt, SignedAtMS: p.Signing.Bluefin.SignedAtMS, ExecutionTTLMS: ttl}
	return r, r.Validate()
}

type Contract struct {
	VenueSymbol    string `json:"venueSymbol"`
	IDSID          string `json:"idsId"`
	StepSizeE9     string `json:"stepSizeE9"`
	TickSizeE9     string `json:"tickSizeE9"`
	CollateralType string `json:"collateralType"`
}
type Prepared struct {
	Version      uint16      `json:"version"`
	ExecutionID  string      `json:"executionId"`
	Intent       OrderIntent `json:"intent"`
	Contract     Contract    `json:"contract"`
	SigningBytes string      `json:"signingBytes"`
	Digest       string      `json:"digest"`
	OrderHash    string      `json:"orderHash"`
	PreparedAtMS int64       `json:"preparedAtMillis"`
	ExpiresAtMS  int64       `json:"expiresAtMillis"`
}
type Signature struct {
	SerializedSignature string `json:"serializedSignature"`
}

// Validate checks the supported serialized Ed25519 signature without exposing its contents.
//
// Version:
//   - 2026-10-14: Added.
func (p Signature) Validate() error {
	if len(p.SerializedSignature) != base64.StdEncoding.EncodedLen(97) {
		return invalid("signature", "invalid")
	}
	value, err := base64.StdEncoding.Strict().DecodeString(p.SerializedSignature)
	if err != nil || len(value) != 97 || value[0] != 0 {
		return invalid("signature", "invalid")
	}
	return nil
}

// UnmarshalJSON rejects extra signature fields, duplicate keys and missing values.
//
// Version:
//   - 2026-10-14: Added.
func (p *Signature) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type plain Signature
	var value plain
	if err := validation.Decode(data, &value, "serializedSignature"); err != nil {
		return &decodeError{err}
	}
	*p = Signature(value)
	return p.Validate()
}

type Position struct {
	Quantity          string `json:"quantity"`
	AverageEntryPrice string `json:"averageEntryPrice"`
	LiquidationPrice  string `json:"liquidationPrice"`
	Leverage          string `json:"leverage"`
	MarginMode        string `json:"marginMode"`
	MarginRequired    string `json:"marginRequired"`
	UnrealizedPnL     string `json:"unrealizedPnl"`
	ObservedAtMS      int64  `json:"observedAtMillis"`
}
type AccountResult struct {
	Scope
	TradingSupported    bool      `json:"tradingSupported"`
	USDCTotal           string    `json:"usdcTotal"`
	MarginAvailable     string    `json:"marginAvailable"`
	OrderMarginRequired string    `json:"orderMarginRequired"`
	AccountValue        string    `json:"accountValue"`
	Position            *Position `json:"position,omitempty"`
	ObservedAtMS        int64     `json:"observedAtMillis"`
}
type Fee struct {
	Kind   string `json:"kind"`
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
}
type Fill struct {
	ID            string `json:"id"`
	Quantity      string `json:"quantity"`
	Price         string `json:"price"`
	QuoteQuantity string `json:"quoteQuantity"`
	Fees          []Fee  `json:"fees"`
	OccurredAtMS  int64  `json:"occurredAtMillis"`
	Side          string `json:"side"`
	FeesComplete  bool   `json:"feesComplete"`
}
type OrderResult struct {
	Scope
	ExecutionID            string `json:"executionId"`
	OrderHash              string `json:"orderHash"`
	ClientOrderID          string `json:"clientOrderId"`
	Status                 string `json:"status"`
	RequestedQuantity      string `json:"requestedQuantity"`
	FilledQuantity         string `json:"filledQuantity"`
	AveragePrice           string `json:"averagePrice,omitempty"`
	Fills                  []Fill `json:"fills"`
	FillsComplete          bool   `json:"fillsComplete"`
	FeesComplete           bool   `json:"feesComplete"`
	ReconciliationRequired bool   `json:"reconciliationRequired"`
	ObservedAtMS           int64  `json:"observedAtMillis"`
}

// Validate checks the client-verifiable preparation envelope before any signature is requested.
//
// Version:
//   - 2026-10-14: Added.
func (p Prepared) Validate() error {
	if err := p.Intent.Validate(); err != nil {
		return err
	}
	if p.Version != 1 || p.PreparedAtMS <= 0 || p.ExpiresAtMS <= p.PreparedAtMS || uint64(p.ExpiresAtMS-p.PreparedAtMS) != p.Intent.ExecutionTTLMS {
		return invalid("preparation", "invalid")
	}
	if err := validation.Text("validate bluefin preparation", "execution_id", p.ExecutionID, 128); err != nil {
		return err
	}
	for _, hash := range []string{p.Digest, p.OrderHash} {
		if len(hash) != 64 {
			return invalid("digest", "invalid")
		}
		if _, err := hex.DecodeString(hash); err != nil {
			return invalid("digest", "invalid")
		}
	}
	if len(p.SigningBytes) > 16<<10 {
		return invalid("signing_bytes", "too_long")
	}
	message, err := base64.StdEncoding.Strict().DecodeString(p.SigningBytes)
	if err != nil || len(message) == 0 {
		return invalid("signing_bytes", "invalid")
	}
	if p.Contract.VenueSymbol != "SUI-PERP" || p.Contract.CollateralType == "" {
		return invalid("contract", "invalid")
	}
	if err := validateAddress(p.Contract.IDSID); err != nil {
		return err
	}
	return nil
}

type decodeError struct{ cause error }

// Error hides authentication values from JSON decoding diagnostics.
//
// Version:
//   - 2026-10-14: Added.
func (e *decodeError) Error() string {
	return "failed to decode bluefin perpetual parameters: json=invalid"
}

// Unwrap retains the original validation error for inspection.
//
// Version:
//   - 2026-10-14: Added.
func (e *decodeError) Unwrap() error { return e.cause }

// UnmarshalJSON rejects unknown, duplicate and missing login fields without exposing their contents.
//
// Version:
//   - 2026-10-14: Added.
func (p *SessionCreateParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type plain SessionCreateParams
	var value plain
	if err := validation.Decode(data, &value, "venue", "network", "symbol", "accountAddress", "signerAddress", "signedAtMillis", "signature"); err != nil {
		return &decodeError{err}
	}
	*p = SessionCreateParams(value)
	return p.Validate()
}

// UnmarshalJSON decodes a bounded session deletion reference.
//
// Version:
//   - 2026-10-14: Added.
func (p *SessionDeleteParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type plain SessionDeleteParams
	var value plain
	if err := validation.Decode(data, &value, "sessionId"); err != nil {
		return &decodeError{err}
	}
	*p = SessionDeleteParams(value)
	return p.Validate()
}

// UnmarshalJSON decodes a strict Bluefin account query.
//
// Version:
//   - 2026-10-14: Added.
func (p *AccountParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type plain AccountParams
	var value plain
	if err := validation.Decode(data, &value, "venue", "network", "symbol", "accountAddress", "sessionId"); err != nil {
		return &decodeError{err}
	}
	*p = AccountParams(value)
	return p.Validate()
}

// UnmarshalJSON decodes a strict owner-scoped Bluefin order query.
//
// Version:
//   - 2026-10-14: Added.
func (p *OrderParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type plain OrderParams
	var value plain
	if err := validation.Decode(data, &value, "venue", "network", "symbol", "accountAddress", "sessionId", "executionId"); err != nil {
		return &decodeError{err}
	}
	*p = OrderParams(value)
	return p.Validate()
}

var _ json.Unmarshaler = (*SessionCreateParams)(nil)
