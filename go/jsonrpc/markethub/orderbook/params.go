package orderbook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

const (
	orderBookSubscriptionNamespace = "MarketHub.OrderBook"
	defaultDepth                   = uint16(3)
	maximumDepth                   = uint16(20)
	maxSymbolLength                = 16
)

type Params struct {
	AssetClass   AssetClass    `json:"assetClass,omitempty"`
	MarketType   MarketType    `json:"marketType"`
	Symbol       Symbol        `json:"symbol"`
	Depth        uint16        `json:"depth,omitempty"`
	SourceFilter *SourceFilter `json:"sourceFilter,omitempty"`
}

// UnmarshalJSON decodes order book parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded order book parameters.
//
// Version:
//   - 2026-09-05: Removed the redundant aggregation mode.
//   - 2026-09-04: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return k4k3ruSDKAppError.Tracef("failed to decode order book parameters: %w: destination=null", k4k3ruSDKAppError.InvalidParameter())
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

// Normalize applies stable defaults and canonical formatting to order book parameters.
//
// Returns:
//   - Normalized order book parameters.
//
// Version:
//   - 2026-09-04: Added.
func (p Params) Normalize() Params {
	p.AssetClass = AssetClass(strings.ToLower(strings.TrimSpace(string(p.AssetClass))))
	if p.AssetClass == AssetClassUnknown {
		p.AssetClass = AssetClassCrypto
	}
	p.MarketType = MarketType(strings.ToLower(strings.TrimSpace(string(p.MarketType))))
	p.Symbol = Symbol(strings.ToUpper(strings.TrimSpace(string(p.Symbol))))
	if p.Depth == 0 {
		p.Depth = defaultDepth
	}
	if p.SourceFilter != nil {
		normalized := p.SourceFilter.Normalize()
		if normalized.VenueCategories == nil && normalized.LiquidityModels == nil && normalized.AMMPoolChains == nil {
			p.SourceFilter = nil
		} else {
			p.SourceFilter = &normalized
		}
	}
	return p
}

// Validate validates order book parameters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-05: Removed aggregation mode validation because order books are always consolidated.
//   - 2026-09-04: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if p.AssetClass != AssetClassCrypto {
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w: asset_class=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if p.MarketType != MarketTypeSpot && p.MarketType != MarketTypePerp {
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w: market_type=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if p.Symbol == "" {
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w: symbol=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if len(p.Symbol) > maxSymbolLength {
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w: symbol=too_long actual_length=%d max_length=%d", k4k3ruSDKAppError.InvalidParameter(), len(p.Symbol), maxSymbolLength)
	}
	if p.Depth == 0 || p.Depth > maximumDepth {
		return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w: depth=out_of_range min_value=1 max_value=%d", k4k3ruSDKAppError.InvalidParameter(), maximumDepth)
	}
	if p.SourceFilter != nil {
		if err := p.SourceFilter.Validate(); err != nil {
			return k4k3ruSDKAppError.Tracef("failed to validate order book parameters: %w", err)
		}
	}
	return nil
}

// SubscriptionKey builds a stable venue-independent order book subscription key.
//
// Returns:
//   - Canonical subscription key.
//   - Validation error.
//
// Version:
//   - 2026-09-05: Removed the aggregation mode from the canonical key.
//   - 2026-09-04: Added.
func (p Params) SubscriptionKey() (string, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return "", k4k3ruSDKAppError.Tracef("failed to build order book subscription key: %w", err)
	}
	return fmt.Sprintf(
		"%s:ac=%s:mt=%s:s=%s:d=%d:src=%s",
		orderBookSubscriptionNamespace,
		strings.ToUpper(string(p.AssetClass)),
		strings.ToUpper(string(p.MarketType)),
		p.Symbol,
		p.Depth,
		p.sourceSelector(),
	), nil
}

func (p Params) sourceSelector() string {
	if p.SourceFilter == nil {
		return "*"
	}
	parts := make([]string, 0, 3)
	if p.SourceFilter.VenueCategories != nil {
		parts = append(parts, "VC="+upperJoin(p.SourceFilter.VenueCategories))
	}
	if p.SourceFilter.LiquidityModels != nil {
		parts = append(parts, "LM="+upperJoin(p.SourceFilter.LiquidityModels))
	}
	if p.SourceFilter.AMMPoolChains != nil {
		parts = append(parts, "APC="+upperJoin(p.SourceFilter.AMMPoolChains))
	}
	if len(parts) == 0 {
		return "*"
	}
	return strings.Join(parts, "|")
}

func upperJoin[T ~string](values []T) string {
	items := make([]string, len(values))
	for index, value := range values {
		items[index] = strings.ToUpper(string(value))
	}
	return strings.Join(items, ",")
}

func invalidJSONError(err error) error {
	return k4k3ruSDKAppError.Tracef("failed to decode order book parameters: %w: %w: json=invalid", k4k3ruSDKAppError.InvalidParameter(), err)
}
