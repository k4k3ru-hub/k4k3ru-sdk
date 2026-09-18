package newpair

import (
	"bytes"
	"encoding/json"
	"fmt"
	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"io"
	"regexp"
	"strings"
)

type Params struct {
	Chain   string `json:"chain,omitempty"`
	Network string `json:"network,omitempty"`
	Venue   string `json:"venue,omitempty"`
}
type GetParams struct {
	Chain   string `json:"chain"`
	Network string `json:"network"`
	Venue   string `json:"venue"`
	PoolID  string `json:"poolId"`
}
type ListParams struct {
	Filter Params `json:"filter"`
	Limit  uint32 `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

var selector = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
var poolID = regexp.MustCompile(`^[A-Za-z0-9:_-]{1,128}$`)

// Normalize normalizes new pair selectors.
//
// Version:
//   - 2026-09-16: Added.
//   - 2026-09-18: Use server lifecycle retention with scope-only request filters.
func (p Params) Normalize() Params {
	p.Chain = strings.ToLower(strings.TrimSpace(p.Chain))
	p.Network = strings.ToLower(strings.TrimSpace(p.Network))
	p.Venue = strings.ToLower(strings.TrimSpace(p.Venue))
	return p
}

// Validate validates chain, network and venue filters.
//
// Version:
//   - 2026-09-18: Limit filters to chain, network and venue.
func (p Params) Validate() error {
	p = p.Normalize()
	for _, value := range []string{p.Chain, p.Network, p.Venue} {
		if value != "" && !selector.MatchString(value) {
			return fmt.Errorf("failed to validate new pair parameters: %w: selector=invalid", app.InvalidParameter())
		}
	}
	return nil
}

// SubscriptionKey identifies the complete normalized new pair filter.
//
// Version:
//   - 2026-09-18: Limit filters to chain, network and venue.
func (p Params) SubscriptionKey() (string, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return "", fmt.Errorf("failed to create new pair subscription key: %w", err)
	}
	return fmt.Sprintf("MarketHub.AMMPool.NewPair:c=%s:n=%s:v=%s", p.Chain, p.Network, p.Venue), nil
}

// Normalize normalizes the pool's owning chain, venue and identifier.
//
// Version:
//   - 2026-09-16: Added.
func (p GetParams) Normalize() GetParams {
	p.Chain = strings.ToLower(strings.TrimSpace(p.Chain))
	p.Network = strings.ToLower(strings.TrimSpace(p.Network))
	p.Venue = strings.ToLower(strings.TrimSpace(p.Venue))
	p.PoolID = strings.TrimSpace(p.PoolID)
	if strings.HasPrefix(p.PoolID, "0x") {
		p.PoolID = strings.ToLower(p.PoolID)
	}
	return p
}

// Validate requires an unambiguous chain-neutral pool identity.
//
// Version:
//   - 2026-09-18: Limit filters to chain, network and venue.
func (p GetParams) Validate() error {
	p = p.Normalize()
	if p.Chain == "" || p.Network == "" || p.Venue == "" {
		return fmt.Errorf("failed to validate new pair identity: %w: selector=empty", app.InvalidParameter())
	}
	if err := (Params{Chain: p.Chain, Network: p.Network, Venue: p.Venue}).Validate(); err != nil {
		return fmt.Errorf("failed to validate new pair identity: %w", err)
	}
	if !poolID.MatchString(p.PoolID) {
		return fmt.Errorf("failed to validate new pair identity: %w: pool_id=invalid", app.InvalidParameter())
	}
	return nil
}

// Validate validates pagination and the new pair filter.
//
// Version:
//   - 2026-09-18: Limit filters to chain, network and venue.
func (p ListParams) Validate() error {
	if err := p.Filter.Validate(); err != nil {
		return fmt.Errorf("failed to validate new pair list: %w", err)
	}
	if p.Limit > 200 || len(p.Cursor) > 2048 {
		return fmt.Errorf("failed to validate new pair list: %w: pagination=out_of_range", app.InvalidParameter())
	}
	return nil
}
func decode(data []byte, v any) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("failed to decode new pair parameters: %w: params=null", app.InvalidParameter())
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return fmt.Errorf("failed to decode new pair parameters: %w: %w", app.InvalidParameter(), err)
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return fmt.Errorf("failed to decode new pair parameters: %w: json=invalid", app.InvalidParameter())
	}
	return nil
}

// UnmarshalJSON rejects unknown fields and invalid filters.
//
// Version:
//   - 2026-09-18: Limit filters to chain, network and venue.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode new pair parameters: destination=null")
	}
	type wire Params
	var v wire
	if err := decode(data, &v); err != nil {
		return err
	}
	*p = Params(v)
	return p.Validate()
}

// UnmarshalJSON rejects unknown fields and invalid pool identities.
//
// Version:
//   - 2026-09-18: Limit filters to chain, network and venue.
func (p *GetParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode new pair identity: destination=null")
	}
	type wire GetParams
	var v wire
	if err := decode(data, &v); err != nil {
		return err
	}
	*p = GetParams(v)
	return p.Validate()
}

// UnmarshalJSON rejects unknown fields and invalid pagination.
//
// Version:
//   - 2026-09-18: Limit filters to chain, network and venue.
func (p *ListParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode new pair list: destination=null")
	}
	type wire ListParams
	var v wire
	if err := decode(data, &v); err != nil {
		return err
	}
	*p = ListParams(v)
	return p.Validate()
}

// Normalize normalizes list filters and supplies the default page size.
//
// Version:
//   - 2026-09-16: Added.
func (p ListParams) Normalize() ListParams {
	p.Filter = p.Filter.Normalize()
	if p.Limit == 0 {
		p.Limit = 100
	}
	return p
}
