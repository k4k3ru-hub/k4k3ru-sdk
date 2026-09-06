package spread

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	k4k3ruSDKMarketHubSpread "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/spread"
	k4k3ruSDKTradeHubExecution "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
)

type Params struct {
	Opportunity k4k3ruSDKMarketHubSpread.Params `json:"opportunity"`
	Execution   ExecutionParams                 `json:"execution"`
}

type ExecutionParams struct {
	Signer         string                                    `json:"signer"`
	SubmissionMode k4k3ruSDKTradeHubExecution.SubmissionMode `json:"submissionMode"`
	Conditions     *k4k3ruSDKTradeHubExecution.Conditions    `json:"conditions"`
}

// UnmarshalJSON decodes Trade Hub Spread parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded parameters.
//
// Version:
//   - 2026-09-06: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination=null")
	}
	type wire Params
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded wire
	if err := decoder.Decode(&decoded); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub spread parameters: %w: %w: json=invalid", k4k3ruSDKAppError.InvalidParameter(), err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing json value")
		}
		return k4k3ruSDKAppError.Tracef("failed to decode trade hub spread parameters: %w: %w: json=invalid", k4k3ruSDKAppError.InvalidParameter(), err)
	}
	*p = Params(decoded)
	return nil
}

// Normalize applies canonical formatting to Trade Hub Spread parameters.
//
// Returns:
//   - Normalized parameters.
//
// Version:
//   - 2026-09-06: Added.
func (p Params) Normalize() Params {
	p.Opportunity = p.Opportunity.Normalize()
	p.Execution.Signer = strings.TrimSpace(p.Execution.Signer)
	p.Execution.SubmissionMode = k4k3ruSDKTradeHubExecution.SubmissionMode(strings.ToLower(strings.TrimSpace(string(p.Execution.SubmissionMode))))
	if p.Execution.Conditions != nil {
		conditions := *p.Execution.Conditions
		p.Execution.Conditions = &conditions
	}
	return p
}

// Validate validates Trade Hub Spread parameters.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-09-06: Added.
func (p Params) Validate() error {
	p = p.Normalize()
	if err := p.Opportunity.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade hub spread parameters: %w", err)
	}
	if p.Execution.Signer == "" {
		return invalid("signer=empty")
	}
	if p.Execution.SubmissionMode != k4k3ruSDKTradeHubExecution.SubmissionModeTradeHubRelay {
		return invalid("submission_mode=invalid")
	}
	if p.Execution.Conditions == nil {
		return invalid("conditions=null")
	}
	if err := p.Execution.Conditions.Validate(); err != nil {
		return k4k3ruSDKAppError.Tracef("failed to validate trade hub spread parameters: %w", err)
	}
	return nil
}

// SubscriptionKey builds a stable Trade Hub Spread subscription key.
//
// Returns:
//   - Subscription key.
//   - Validation error.
//
// Version:
//   - 2026-09-06: Added.
func (p Params) SubscriptionKey() (string, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return "", k4k3ruSDKAppError.Tracef("failed to build trade hub spread subscription key: %w", err)
	}
	opportunityKey, err := p.Opportunity.SubscriptionKey()
	if err != nil {
		return "", k4k3ruSDKAppError.Tracef("failed to build trade hub spread subscription key: %w", err)
	}
	conditions, err := json.Marshal(p.Execution.Conditions)
	if err != nil {
		return "", k4k3ruSDKAppError.Tracef("failed to build trade hub spread subscription key: %w", err)
	}
	return "TradeHub.Spread:" + opportunityKey + ":signer=" + p.Execution.Signer + ":mode=" + string(p.Execution.SubmissionMode) + ":conditions=" + string(conditions), nil
}

func invalid(state string) error {
	return k4k3ruSDKAppError.Tracef("failed to validate trade hub spread parameters: %w: %s", k4k3ruSDKAppError.InvalidParameter(), state)
}
