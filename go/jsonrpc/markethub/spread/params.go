package spread

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

const spreadSubscriptionNamespace = "MarketHub.Spread"

type Params struct {
	AssetClass            AssetClass    `json:"assetClass,omitempty"`
	Symbol                Symbol        `json:"symbol"`
	BaseAsset             Asset         `json:"baseAsset"`
	Quantity              string        `json:"quantity"`
	MinimumGrossSpreadBps string        `json:"minimumGrossSpreadBps,omitempty"`
	RouteFamilies         []RouteFamily `json:"routeFamilies,omitempty"`
	SourceFilter          *SourceFilter `json:"sourceFilter,omitempty"`
}

// UnmarshalJSON decodes Spread parameters and rejects unknown fields.
//
// Version:
//   - 2026-09-05: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return k4k3ruSDKAppError.Tracef("failed to decode spread parameters: %w: destination=null", k4k3ruSDKAppError.InvalidParameter())
	}
	type wire Params
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var v wire
	if err := d.Decode(&v); err != nil {
		return invalidJSONError(err)
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing json value")
		}
		return invalidJSONError(err)
	}
	*p = Params(v)
	return nil
}

// Normalize applies stable Spread defaults and canonical formatting.
//
// Version:
//   - 2026-09-05: Added.
func (p Params) Normalize() Params {
	p.AssetClass = AssetClass(strings.ToLower(strings.TrimSpace(string(p.AssetClass))))
	if p.AssetClass == "" {
		p.AssetClass = AssetClassCrypto
	}
	p.Symbol = Symbol(strings.ToUpper(strings.TrimSpace(string(p.Symbol))))
	p.BaseAsset = Asset(strings.ToUpper(strings.TrimSpace(string(p.BaseAsset))))
	p.Quantity = normalizeDecimal(p.Quantity)
	p.MinimumGrossSpreadBps = normalizeDecimal(p.MinimumGrossSpreadBps)
	if p.MinimumGrossSpreadBps == "" {
		p.MinimumGrossSpreadBps = "0"
	}
	if p.RouteFamilies == nil || len(p.RouteFamilies) == 0 {
		p.RouteFamilies = normalizeValues(allRouteFamilies)
	} else {
		p.RouteFamilies = normalizeValues(p.RouteFamilies)
	}
	if p.SourceFilter != nil {
		f := p.SourceFilter.Normalize()
		if f.VenueCategories == nil && f.LiquidityModels == nil && f.AMMPoolChains == nil {
			p.SourceFilter = nil
		} else {
			p.SourceFilter = &f
		}
	}
	return p
}

// Validate validates Spread parameters.
//
// Version:
//   - 2026-09-05: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if p.AssetClass != AssetClassCrypto {
		return invalid("asset_class=invalid")
	}
	parts := strings.Split(string(p.Symbol), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == parts[1] {
		return invalid("symbol=invalid")
	}
	if len(p.Symbol) > 32 {
		return invalid(fmt.Sprintf("symbol=too_long actual_length=%d max_length=32", len(p.Symbol)))
	}
	if p.BaseAsset == "" {
		return invalid("base_asset=empty")
	}
	if string(p.BaseAsset) != parts[0] {
		return invalid("base_asset=invalid")
	}
	if err := validateDecimal(p.Quantity, "quantity", false); err != nil {
		return err
	}
	if err := validateDecimal(p.MinimumGrossSpreadBps, "minimum_gross_spread_bps", true); err != nil {
		return err
	}
	if err := validateUnique(p.RouteFamilies, func(v RouteFamily) bool {
		return v == RouteFamilySpotSpot || v == RouteFamilySpotPerp || v == RouteFamilyPerpSpot || v == RouteFamilyPerpPerp
	}); err != nil {
		return invalid("route_family=invalid")
	}
	if p.SourceFilter != nil {
		if err := p.SourceFilter.Validate(); err != nil {
			return k4k3ruSDKAppError.Tracef("failed to validate spread parameters: %w", err)
		}
	}
	return nil
}

// SubscriptionKey builds a stable Spread subscription key.
//
// Version:
//   - 2026-09-05: Added.
func (p Params) SubscriptionKey() (string, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return "", k4k3ruSDKAppError.Tracef("failed to build spread subscription key: %w", err)
	}
	return fmt.Sprintf("%s:ac=%s:s=%s:ba=%s:q=%s:mgsb=%s:rf=%s:src=%s", spreadSubscriptionNamespace, strings.ToUpper(string(p.AssetClass)), p.Symbol, p.BaseAsset, p.Quantity, p.MinimumGrossSpreadBps, upperJoin(p.RouteFamilies), p.sourceSelector()), nil
}

func (p Params) sourceSelector() string {
	if p.SourceFilter == nil {
		return "*"
	}
	f := p.SourceFilter
	parts := make([]string, 0, 3)
	if f.VenueCategories != nil {
		parts = append(parts, "VC="+upperJoin(f.VenueCategories))
	}
	if f.LiquidityModels != nil {
		parts = append(parts, "LM="+upperJoin(f.LiquidityModels))
	}
	if f.AMMPoolChains != nil {
		parts = append(parts, "APC="+upperJoin(f.AMMPoolChains))
	}
	return strings.Join(parts, "|")
}
func upperJoin[T ~string](values []T) string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = strings.ToUpper(string(value))
	}
	return strings.Join(out, ",")
}
func normalizeDecimal(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "/eE") {
		return value
	}
	v := new(big.Rat)
	if _, ok := v.SetString(value); !ok {
		return value
	}
	scale := 0
	if i := strings.IndexByte(value, '.'); i >= 0 {
		scale = len(value) - i - 1
	}
	out := v.FloatString(scale)
	if strings.ContainsRune(out, '.') {
		out = strings.TrimRight(strings.TrimRight(out, "0"), ".")
	}
	if out == "" || out == "-0" {
		return "0"
	}
	return out
}
func validateDecimal(value, field string, allowZero bool) error {
	if value == "" {
		return invalid(field + "=empty")
	}
	if strings.ContainsAny(value, "/eE") {
		return invalid(field + "=invalid")
	}
	v := new(big.Rat)
	if _, ok := v.SetString(value); !ok {
		return invalid(field + "=invalid")
	}
	if v.Sign() < 0 || (!allowZero && v.Sign() == 0) {
		return invalid(field + "=out_of_range")
	}
	return nil
}
func invalid(state string) error {
	return k4k3ruSDKAppError.Tracef("failed to validate spread parameters: %w: %s", k4k3ruSDKAppError.InvalidParameter(), state)
}
func invalidJSONError(err error) error {
	return k4k3ruSDKAppError.Tracef("failed to decode spread parameters: %w: %w: json=invalid", k4k3ruSDKAppError.InvalidParameter(), err)
}
