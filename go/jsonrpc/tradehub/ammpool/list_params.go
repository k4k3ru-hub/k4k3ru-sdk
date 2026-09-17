package ammpool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

const maxListFilterLength = 64

var listFilterPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

type ListParams struct {
	Chain   string `json:"chain,omitempty"`
	Network string `json:"network,omitempty"`
	Venue   string `json:"venue,omitempty"`
}

// Normalize canonicalizes TradeHub AMM pool list filters.
//
// Returns:
//   - Normalized list parameters.
//
// Version:
//   - 2026-09-17: Added.
func (p ListParams) Normalize() ListParams {
	p.Chain = strings.ToLower(strings.TrimSpace(p.Chain))
	p.Network = strings.ToLower(strings.TrimSpace(p.Network))
	p.Venue = strings.ToLower(strings.TrimSpace(p.Venue))
	return p
}

// Validate validates TradeHub AMM pool list filters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-17: Added.
func (p ListParams) Validate() error {
	p = p.Normalize()
	for name, value := range map[string]string{
		"chain": p.Chain, "network": p.Network, "venue": p.Venue,
	} {
		if value == "" {
			continue
		}
		if len(value) > maxListFilterLength {
			return fmt.Errorf("failed to validate trade hub amm pool list parameters: %w: %s=too_long actual_length=%d max_length=%d", k4k3ruSDKAppError.InvalidParameter(), name, len(value), maxListFilterLength)
		}
		if !listFilterPattern.MatchString(value) {
			return fmt.Errorf("failed to validate trade hub amm pool list parameters: %w: %s=invalid", k4k3ruSDKAppError.InvalidParameter(), name)
		}
	}

	return nil
}

// UnmarshalJSON decodes strict TradeHub AMM pool list parameters.
//
// Parameters:
//   - data: JSON object bytes.
//
// Returns:
//   - Decode or validation error.
//
// Version:
//   - 2026-09-17: Added.
func (p *ListParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return fmt.Errorf("failed to decode trade hub amm pool list parameters: %w: destination=null", k4k3ruSDKAppError.InvalidParameter())
	}
	if trimmed := bytes.TrimSpace(data); len(trimmed) == 0 || trimmed[0] != '{' {
		return fmt.Errorf("failed to decode trade hub amm pool list parameters: %w: json=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	type wire ListParams
	var value wire
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("failed to decode trade hub amm pool list parameters: %w: %w", k4k3ruSDKAppError.InvalidParameter(), err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("failed to decode trade hub amm pool list parameters: %w: json=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	normalized := ListParams(value).Normalize()
	if err := normalized.Validate(); err != nil {
		return fmt.Errorf("failed to decode trade hub amm pool list parameters: %w", err)
	}
	*p = normalized
	return nil
}
