package carry

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type MarketSelector struct {
	Venue      Venue      `json:"venue"`
	MarketType MarketType `json:"marketType"`
	Chain      Chain      `json:"chain,omitempty"`
	Network    Network    `json:"network,omitempty"`
	PoolID     string     `json:"poolId,omitempty"`
}
type RouteSelector struct {
	Buy  MarketSelector `json:"buy"`
	Sell MarketSelector `json:"sell"`
}
type Params struct {
	HoldingPeriodMinutes uint32        `json:"holdingPeriodMinutes"`
	AssetClass           AssetClass    `json:"assetClass,omitempty"`
	Symbol               Symbol        `json:"symbol"`
	BaseAsset            Asset         `json:"baseAsset"`
	Quantity             string        `json:"quantity"`
	Route                RouteSelector `json:"route"`
}

// Normalize normalizes a directed market pair without changing pool identifiers.
//
// Version:
//   - 2026-09-06: Added.
func (r RouteSelector) Normalize() RouteSelector {
	normalize := func(m MarketSelector) MarketSelector {
		m.Venue = Venue(strings.ToLower(strings.TrimSpace(string(m.Venue))))
		m.MarketType = MarketType(strings.ToLower(strings.TrimSpace(string(m.MarketType))))
		m.Chain = Chain(strings.ToLower(strings.TrimSpace(string(m.Chain))))
		m.Network = Network(strings.ToLower(strings.TrimSpace(string(m.Network))))
		m.PoolID = strings.TrimSpace(m.PoolID)
		if m.Chain != "" && m.Network == "" {
			m.Network = NetworkMainnet
		}
		return m
	}
	r.Buy = normalize(r.Buy)
	r.Sell = normalize(r.Sell)
	return r
}

// Validate validates a fixed Carry route.
//
// Version:
//   - 2026-09-06: Added.
func (r RouteSelector) Validate() error {
	r = r.Normalize()
	for _, m := range []MarketSelector{r.Buy, r.Sell} {
		if m.Venue == "" {
			return invalid("venue=empty")
		}
		if len(m.Venue) > 64 || len(m.PoolID) > 256 {
			return invalid("market_selector=too_long")
		}
		if m.MarketType != MarketTypeSpot && m.MarketType != MarketTypePerp {
			return invalid("market_type=invalid")
		}
		if m.Chain == "" {
			if m.PoolID != "" || m.Network != "" {
				return invalid("market_selector=invalid")
			}
		} else {
			switch m.Chain {
			case ChainEthereum, ChainBase, ChainBNB, ChainSolana, ChainSui:
			default:
				return invalid("chain=invalid")
			}
			if m.Network != NetworkMainnet {
				return invalid("network=invalid")
			}
			if m.PoolID == "" {
				return invalid("pool_id=empty")
			}
		}
	}
	if r.Buy == r.Sell {
		return invalid("route=invalid")
	}
	if r.Buy.MarketType == MarketTypeSpot && r.Sell.MarketType == MarketTypeSpot {
		return invalid("route_family=invalid")
	}
	return nil
}

// UnmarshalJSON rejects search filters and unknown fields in fixed-route requests.
//
// Version:
//   - 2026-09-06: Require a fixed route.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination=null")
	}
	type wire Params
	var v wire
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&v); err != nil {
		return invalidJSONError(err)
	}
	*p = Params(v)
	return nil
}

// Normalize canonicalizes fixed-route evaluation parameters.
//
// Version:
//   - 2026-09-06: Require a fixed route.
func (p Params) Normalize() Params {
	common := (SearchParams{AssetClass: p.AssetClass, Symbol: p.Symbol, BaseAsset: p.BaseAsset, Quantity: p.Quantity, HoldingPeriodMinutes: p.HoldingPeriodMinutes}).Normalize()
	p.AssetClass = common.AssetClass
	p.Symbol = common.Symbol
	p.BaseAsset = common.BaseAsset
	p.Quantity = common.Quantity
	p.Route = p.Route.Normalize()
	return p
}

// Validate validates fixed-route evaluation parameters.
//
// Version:
//   - 2026-09-06: Require a fixed route and reject unsupported directions.
func (p Params) Validate() error {
	p = p.Normalize()
	if err := (SearchParams{AssetClass: p.AssetClass, Symbol: p.Symbol, BaseAsset: p.BaseAsset, Quantity: p.Quantity, HoldingPeriodMinutes: p.HoldingPeriodMinutes}).Validate(); err != nil {
		return err
	}
	return p.Route.Validate()
}

// RouteID identifies the directed market pair independently of quantity and period.
//
// Version:
//   - 2026-09-06: Added.
func (p Params) RouteID() (string, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return "", err
	}
	data, err := json.Marshal(struct {
		AssetClass AssetClass
		Symbol     Symbol
		Route      RouteSelector
	}{p.AssetClass, p.Symbol, p.Route})
	if err != nil {
		return "", fmt.Errorf("failed to encode carry route identity: %w", err)
	}
	sum := sha256.Sum256(data)
	return "route_" + hex.EncodeToString(sum[:16]), nil
}

// SubscriptionKey identifies a fixed-route evaluation including quantity and period.
//
// Version:
//   - 2026-09-06: Separate evaluation identity from market pair identity.
func (p Params) SubscriptionKey() (string, error) {
	p = p.Normalize()
	id, err := p.RouteID()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("MarketHub.Carry:route=%s:q=%s:hpm=%d", id, p.Quantity, p.HoldingPeriodMinutes), nil
}
