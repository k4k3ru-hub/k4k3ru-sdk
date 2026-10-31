// Package news defines retained, source-attributed Asset news metadata.
package news

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/asset"
)

type Params struct {
	AssetType market.AssetClass `json:"assetType"`
	Asset     string            `json:"asset"`
	From      int64             `json:"from,omitempty"` // Inclusive source publication time, Unix microseconds; omitted means no lower bound.
	To        int64             `json:"to,omitempty"`   // Exclusive source publication time; omitted means no upper bound.
	Page      int               `json:"page,omitempty"`
	Limit     int               `json:"limit,omitempty"`
	Snapshot  string            `json:"snapshot,omitempty"` // Required after page one; returned by the first response.
}

// Normalize fills omitted pagination without changing the caller's request.
//
// Version:
//   - 2026-10-31: Added.
func (p Params) Normalize() Params {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.Limit == 0 {
		p.Limit = 20
	}
	return p
}

// Validate validates an Asset-scoped Mainnet-only retained news read.
//
// Version:
//   - 2026-10-31: Added.
func (p Params) Validate() error {
	if err := (asset.Params{AssetType: p.AssetType, Asset: p.Asset}).Validate(); err != nil {
		return fmt.Errorf("failed to validate news parameters: %w", err)
	}
	p = p.Normalize()
	if p.Page < 1 || p.Page > 100 || p.Limit < 1 || p.Limit > 100 {
		return invalid("pagination")
	}
	if p.From < 0 || p.To < 0 || (p.To != 0 && p.From >= p.To) {
		return invalid("period")
	}
	if len(p.Snapshot) > 8192 || (p.Page > 1 && p.Snapshot == "") {
		return invalid("snapshot")
	}
	return nil
}

// UnmarshalJSON rejects unknown, duplicate and null selectors before applying defaults.
//
// Version:
//   - 2026-10-31: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination")
	}
	type wire Params
	value := wire{Page: 1, Limit: 20}
	if err := jsonobject.Decode(data, &value, "assetType", "asset"); err != nil {
		return fmt.Errorf("failed to decode news parameters: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to decode news parameters: %w", err)
	}
	for _, raw := range fields {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return invalid("selector")
		}
	}
	if value.Page == 0 || value.Limit == 0 {
		return invalid("pagination")
	}
	if err := Params(value).Validate(); err != nil {
		return err
	}
	*p = Params(value)
	return nil
}

func invalid(field string) error {
	return fmt.Errorf("failed to validate news: %w: %s=invalid", apperror.InvalidParameter(), field)
}
