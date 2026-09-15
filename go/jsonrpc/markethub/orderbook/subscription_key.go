package orderbook

import (
	sdkError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"strconv"
	"strings"
)

// ParseSubscriptionKey restores normalized parameters from a canonical subscription key.
//
// Returns:
//   - Normalized parameters, or an invalid-parameter error for a noncanonical key.
//
// Version:
//   - 2026-09-16: Added.
func ParseSubscriptionKey(key string) (Params, error) {
	invalid := func() (Params, error) {
		return Params{}, sdkError.Tracef("failed to parse orderbook subscription key: %w: subscription_key=invalid", sdkError.InvalidParameter())
	}
	body, ok := strings.CutPrefix(key, "MarketHub.OrderBook:ac=")
	if !ok {
		return invalid()
	}
	asset, body, ok := strings.Cut(body, ":mt=")
	if !ok {
		return invalid()
	}
	market, body, ok := strings.Cut(body, ":s=")
	if !ok {
		return invalid()
	}
	sourceAt := strings.LastIndex(body, ":src=")
	if sourceAt < 0 {
		return invalid()
	}
	source := body[sourceAt+5:]
	body = body[:sourceAt]
	p := Params{AssetClass: AssetClass(asset), MarketType: MarketType(market)}

	depthAt := strings.LastIndex(body, ":d=")
	if depthAt < 0 {
		return invalid()
	}
	depth, err := strconv.ParseUint(body[depthAt+3:], 10, 16)
	if err != nil {
		return invalid()
	}
	p.Depth = uint16(depth)
	body = body[:depthAt]

	p.Symbol = Symbol(body)
	if source != "*" {
		filter := &SourceFilter{}
		for _, part := range strings.Split(source, "|") {
			name, value, ok := strings.Cut(part, "=")
			if !ok {
				return invalid()
			}
			values := strings.Split(value, ",")
			switch name {
			case "VC":
				if filter.VenueCategories != nil {
					return invalid()
				}
				for _, v := range values {
					filter.VenueCategories = append(filter.VenueCategories, VenueCategory(v))
				}
			case "LM":
				if filter.LiquidityModels != nil {
					return invalid()
				}
				for _, v := range values {
					filter.LiquidityModels = append(filter.LiquidityModels, LiquidityModel(v))
				}
			case "APC":
				if filter.AMMPoolChains != nil {
					return invalid()
				}
				for _, v := range values {
					filter.AMMPoolChains = append(filter.AMMPoolChains, Chain(v))
				}
			default:
				return invalid()
			}
		}
		p.SourceFilter = filter
	}
	p = p.Normalize()
	generated, err := p.SubscriptionKey()
	if err != nil {
		return Params{}, sdkError.Tracef("failed to parse orderbook subscription key: %w", err)
	}
	if generated != key {
		return invalid()
	}
	return p, nil
}
