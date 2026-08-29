package aggregation

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
	aggregationSubscriptionNamespace = "MarketHub.Aggregation"
	maxSymbolLength                  = 16
)

type Params struct {
	AssetClass      AssetClass               `json:"assetClass,omitempty"`
	MarketType      MarketType               `json:"marketType"`
	Symbol          Symbol                   `json:"symbol"`
	AggregationMode AggregationMode          `json:"aggregationMode,omitempty"`
	SourceFilter    *AggregationSourceFilter `json:"sourceFilter,omitempty"`
}

// UnmarshalJSON decodes aggregation parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded aggregation parameters.
//
// Version:
//   - 2026-08-29: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return k4k3ruSDKAppError.Tracef("failed to decode aggregation parameters: %w: destination=null", k4k3ruSDKAppError.InvalidParameter())
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

// Normalize applies stable defaults and canonical formatting to aggregation parameters.
//
// Returns:
//   - Normalized aggregation parameters.
//
// Version:
//   - 2026-08-29: Added.
func (p Params) Normalize() Params {
	p.AssetClass = AssetClass(strings.ToLower(strings.TrimSpace(string(p.AssetClass))))
	if p.AssetClass == AssetClassUnknown {
		p.AssetClass = AssetClassCrypto
	}
	p.MarketType = MarketType(strings.ToLower(strings.TrimSpace(string(p.MarketType))))
	p.Symbol = Symbol(strings.ToUpper(strings.TrimSpace(string(p.Symbol))))
	p.AggregationMode = AggregationMode(strings.ToLower(strings.TrimSpace(string(p.AggregationMode))))
	if p.AggregationMode == AggregationModeUnknown {
		p.AggregationMode = AggregationModeCompositeMid
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

// Validate validates aggregation parameters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-29: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if p.AssetClass != AssetClassCrypto {
		return k4k3ruSDKAppError.Tracef("failed to validate aggregation parameters: %w: asset_class=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if p.MarketType != MarketTypeSpot && p.MarketType != MarketTypePerp {
		return k4k3ruSDKAppError.Tracef("failed to validate aggregation parameters: %w: market_type=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if p.Symbol == "" {
		return k4k3ruSDKAppError.Tracef("failed to validate aggregation parameters: %w: symbol=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if len(p.Symbol) > maxSymbolLength {
		return k4k3ruSDKAppError.Tracef("failed to validate aggregation parameters: %w: symbol=too_long actual_length=%d max_length=%d", k4k3ruSDKAppError.InvalidParameter(), len(p.Symbol), maxSymbolLength)
	}
	if p.AggregationMode != AggregationModeCompositeMid {
		return k4k3ruSDKAppError.Tracef("failed to validate aggregation parameters: %w: aggregation_mode=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	if p.SourceFilter != nil {
		if err := p.SourceFilter.Validate(); err != nil {
			return k4k3ruSDKAppError.Tracef("failed to validate aggregation parameters: %w", err)
		}
	}
	return nil
}

// SubscriptionKey builds a stable venue-independent aggregation subscription key.
//
// Returns:
//   - Canonical subscription key.
//   - Validation error.
//
// Version:
//   - 2026-08-29: Added.
func (p Params) SubscriptionKey() (string, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return "", k4k3ruSDKAppError.Tracef("failed to build aggregation subscription key: %w", err)
	}
	return fmt.Sprintf(
		"%s:ac=%s:mt=%s:s=%s:am=%s:src=%s",
		aggregationSubscriptionNamespace,
		strings.ToUpper(string(p.AssetClass)),
		strings.ToUpper(string(p.MarketType)),
		string(p.Symbol),
		strings.ToUpper(string(p.AggregationMode)),
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
	for i, value := range values {
		items[i] = strings.ToUpper(string(value))
	}
	return strings.Join(items, ",")
}

func invalidJSONError(err error) error {
	return k4k3ruSDKAppError.Tracef("failed to decode aggregation parameters: %w: %w: json=invalid", k4k3ruSDKAppError.InvalidParameter(), err)
}
