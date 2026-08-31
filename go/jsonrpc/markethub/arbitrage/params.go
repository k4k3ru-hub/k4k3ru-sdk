package arbitrage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"

	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

const (
	arbitrageSubscriptionNamespace = "MarketHub.Arbitrage"
	maxSymbolLength                = 32
	maxAssetLength                 = 32
)

type Params struct {
	ArbitrageType      ArbitrageType `json:"arbitrageType"`
	Chain              Chain         `json:"chain"`
	Network            Network       `json:"network"`
	Symbol             Symbol        `json:"symbol"`
	InputAsset         string        `json:"inputAsset"`
	AmountIn           string        `json:"amountIn"`
	MinimumGrossProfit string        `json:"minimumGrossProfit"`
	SourceFilter       *SourceFilter `json:"sourceFilter,omitempty"`
}

// UnmarshalJSON decodes Arbitrage parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded Arbitrage parameters.
//
// Version:
//   - 2026-08-30: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return k4k3ruSDKAppError.Tracef("failed to decode arbitrage parameters: %w: destination=null", k4k3ruSDKAppError.InvalidParameter())
	}
	type wireParams Params
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded wireParams
	if err := decoder.Decode(&decoded); err != nil {
		return invalidJSONError(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing json value")
		}
		return invalidJSONError(err)
	}
	*p = Params(decoded)
	return nil
}

// Normalize applies canonical formatting to Arbitrage parameters.
//
// Returns:
//   - Normalized Arbitrage parameters.
//
// Version:
//   - 2026-08-30: Added.
func (p Params) Normalize() Params {
	p.ArbitrageType = ArbitrageType(strings.ToLower(strings.TrimSpace(string(p.ArbitrageType))))
	p.Chain = Chain(strings.ToLower(strings.TrimSpace(string(p.Chain))))
	p.Network = Network(strings.ToLower(strings.TrimSpace(string(p.Network))))
	p.Symbol = Symbol(strings.ToUpper(strings.TrimSpace(string(p.Symbol))))
	p.InputAsset = strings.ToUpper(strings.TrimSpace(p.InputAsset))
	p.AmountIn = normalizeDecimal(p.AmountIn)
	p.MinimumGrossProfit = normalizeDecimal(p.MinimumGrossProfit)
	if p.MinimumGrossProfit == "" {
		p.MinimumGrossProfit = "0"
	}
	if p.SourceFilter != nil {
		normalized := p.SourceFilter.Normalize()
		p.SourceFilter = &normalized
	}
	return p
}

// Validate validates Arbitrage parameters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-31: Supported Solana chains.
//   - 2026-08-30: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if p.ArbitrageType != ArbitrageTypeAtomic {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: arbitrage_type=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := p.Chain.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: chain=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := p.Network.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: network=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	chainFamily, err := p.Chain.ResolveChainFamily()
	if err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: chain=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if chainFamily != k4k3ruOnchainCore.ChainFamilyEVM && chainFamily != k4k3ruOnchainCore.ChainFamilySolana {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: chain_family=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	baseAsset, quoteAsset, ok := splitSymbol(p.Symbol)
	if !ok {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: symbol=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if len(p.Symbol) > maxSymbolLength {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: symbol=too_long actual_length=%d max_length=%d", k4k3ruSDKAppError.InvalidParameter(), len(p.Symbol), maxSymbolLength)
	}
	if p.InputAsset == "" {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: input_asset=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if len(p.InputAsset) > maxAssetLength {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: input_asset=too_long actual_length=%d max_length=%d", k4k3ruSDKAppError.InvalidParameter(), len(p.InputAsset), maxAssetLength)
	}
	if p.InputAsset != baseAsset && p.InputAsset != quoteAsset {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: input_asset=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if err := validateDecimal(p.AmountIn, "amount_in", false); err != nil {
		return err
	}
	if err := validateDecimal(p.MinimumGrossProfit, "minimum_gross_profit", true); err != nil {
		return err
	}
	if p.SourceFilter != nil {
		if err := p.SourceFilter.Validate(); err != nil {
			return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w", err)
		}
	}
	return nil
}

// SubscriptionKey builds a stable Arbitrage subscription key.
//
// Returns:
//   - Canonical subscription key.
//   - Validation error.
//
// Version:
//   - 2026-08-30: Added.
func (p Params) SubscriptionKey() (string, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return "", k4k3ruSDKAppError.Tracef("failed to build arbitrage subscription key: %w", err)
	}
	return fmt.Sprintf(
		"%s:at=%s:c=%s:n=%s:s=%s:ia=%s:ai=%s:mgp=%s:src=%s",
		arbitrageSubscriptionNamespace,
		strings.ToUpper(string(p.ArbitrageType)),
		strings.ToUpper(string(p.Chain)),
		strings.ToUpper(string(p.Network)),
		p.Symbol,
		p.InputAsset,
		p.AmountIn,
		p.MinimumGrossProfit,
		p.sourceSelector(),
	), nil
}

func (p Params) sourceSelector() string {
	if p.SourceFilter == nil {
		return "*"
	}
	venues := make([]string, len(p.SourceFilter.Venues))
	for i, venue := range p.SourceFilter.Venues {
		venues[i] = strings.ToUpper(string(venue))
	}
	return strings.Join(venues, ",")
}

func splitSymbol(symbol Symbol) (string, string, bool) {
	parts := strings.Split(string(symbol), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == parts[1] {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func normalizeDecimal(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "/eE") {
		return value
	}
	decimal := new(big.Rat)
	if _, ok := decimal.SetString(value); !ok {
		return value
	}
	scale := 0
	if point := strings.IndexByte(value, '.'); point >= 0 {
		scale = len(value) - point - 1
	}
	normalized := decimal.FloatString(scale)
	if strings.Contains(normalized, ".") {
		normalized = strings.TrimRight(normalized, "0")
		normalized = strings.TrimRight(normalized, ".")
	}
	if normalized == "" || normalized == "-0" {
		return "0"
	}
	return normalized
}

func validateDecimal(value, field string, allowZero bool) error {
	if value == "" {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: %s=empty", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	if strings.ContainsAny(value, "/eE") {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: %s=invalid", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	decimal := new(big.Rat)
	if _, ok := decimal.SetString(value); !ok {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: %s=invalid", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	if decimal.Sign() < 0 || (!allowZero && decimal.Sign() == 0) {
		return k4k3ruSDKAppError.Tracef("failed to validate arbitrage parameters: %w: %s=out_of_range", k4k3ruSDKAppError.InvalidParameter(), field)
	}
	return nil
}

func invalidJSONError(err error) error {
	return k4k3ruSDKAppError.Tracef("failed to decode arbitrage parameters: %w: %w: json=invalid", k4k3ruSDKAppError.InvalidParameter(), err)
}
