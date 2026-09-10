package ammpool

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
	Symbol        string `json:"symbol"`
	MaxAgeSeconds uint32 `json:"maxAgeSeconds"`
}

var symbolPattern = regexp.MustCompile(`^[A-Z0-9._-]+/[A-Z0-9._-]+$`)

// Normalize canonicalizes the symbol without changing the requested age.
//
// Version:
//   - 2026-09-10: Added.
func (p Params) Normalize() Params {
	p.Symbol = strings.ToUpper(strings.TrimSpace(p.Symbol))
	return p
}

// Validate requires a pair symbol and a positive maximum age in seconds.
//
// Version:
//   - 2026-09-10: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if p.Symbol == "" {
		return fmt.Errorf("failed to validate amm pool parameters: %w: symbol=empty", app.InvalidParameter())
	}
	if len(p.Symbol) > 16 {
		return fmt.Errorf("failed to validate amm pool parameters: %w: symbol=too_long max_length=16", app.InvalidParameter())
	}
	if !symbolPattern.MatchString(p.Symbol) {
		return fmt.Errorf("failed to validate amm pool parameters: %w: symbol=invalid", app.InvalidParameter())
	}
	if p.MaxAgeSeconds == 0 {
		return fmt.Errorf("failed to validate amm pool parameters: %w: max_age_seconds=empty", app.InvalidParameter())
	}
	return nil
}

// SubscriptionKey identifies one symbol and freshness policy.
//
// Version:
//   - 2026-09-10: Added.
func (p Params) SubscriptionKey() (string, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return "", fmt.Errorf("failed to build amm pool subscription key: %w", err)
	}
	return fmt.Sprintf("MarketHub.AMMPool:s=%s:age=%d", p.Symbol, p.MaxAgeSeconds), nil
}

// UnmarshalJSON rejects unknown fields and malformed parameter objects.
//
// Version:
//   - 2026-09-10: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode amm pool parameters: %w: destination=null", app.InvalidParameter())
	}
	type wire Params
	var v wire
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		return fmt.Errorf("failed to decode amm pool parameters: %w: %w", app.InvalidParameter(), err)
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("failed to decode amm pool parameters: %w: json=invalid", app.InvalidParameter())
	}
	*p = Params(v)
	return p.Validate()
}
