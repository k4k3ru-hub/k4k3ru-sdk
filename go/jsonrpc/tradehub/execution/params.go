package execution

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

type Intent struct {
	Kind         IntentKind `json:"kind"`
	EvaluationID string     `json:"evaluationId"`
	RouteID      string     `json:"routeId"`
}

type Conditions struct {
	MinimumNetProfit        string  `json:"minimumNetProfit"`
	MaximumSlippageBPS      *uint64 `json:"maximumSlippageBps"`
	MaximumExecutionCost    string  `json:"maximumExecutionCost,omitempty"`
	MaximumOpportunityAgeMS *uint64 `json:"maximumOpportunityAgeMs"`
	ExecutionTTLMS          *uint64 `json:"executionTtlMs"`
}

type Params struct {
	Intent         Intent         `json:"intent"`
	Signer         string         `json:"signer"`
	SubmissionMode SubmissionMode `json:"submissionMode"`
	Conditions     *Conditions    `json:"conditions"`
	IdempotencyKey string         `json:"idempotencyKey"`
}

// Normalize applies canonical formatting to execution preparation parameters.
//
// Returns:
//   - Normalized parameters.
//
// Version:
//   - 2026-09-03: Added.
func (p Params) Normalize() Params {
	p.Intent.Kind = IntentKind(strings.ToLower(strings.TrimSpace(string(p.Intent.Kind))))
	p.Intent.EvaluationID = strings.TrimSpace(p.Intent.EvaluationID)
	p.Intent.RouteID = strings.TrimSpace(p.Intent.RouteID)
	p.Signer = strings.TrimSpace(p.Signer)
	p.SubmissionMode = SubmissionMode(strings.ToLower(strings.TrimSpace(string(p.SubmissionMode))))
	p.IdempotencyKey = strings.TrimSpace(p.IdempotencyKey)
	if p.Conditions != nil {
		conditions := *p.Conditions
		conditions.MinimumNetProfit = normalizeDecimal(conditions.MinimumNetProfit)
		conditions.MaximumExecutionCost = normalizeDecimal(conditions.MaximumExecutionCost)
		p.Conditions = &conditions
	}
	return p
}

// Validate validates execution preparation parameters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-03: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if p.Intent.Kind != IntentKindAtomicArbitrage {
		return invalidParameterError("intent_kind=invalid")
	}
	if p.Intent.EvaluationID == "" {
		return invalidParameterError("evaluation_id=empty")
	}
	if p.Intent.RouteID == "" {
		return invalidParameterError("route_id=empty")
	}
	if p.Signer == "" {
		return invalidParameterError("signer=empty")
	}
	if p.SubmissionMode != SubmissionModeTradeHubRelay {
		return invalidParameterError("submission_mode=invalid")
	}
	if p.Conditions == nil {
		return invalidParameterError("conditions=null")
	}
	if err := p.Conditions.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade hub execution preparation parameters: %w", err)
	}
	if p.IdempotencyKey == "" {
		return invalidParameterError("idempotency_key=empty")
	}
	return nil
}

// Validate validates execution conditions.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-03: Added.
func (c Conditions) Validate() error {
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

// UnmarshalJSON decodes execution preparation parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded parameters.
//
// Version:
//   - 2026-09-03: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalidParameterError("destination=null")
	}
	type wireParams Params
	var decoded wireParams
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub execution preparation parameters: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing json value")
		}
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub execution preparation parameters: %w", err)
	}
	*p = Params(decoded)
	return nil
}

func validateNonNegativeDecimal(value string, field string, optional bool) error {
	value = strings.TrimSpace(value)
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
	normalized := strings.TrimRight(strings.TrimRight(decimal.FloatString(scale), "0"), ".")
	if normalized == "" || normalized == "-0" {
		return "0"
	}
	return normalized
}

func invalidParameterError(reason string) error {
	return k4k3ruSDKAppError.Tracef("failed to validate trade hub execution preparation parameters: %w: %s", k4k3ruSDKAppError.InvalidParameter(), reason)
}
