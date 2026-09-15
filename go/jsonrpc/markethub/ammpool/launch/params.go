package launch

import (
	"bytes"
	"encoding/json"
	"fmt"
	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"io"
	"regexp"
	"strings"
)

// MaximumAgeSeconds bounds the server's in-memory observation window.
const MaximumAgeSeconds uint32 = 604800

type Params struct {
	Chain              string `json:"chain,omitempty"`
	Network            string `json:"network,omitempty"`
	Venue              string `json:"venue,omitempty"`
	MaxPoolAgeSeconds  uint32 `json:"maxPoolAgeSeconds"`
	MaxTokenAgeSeconds uint32 `json:"maxTokenAgeSeconds,omitempty"`
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
var poolID = regexp.MustCompile(`^0x(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// Normalize normalizes launch selectors.
//
// Version:
//   - 2026-09-15: Added.
func (p Params) Normalize() Params {
	p.Chain = strings.ToLower(strings.TrimSpace(p.Chain))
	p.Network = strings.ToLower(strings.TrimSpace(p.Network))
	p.Venue = strings.ToLower(strings.TrimSpace(p.Venue))
	return p
}

// Validate validates bounded pool and token age filters.
//
// Version:
//   - 2026-09-15: Use maxPoolAgeSeconds for the pool age selector.
func (p Params) Validate() error {
	p = p.Normalize()
	for _, value := range []string{p.Chain, p.Network, p.Venue} {
		if value != "" && !selector.MatchString(value) {
			return fmt.Errorf("failed to validate launch parameters: %w: selector=invalid", app.InvalidParameter())
		}
	}
	if p.Venue != "" && p.Venue != "uniswap-v3" && p.Venue != "uniswap-v4" && p.Venue != "aerodrome" {
		return fmt.Errorf("failed to validate launch parameters: %w: venue=invalid", app.InvalidParameter())
	}
	if p.MaxPoolAgeSeconds == 0 || p.MaxPoolAgeSeconds > MaximumAgeSeconds || p.MaxTokenAgeSeconds > MaximumAgeSeconds {
		return fmt.Errorf("failed to validate launch parameters: %w: age=out_of_range max_value=%d", app.InvalidParameter(), MaximumAgeSeconds)
	}
	return nil
}

// SubscriptionKey identifies the complete normalized launch filter.
//
// Version:
//   - 2026-09-15: Use maxPoolAgeSeconds for the pool age selector.
func (p Params) SubscriptionKey() (string, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return "", fmt.Errorf("failed to create launch subscription key: %w", err)
	}
	return fmt.Sprintf("MarketHub.AMMPool.Launch:c=%s:n=%s:v=%s:pool_age=%d:token_age=%d", p.Chain, p.Network, p.Venue, p.MaxPoolAgeSeconds, p.MaxTokenAgeSeconds), nil
}

// Normalize normalizes the pool's owning chain, venue and identifier.
//
// Version:
//   - 2026-09-15: Added.
func (p GetParams) Normalize() GetParams {
	p.Chain = strings.ToLower(strings.TrimSpace(p.Chain))
	p.Network = strings.ToLower(strings.TrimSpace(p.Network))
	p.Venue = strings.ToLower(strings.TrimSpace(p.Venue))
	p.PoolID = strings.ToLower(strings.TrimSpace(p.PoolID))
	return p
}

// Validate requires an unambiguous EVM pool identity.
//
// Version:
//   - 2026-09-15: Use maxPoolAgeSeconds for the pool age selector.
func (p GetParams) Validate() error {
	p = p.Normalize()
	if p.Chain == "" || p.Network == "" || p.Venue == "" {
		return fmt.Errorf("failed to validate launch identity: %w: selector=empty", app.InvalidParameter())
	}
	if err := (Params{Chain: p.Chain, Network: p.Network, Venue: p.Venue, MaxPoolAgeSeconds: 1}).Validate(); err != nil {
		return fmt.Errorf("failed to validate launch identity: %w", err)
	}
	if !poolID.MatchString(p.PoolID) || (p.Venue == "uniswap-v4" && len(p.PoolID) != 66) || (p.Venue != "uniswap-v4" && len(p.PoolID) != 42) {
		return fmt.Errorf("failed to validate launch identity: %w: pool_id=invalid", app.InvalidParameter())
	}
	return nil
}

// Validate validates pagination and the launch filter.
//
// Version:
//   - 2026-09-15: Use maxPoolAgeSeconds for the pool age selector.
func (p ListParams) Validate() error {
	if err := p.Filter.Validate(); err != nil {
		return fmt.Errorf("failed to validate launch list: %w", err)
	}
	if p.Limit > 200 || len(p.Cursor) > 2048 {
		return fmt.Errorf("failed to validate launch list: %w: pagination=out_of_range", app.InvalidParameter())
	}
	return nil
}
func decode(data []byte, v any) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return fmt.Errorf("failed to decode launch parameters: %w: params=null", app.InvalidParameter())
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return fmt.Errorf("failed to decode launch parameters: %w: %w", app.InvalidParameter(), err)
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return fmt.Errorf("failed to decode launch parameters: %w: json=invalid", app.InvalidParameter())
	}
	return nil
}

// UnmarshalJSON rejects unknown fields and invalid filters.
//
// Version:
//   - 2026-09-15: Use maxPoolAgeSeconds for the pool age selector.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode launch parameters: destination=null")
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
//   - 2026-09-15: Use maxPoolAgeSeconds for the pool age selector.
func (p *GetParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode launch identity: destination=null")
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
//   - 2026-09-15: Use maxPoolAgeSeconds for the pool age selector.
func (p *ListParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode launch list: destination=null")
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
//   - 2026-09-15: Added.
func (p ListParams) Normalize() ListParams {
	p.Filter = p.Filter.Normalize()
	if p.Limit == 0 {
		p.Limit = 100
	}
	return p
}
