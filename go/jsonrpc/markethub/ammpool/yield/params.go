package yield

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
var suiPoolID = regexp.MustCompile(`^[0-9a-f]{1,64}$`)

// Normalize normalizes yield selectors.
//
// Version:
//   - 2026-09-19: Added.
//   - 2026-09-19: Use server lifecycle retention with scope-only request filters.
func (p Params) Normalize() Params {
	p.Chain = strings.ToLower(strings.TrimSpace(p.Chain))
	p.Network = strings.ToLower(strings.TrimSpace(p.Network))
	p.Venue = strings.ToLower(strings.TrimSpace(p.Venue))
	return p
}

// Validate validates chain, network and venue filters.
//
// Version:
//   - 2026-09-19: Limit filters to chain, network and venue.
func (p Params) Validate() error {
	p = p.Normalize()
	for _, value := range []string{p.Chain, p.Network, p.Venue} {
		if len(value) > 64 {
			return fmt.Errorf("failed to validate yield parameters: %w: selector=too_long actual_length=%d max_length=64", app.InvalidParameter(), len(value))
		}
		if value != "" && !selector.MatchString(value) {
			return fmt.Errorf("failed to validate yield parameters: %w: selector=invalid", app.InvalidParameter())
		}
	}
	return nil
}

// SubscriptionKey identifies the complete normalized yield filter.
//
// Version:
//   - 2026-09-19: Limit filters to chain, network and venue.
func (p Params) SubscriptionKey() (string, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return "", fmt.Errorf("failed to create yield subscription key: %w", err)
	}
	return fmt.Sprintf("MarketHub.AMMPool.Yield:c=%s:n=%s:v=%s", p.Chain, p.Network, p.Venue), nil
}

// Normalize normalizes the pool's owning chain, venue and identifier.
//
// Version:
//   - 2026-09-19: Added.
func (p GetParams) Normalize() GetParams {
	p.Chain = strings.ToLower(strings.TrimSpace(p.Chain))
	p.Network = strings.ToLower(strings.TrimSpace(p.Network))
	p.Venue = strings.ToLower(strings.TrimSpace(p.Venue))
	p.PoolID = strings.TrimSpace(p.PoolID)
	if p.Chain == "sui" {
		hex := strings.TrimPrefix(strings.ToLower(p.PoolID), "0x")
		if suiPoolID.MatchString(hex) {
			p.PoolID = "0x" + strings.Repeat("0", 64-len(hex)) + hex
		}
	}
	if strings.HasPrefix(p.PoolID, "0x") {
		p.PoolID = strings.ToLower(p.PoolID)
	}
	return p
}

// Validate requires an unambiguous chain-neutral pool identity.
//
// Version:
//   - 2026-09-19: Limit filters to chain, network and venue.
func (p GetParams) Validate() error {
	p = p.Normalize()
	if p.Chain == "" || p.Network == "" || p.Venue == "" {
		return fmt.Errorf("failed to validate yield identity: %w: selector=empty", app.InvalidParameter())
	}
	if err := (Params{Chain: p.Chain, Network: p.Network, Venue: p.Venue}).Validate(); err != nil {
		return fmt.Errorf("failed to validate yield identity: %w", err)
	}
	if p.PoolID == "" {
		return fmt.Errorf("failed to validate yield identity: %w: pool_id=empty", app.InvalidParameter())
	}
	if len(p.PoolID) > 128 {
		return fmt.Errorf("failed to validate yield identity: %w: pool_id=too_long actual_length=%d max_length=128", app.InvalidParameter(), len(p.PoolID))
	}
	if !poolID.MatchString(p.PoolID) {
		return fmt.Errorf("failed to validate yield identity: %w: pool_id=invalid", app.InvalidParameter())
	}
	if p.Chain == "sui" {
		hex := strings.TrimPrefix(p.PoolID, "0x")
		if !suiPoolID.MatchString(hex) || strings.Trim(hex, "0") == "" {
			return fmt.Errorf("failed to validate yield identity: %w: pool_id=invalid", app.InvalidParameter())
		}
	}
	return nil
}

// Validate validates pagination and the yield filter.
//
// Version:
//   - 2026-09-19: Limit filters to chain, network and venue.
func (p ListParams) Validate() error {
	if err := p.Filter.Validate(); err != nil {
		return fmt.Errorf("failed to validate yield list: %w", err)
	}
	if p.Limit > 200 {
		return fmt.Errorf("failed to validate yield list: %w: pagination=out_of_range", app.InvalidParameter())
	}
	if len(p.Cursor) > 2048 {
		return fmt.Errorf("failed to validate yield list: %w: cursor=too_long actual_length=%d max_length=2048", app.InvalidParameter(), len(p.Cursor))
	}
	return nil
}
func decode(data []byte, v any) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("failed to decode yield parameters: %w: params=null", app.InvalidParameter())
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return fmt.Errorf("failed to decode yield parameters: %w: %w", app.InvalidParameter(), err)
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return fmt.Errorf("failed to decode yield parameters: %w: json=invalid", app.InvalidParameter())
	}
	return nil
}

// UnmarshalJSON rejects unknown fields and invalid filters.
//
// Version:
//   - 2026-09-19: Limit filters to chain, network and venue.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode yield parameters: destination=null")
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
//   - 2026-09-19: Limit filters to chain, network and venue.
func (p *GetParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode yield identity: destination=null")
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
//   - 2026-09-19: Limit filters to chain, network and venue.
func (p *ListParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode yield list: destination=null")
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
//   - 2026-09-19: Added.
func (p ListParams) Normalize() ListParams {
	p.Filter = p.Filter.Normalize()
	if p.Limit == 0 {
		p.Limit = 100
	}
	return p
}
