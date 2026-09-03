package arbitrage

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	k4k3ruSDKMarketHubArbitrage "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/arbitrage"
	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

type Params struct {
	Opportunity k4k3ruSDKMarketHubArbitrage.Params `json:"opportunity"`
	Execution   ExecutionParams                    `json:"execution"`
}

type ExecutionParams struct {
	Signer         string               `json:"signer"`
	SubmissionMode SubmissionMode       `json:"submissionMode"`
	Conditions     *ExecutionConditions `json:"conditions"`
}

type ExecutionConditions struct {
	MinimumNetProfit        string  `json:"minimumNetProfit"`
	MaximumSlippageBPS      *uint64 `json:"maximumSlippageBps"`
	MaximumExecutionCost    string  `json:"maximumExecutionCost,omitempty"`
	MaximumOpportunityAgeMS *uint64 `json:"maximumOpportunityAgeMs"`
	ExecutionTTLMS          *uint64 `json:"executionTtlMs"`
}

// UnmarshalJSON decodes Trade Hub Arbitrage subscription parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded subscription parameters.
//
// Version:
//   - 2026-09-03: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalidParameterError("destination=null")
	}
	type wireParams Params
	var decoded wireParams
	if err := decodeStrictJSON(data, &decoded); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub arbitrage parameters: %w", err)
	}
	*p = Params(decoded)
	return nil
}

// UnmarshalJSON decodes execution parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded execution parameters.
//
// Version:
//   - 2026-09-03: Added.
func (p *ExecutionParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalidParameterError("destination=null")
	}
	type wireParams ExecutionParams
	var decoded wireParams
	if err := decodeStrictJSON(data, &decoded); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub arbitrage execution parameters: %w", err)
	}
	*p = ExecutionParams(decoded)
	return nil
}

// UnmarshalJSON decodes execution conditions and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded execution conditions.
//
// Version:
//   - 2026-09-03: Added.
func (c *ExecutionConditions) UnmarshalJSON(data []byte) error {
	if c == nil {
		return invalidParameterError("destination=null")
	}
	type wireConditions ExecutionConditions
	var decoded wireConditions
	if err := decodeStrictJSON(data, &decoded); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub arbitrage execution conditions: %w", err)
	}
	*c = ExecutionConditions(decoded)
	return nil
}

// Normalize applies canonical formatting to Trade Hub Arbitrage subscription parameters.
//
// Returns:
//   - Normalized subscription parameters.
//
// Version:
//   - 2026-09-03: Added.
func (p Params) Normalize() Params {
	p.Opportunity = p.Opportunity.Normalize()
	p.Execution = p.Execution.Normalize()
	return p
}

// Validate validates Trade Hub Arbitrage subscription parameters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-03: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if err := p.Opportunity.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade hub arbitrage parameters: %w", err)
	}
	if p.Opportunity.Chain != k4k3ruOnchainCore.ChainSui && p.Opportunity.Chain != k4k3ruOnchainCore.ChainBase {
		return invalidParameterError("opportunity_chain=invalid")
	}
	if err := p.Execution.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade hub arbitrage parameters: %w", err)
	}
	return nil
}

// Normalize applies canonical formatting to Trade Hub execution parameters.
//
// Returns:
//   - Normalized execution parameters.
//
// Version:
//   - 2026-09-03: Added.
func (p ExecutionParams) Normalize() ExecutionParams {
	p.Signer = strings.TrimSpace(p.Signer)
	p.SubmissionMode = SubmissionMode(strings.ToLower(strings.TrimSpace(string(p.SubmissionMode))))
	if p.Conditions != nil {
		normalized := p.Conditions.Normalize()
		p.Conditions = &normalized
	}
	return p
}

// Validate validates Trade Hub execution parameters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-03: Added.
func (p ExecutionParams) Validate() error {
	p = p.Normalize()
	if p.Signer == "" {
		return invalidParameterError("signer=empty")
	}
	if err := p.SubmissionMode.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade hub arbitrage execution parameters: %w", err)
	}
	if p.Conditions == nil {
		return invalidParameterError("conditions=null")
	}
	if err := p.Conditions.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade hub arbitrage execution parameters: %w", err)
	}
	return nil
}

// Normalize applies canonical formatting to Trade Hub execution conditions.
//
// Returns:
//   - Normalized execution conditions.
//
// Version:
//   - 2026-09-03: Added.
func (c ExecutionConditions) Normalize() ExecutionConditions {
	c.MinimumNetProfit = normalizeDecimal(c.MinimumNetProfit)
	c.MaximumExecutionCost = normalizeDecimal(c.MaximumExecutionCost)
	return c
}

// Validate validates Trade Hub execution conditions.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-03: Added.
func (c ExecutionConditions) Validate() error {
	c = c.Normalize()
	if err := validateNonNegativeDecimal(c.MinimumNetProfit, "minimum_net_profit", false); err != nil {
		return err
	}
	if c.MaximumSlippageBPS == nil {
		return invalidParameterError("maximum_slippage_bps=null")
	}
	if err := validateNonNegativeDecimal(c.MaximumExecutionCost, "maximum_execution_cost", true); err != nil {
		return err
	}
	if c.MaximumOpportunityAgeMS == nil {
		return invalidParameterError("maximum_opportunity_age_ms=null")
	}
	if *c.MaximumOpportunityAgeMS == 0 {
		return invalidParameterError("maximum_opportunity_age_ms=empty")
	}
	if c.ExecutionTTLMS == nil {
		return invalidParameterError("execution_ttl_ms=null")
	}
	if *c.ExecutionTTLMS == 0 {
		return invalidParameterError("execution_ttl_ms=empty")
	}
	return nil
}

func decodeStrictJSON(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return invalidJSONError(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing json value")
		}
		return invalidJSONError(err)
	}
	return nil
}

func validateNonNegativeDecimal(value string, field string, optional bool) error {
	if value == "" {
		if optional {
			return nil
		}
		return invalidParameterError(field + "=empty")
	}
	if strings.ContainsAny(value, "/eE") {
		return invalidParameterError(field + "=invalid")
	}
	decimal := new(big.Rat)
	if _, ok := decimal.SetString(value); !ok {
		return invalidParameterError(field + "=invalid")
	}
	if decimal.Sign() < 0 {
		return invalidParameterError(field + "=out_of_range")
	}
	return nil
}

func normalizeDecimal(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "/eE") {
		return value
	}
	decimal := new(big.Rat)
	if _, ok := decimal.SetString(value); !ok {
		return value
	}
	scale := 0
	if point := strings.IndexByte(value, '.'); point >= 0 {
		scale = len(value) - point - 1
	}
	normalized := decimal.FloatString(scale)
	if strings.Contains(normalized, ".") {
		normalized = strings.TrimRight(normalized, "0")
		normalized = strings.TrimRight(normalized, ".")
	}
	if normalized == "" || normalized == "-0" {
		return "0"
	}
	return normalized
}

func invalidJSONError(err error) error {
	return k4k3ruSDKAppError.Tracef("failed to decode json: %w: %w: json=invalid", k4k3ruSDKAppError.InvalidParameter(), err)
}

func invalidParameterError(reason string) error {
	return k4k3ruSDKAppError.Tracef("failed to validate trade hub arbitrage parameters: %w: %s", k4k3ruSDKAppError.InvalidParameter(), reason)
}
