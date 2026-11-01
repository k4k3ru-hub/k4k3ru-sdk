package overview

import (
	"encoding/json"
	"fmt"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/asset"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/funding"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/markets"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/openinterest"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/change"
	pricerange "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/range"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/volume"
	onchain "github.com/k4k3ru-hub/onchain/go/core"
	"reflect"
	"time"
)

type State string

const (
	StateReady         State = "ready"
	StateStale         State = "stale"
	StateUnavailable   State = "unavailable"
	StateNotApplicable State = "not_applicable"
)

type Section[T any] struct {
	State        State   `json:"state"`
	Reason       *string `json:"reason"`
	CalculatedAt *int64  `json:"calculatedAt"`
	ExpiresAt    *int64  `json:"expiresAt"`
	Data         *T      `json:"data"`
}
type PointTimes struct {
	ObservedAt *int64 `json:"observedAt"`
	FreshUntil *int64 `json:"freshUntil"`
	ExpiresAt  *int64 `json:"expiresAt"`
}
type Price struct {
	Definition string       `json:"definition"`
	Currency   string       `json:"currency"`
	Method     string       `json:"method"`
	Interval   string       `json:"interval"`
	Price      *string      `json:"price"`
	Status     price.Status `json:"status"`
	VenueCount uint32       `json:"venueCount"`
	PointTimes
}
type PriceChanges struct {
	Definition string          `json:"definition"`
	Currency   string          `json:"currency"`
	Method     string          `json:"method"`
	Price      *string         `json:"price"`
	Changes    []change.Change `json:"changes"`
	PointTimes
}
type PriceRanges struct {
	Definition string             `json:"definition"`
	Currency   string             `json:"currency"`
	Method     string             `json:"method"`
	Ranges     []pricerange.Range `json:"ranges"`
}
type Volume struct {
	Definition string                `json:"definition"`
	Currency   string                `json:"currency"`
	Method     string                `json:"method"`
	Volumes    []volume.PeriodVolume `json:"volumes"`
}
type MarketPrice struct {
	Currency string       `json:"currency"`
	Method   string       `json:"method"`
	Price    *string      `json:"price"`
	Status   price.Status `json:"status"`
	PointTimes
}
type Market struct {
	markets.Market
	Price MarketPrice `json:"price"`
}
type Markets struct {
	Total    int      `json:"total"`
	Eligible int      `json:"eligible"`
	Items    []Market `json:"items"`
}
type Funding struct {
	funding.Rate
	Symbol market.Symbol `json:"symbol"`
	PointTimes
}
type OpenInterest struct {
	openinterest.Interest
	Symbol market.Symbol `json:"symbol"`
	PointTimes
}
type Snapshot struct {
	ID          string `json:"id"`
	GeneratedAt int64  `json:"generatedAt"`
}
type Group struct {
	MarketType   market.MarketType       `json:"marketType"`
	Price        Section[Price]          `json:"price"`
	PriceChanges Section[PriceChanges]   `json:"priceChanges"`
	PriceRanges  Section[PriceRanges]    `json:"priceRanges"`
	Volume       Section[Volume]         `json:"volume"`
	Markets      Section[Markets]        `json:"markets"`
	Funding      Section[[]Funding]      `json:"funding"`
	OpenInterest Section[[]OpenInterest] `json:"openInterest"`
}
type Result struct {
	AssetType   market.AssetClass   `json:"assetType"`
	Asset       string              `json:"asset"`
	Symbol      string              `json:"symbol"`
	Name        string              `json:"name"`
	MarketTypes []market.MarketType `json:"marketTypes"`
	Snapshot    Snapshot            `json:"snapshot"`
	Groups      []Group             `json:"groups"`
	SentAt      int64               `json:"sentAt"`
}

// DecodeResult decodes a bounded response, rejecting unknown and duplicate fields.
//
// Version:
//   - 2026-11-01: Added.
func DecodeResult(data []byte) (*Result, error) {
	if len(data) > MaximumResponseBytes {
		return nil, invalid("response=too_long")
	}
	type wire Result
	var v wire
	if err := jsonobject.Decode(data, &v, "assetType", "asset", "symbol", "name", "marketTypes", "snapshot", "groups", "sentAt"); err != nil {
		return nil, fmt.Errorf("failed to decode overview response: %w", err)
	}
	r := Result(v)
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

// ValidateFor validates the complete response and binds it to one requested asset.
//
// Version:
//   - 2026-11-01: Added.
func (r Result) ValidateFor(p Params) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("failed to validate requested overview: %w", err)
	}
	if err := r.Validate(); err != nil {
		return err
	}
	if r.AssetType != p.AssetType || r.Asset != p.Asset {
		return invalid("response_scope=invalid")
	}
	return nil
}

// Validate validates identity, availability, original calculation times and metric contracts.
// Financial calculations remain the responsibility of the owning server domains.
//
// Version:
//   - 2026-11-01: Preserve unavailable registered CEX metadata without adding CEX prices.
func (r Result) Validate() error {
	if err := (asset.Result{AssetType: r.AssetType, Asset: r.Asset, Symbol: r.Symbol, Name: r.Name, MarketTypes: r.MarketTypes, SentAt: r.SentAt}).Validate(); err != nil {
		return fmt.Errorf("failed to validate overview response: %w", err)
	}
	if r.Snapshot.ID == "" || len(r.Snapshot.ID) > 128 || r.Snapshot.GeneratedAt <= 0 || r.Snapshot.GeneratedAt > r.SentAt || len(r.Groups) != len(r.MarketTypes) {
		return invalid("snapshot_or_groups=invalid")
	}
	for i, g := range r.Groups {
		if g.MarketType != r.MarketTypes[i] {
			return invalid("group_scope=invalid")
		}
		for _, err := range []error{g.Price.Validate(r.Snapshot.GeneratedAt), g.PriceChanges.Validate(r.Snapshot.GeneratedAt), g.PriceRanges.Validate(r.Snapshot.GeneratedAt), g.Volume.Validate(r.Snapshot.GeneratedAt), g.Markets.Validate(r.Snapshot.GeneratedAt), g.Funding.Validate(r.Snapshot.GeneratedAt), g.OpenInterest.Validate(r.Snapshot.GeneratedAt)} {
			if err != nil {
				return err
			}
		}
		if g.MarketType == market.MarketTypeSpot {
			if g.Funding.State != StateNotApplicable || g.OpenInterest.State != StateNotApplicable {
				return invalid("spot_derivatives=invalid")
			}
		} else if g.Funding.State == StateNotApplicable || g.OpenInterest.State == StateNotApplicable {
			return invalid("perpetual_derivatives=invalid")
		}
		if g.Price.State == StateNotApplicable || g.PriceChanges.State == StateNotApplicable || g.PriceRanges.State == StateNotApplicable || g.Volume.State == StateNotApplicable || g.Markets.State == StateNotApplicable {
			return invalid("section_applicability=invalid")
		}
		if v := g.Price.Data; v != nil {
			if err := (price.Result{Asset: r.Asset, MarketType: g.MarketType, Definition: v.Definition, Currency: v.Currency, Method: v.Method, Interval: v.Interval, Price: v.Price, Status: v.Status, VenueCount: v.VenueCount, SentAt: *g.Price.CalculatedAt}).Validate(); err != nil {
				return fmt.Errorf("failed to validate overview price: %w", err)
			}
			if err := validatePoint(v.PointTimes, v.Price != nil, *g.Price.CalculatedAt); err != nil {
				return err
			}
		}
		if v := g.PriceChanges.Data; v != nil {
			if err := (change.Result{Asset: r.Asset, MarketType: g.MarketType, Definition: v.Definition, Currency: v.Currency, Method: v.Method, Price: v.Price, Changes: v.Changes, SentAt: *g.PriceChanges.CalculatedAt}).Validate(); err != nil {
				return fmt.Errorf("failed to validate overview changes: %w", err)
			}
			if len(v.Changes) != 6 {
				return invalid("change_periods=invalid")
			}
			for j, c := range v.Changes {
				if string(c.Period) != periods[j] {
					return invalid("change_order=invalid")
				}
			}
			if err := validatePoint(v.PointTimes, v.Price != nil, *g.PriceChanges.CalculatedAt); err != nil {
				return err
			}
			if p := g.Price.Data; p != nil && (!reflect.DeepEqual(p.Price, v.Price) || !reflect.DeepEqual(p.PointTimes, v.PointTimes) || !reflect.DeepEqual(g.Price.CalculatedAt, g.PriceChanges.CalculatedAt) || !reflect.DeepEqual(g.Price.ExpiresAt, g.PriceChanges.ExpiresAt)) {
				return invalid("price_pair=invalid")
			}
		}
		if v := g.PriceRanges.Data; v != nil {
			if err := (pricerange.Result{Asset: r.Asset, MarketType: g.MarketType, Definition: v.Definition, Currency: v.Currency, Method: v.Method, Ranges: v.Ranges, SentAt: *g.PriceRanges.CalculatedAt}).Validate(); err != nil {
				return fmt.Errorf("failed to validate overview ranges: %w", err)
			}
			if len(v.Ranges) != 6 {
				return invalid("range_periods=invalid")
			}
			for j, c := range v.Ranges {
				if string(c.Period) != periods[j] || c.To != *g.PriceRanges.CalculatedAt {
					return invalid("range_order_or_time=invalid")
				}
			}
		}
		if v := g.Volume.Data; v != nil {
			if err := (volume.Result{Asset: r.Asset, MarketType: g.MarketType, Definition: v.Definition, Currency: v.Currency, Method: v.Method, Volumes: v.Volumes, SentAt: *g.Volume.CalculatedAt}).Validate(); err != nil {
				return fmt.Errorf("failed to validate overview volume: %w", err)
			}
			if len(v.Volumes) != 1 || string(v.Volumes[0].Period) != "24h" || v.Volumes[0].To != *g.Volume.CalculatedAt {
				return invalid("volume_period_or_time=invalid")
			}
		}
		if v := g.Markets.Data; v != nil {
			if v.Total != len(v.Items) || v.Total > MaximumMarkets || v.Eligible < 0 || v.Eligible > v.Total || v.Items == nil {
				return invalid("market_counts=invalid")
			}
			eligible := 0
			for j, m := range v.Items {
				// The existing market validator accepts at most 500 rows per page; validate each
				// identity independently and enforce collection ordering here.
				if err := (markets.Result{Asset: r.Asset, MarketType: g.MarketType, Page: 1, Limit: 1, Total: 1, Markets: []markets.Market{m.Market}, SentAt: *g.Markets.CalculatedAt}).Validate(); err != nil {
					return fmt.Errorf("failed to validate overview market: %w", err)
				}
				if j > 0 && !referenceBefore(v.Items[j-1].Market.Market, m.Market.Market) {
					return invalid("market_order=invalid")
				}
				if m.CompositeEligible {
					eligible++
				}
				if m.Price.Price == nil && m.Price.Status == price.StatusUnavailable {
					// Registered CEX identities remain metadata rows without enabling
					// unsupported individual prices in the existing Price.Get API.
					method := "last_trade"
					if m.Market.Market.PoolID != "" {
						method = "pool_price"
					}
					if m.Price.Currency != "USD" || m.Price.Method != method {
						return invalid("market_price_metadata=invalid")
					}
				} else if err := (price.Result{Asset: r.Asset, MarketType: g.MarketType, Definition: "composite", Currency: "USD", Method: "volume_weighted_reference", Interval: "60s", Status: price.StatusUnavailable, Markets: []price.MarketPrice{{Market: m.Market.Market, Symbol: m.Symbol, Currency: m.Price.Currency, Method: m.Price.Method, Price: m.Price.Price, Status: m.Price.Status}}, SentAt: *g.Markets.CalculatedAt}).Validate(); err != nil {
					return fmt.Errorf("failed to validate overview market price: %w", err)
				}
				if err := validatePoint(m.Price.PointTimes, m.Price.Price != nil, *g.Markets.CalculatedAt); err != nil {
					return err
				}
			}
			if eligible != v.Eligible {
				return invalid("eligible_count=invalid")
			}
		}
		if v := g.Funding.Data; v != nil {
			if len(*v) > MaximumMarkets || *v == nil {
				return invalid("funding_rows=invalid")
			}
			for j, x := range *v {
				if x.Market.Network != onchain.NetworkMainnet {
					return invalid("derivative_network=invalid")
				}
				if g.Markets.Data != nil {
					found := false
					for _, m := range g.Markets.Data.Items {
						if m.Market.Market == x.Market && m.Symbol == x.Symbol {
							found = true
						}
					}
					if !found {
						return invalid("derivative_scope=invalid")
					}
				}
				if err := (funding.Result{Symbol: x.Symbol, Rates: []funding.Rate{x.Rate}, SentAt: *g.Funding.CalculatedAt}).Validate(); err != nil {
					return fmt.Errorf("failed to validate overview funding: %w", err)
				}
				if j > 0 && !referenceBefore((*v)[j-1].Market, x.Market) {
					return invalid("funding_order=invalid")
				}
				if err := validatePoint(x.PointTimes, x.Rate.Rate != nil, *g.Funding.CalculatedAt); err != nil {
					return err
				}
			}
		}
		if v := g.OpenInterest.Data; v != nil {
			if len(*v) > MaximumMarkets || *v == nil {
				return invalid("open_interest_rows=invalid")
			}
			for j, x := range *v {
				if x.Market.Network != onchain.NetworkMainnet {
					return invalid("derivative_network=invalid")
				}
				if g.Markets.Data != nil {
					found := false
					for _, m := range g.Markets.Data.Items {
						if m.Market.Market == x.Market && m.Symbol == x.Symbol {
							found = true
						}
					}
					if !found {
						return invalid("derivative_scope=invalid")
					}
				}
				if err := (openinterest.Result{Symbol: x.Symbol, Interests: []openinterest.Interest{x.Interest}, SentAt: *g.OpenInterest.CalculatedAt}).Validate(); err != nil {
					return fmt.Errorf("failed to validate overview open interest: %w", err)
				}
				if j > 0 && !referenceBefore((*v)[j-1].Market, x.Market) {
					return invalid("open_interest_order=invalid")
				}
				if err := validatePoint(x.PointTimes, x.Quantity != nil, *g.OpenInterest.CalculatedAt); err != nil {
					return err
				}
			}
		}
	}
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("failed to encode overview response: %w", err)
	}
	if len(b) > MaximumResponseBytes {
		return invalid("response=too_long")
	}
	return nil
}

var periods = []string{"5m", "15m", "1h", "4h", "24h", "7d"}

func invalid(reason string) error {
	return fmt.Errorf("failed to validate overview response: %s", reason)
}

// Validate checks retained computation metadata against its publication time.
//
// Version:
//   - 2026-11-01: Added.
func (s Section[T]) Validate(generated int64) error {
	if s.CalculatedAt != nil && (*s.CalculatedAt <= 0 || *s.CalculatedAt > generated) {
		return invalid("calculated_at=invalid")
	}
	if s.ExpiresAt != nil && (s.CalculatedAt == nil || *s.ExpiresAt <= *s.CalculatedAt || *s.ExpiresAt > *s.CalculatedAt+int64(24*time.Hour/time.Microsecond)) {
		return invalid("expires_at=invalid")
	}
	switch s.State {
	case StateReady, StateStale:
		if s.Data == nil || s.CalculatedAt == nil || s.ExpiresAt == nil {
			return invalid("available_section=invalid")
		}
		if s.State == StateReady && s.Reason != nil {
			return invalid("ready_reason=invalid")
		}
		if s.State == StateStale && (s.Reason == nil || (*s.Reason != "refresh_delayed" && *s.Reason != "refresh_failed")) {
			return invalid("stale_reason=invalid")
		}
	case StateUnavailable:
		if s.Data != nil || s.Reason == nil {
			return invalid("unavailable_section=invalid")
		}
		switch *s.Reason {
		case "not_ready", "no_observations", "expired", "invalidated":
		default:
			return invalid("unavailable_reason=invalid")
		}
	case StateNotApplicable:
		if s.Data != nil || s.Reason != nil || s.CalculatedAt != nil || s.ExpiresAt != nil {
			return invalid("not_applicable_section=invalid")
		}
	default:
		return invalid("section_state=invalid")
	}
	return nil
}
func validatePoint(p PointTimes, available bool, calculated int64) error {
	if !available {
		if p.ObservedAt != nil || p.FreshUntil != nil || p.ExpiresAt != nil {
			return invalid("unavailable_point_times=invalid")
		}
		return nil
	}
	if p.ObservedAt == nil || p.FreshUntil == nil || p.ExpiresAt == nil || *p.ObservedAt <= 0 || *p.ObservedAt > calculated || *p.FreshUntil <= 0 || *p.ExpiresAt <= *p.ObservedAt || *p.ExpiresAt > *p.ObservedAt+int64(24*time.Hour/time.Microsecond) {
		return invalid("point_times=invalid")
	}
	return nil
}
func referenceBefore(a, b market.MarketRef) bool {
	l := []string{string(a.Venue), string(a.Network), string(a.Chain), a.PoolID, a.VenueSymbol}
	r := []string{string(b.Venue), string(b.Network), string(b.Chain), b.PoolID, b.VenueSymbol}
	for i := range l {
		if l[i] != r[i] {
			return l[i] < r[i]
		}
	}
	return false
}

// UnmarshalJSON decodes the strict response shape atomically before Gateway timestamp validation.
//
// Version:
//   - 2026-11-01: Added.
func (r *Result) UnmarshalJSON(data []byte) error {
	if r == nil {
		return invalid("destination=null")
	}
	if len(data) > MaximumResponseBytes {
		return invalid("response=too_long")
	}
	type wire Result
	var value wire
	if err := jsonobject.Decode(data, &value, "assetType", "asset", "symbol", "name", "marketTypes", "snapshot", "groups", "sentAt"); err != nil {
		return fmt.Errorf("failed to decode overview response: %w", err)
	}
	*r = Result(value)
	return nil
}
