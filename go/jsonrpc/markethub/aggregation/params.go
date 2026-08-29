package aggregation

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

type Params struct {
	AssetClass      AssetClass               `json:"assetClass,omitempty"`
	MarketType      MarketType               `json:"marketType"`
	Symbol          Symbol                   `json:"symbol"`
	AggregationMode AggregationMode          `json:"aggregationMode,omitempty"`
	SourceFilter    *AggregationSourceFilter `json:"sourceFilter,omitempty"`
}

// UnmarshalJSON decodes aggregation parameters and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded aggregation parameters.
//
// Version:
//   - 2026-08-29: Added.
func (p *Params) UnmarshalJSON(data []byte) error {
	if p == nil {
		return k4k3ruSDKAppError.Tracef("failed to decode aggregation parameters: %w: destination=null", k4k3ruSDKAppError.InvalidParameter())
	}
	type wireParams Params
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded wireParams
	if err := decoder.Decode(&decoded); err != nil {
		return invalidJSONError(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected trailing json value")
		}
		return invalidJSONError(err)
	}
	*p = Params(decoded)
	return nil
}

func invalidJSONError(err error) error {
	return k4k3ruSDKAppError.Tracef("failed to decode aggregation parameters: %w: %w: json=invalid", k4k3ruSDKAppError.InvalidParameter(), err)
}
