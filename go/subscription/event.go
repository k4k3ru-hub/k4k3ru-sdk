package subscription

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

type EventType string

const (
	EventTypeAggregation EventType = "ag"
	EventTypeArbitrage   EventType = "ar"
	EventTypeOrderBook   EventType = "ob"
	EventTypeExecution   EventType = "ex"
)

type Event struct {
	Type EventType       `json:"e"`
	Data json.RawMessage `json:"data"`
}

// UnmarshalJSON decodes a subscription event and rejects unknown fields.
//
// Parameters:
//   - data: JSON-encoded subscription event.
//
// Version:
//   - 2026-08-30: Added.
func (e *Event) UnmarshalJSON(data []byte) error {
	if e == nil {
		return k4k3ruSDKAppError.Tracef("failed to decode subscription event: %w: destination=null", k4k3ruSDKAppError.InvalidParameter())
	}
	type wireEvent Event
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var decoded wireEvent
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
	*e = Event(decoded)
	return nil
}

// Validate validates a public subscription event.
//
// Returns:
//   - Validation error.
//
// Version:
//   - 2026-08-30: Added.
func (e Event) Validate() error {
	switch e.Type {
	case EventTypeAggregation, EventTypeArbitrage, EventTypeOrderBook, EventTypeExecution:
	default:
		if e.Type == "" {
			return k4k3ruSDKAppError.Tracef("failed to validate subscription event: %w: event_type=empty", k4k3ruSDKAppError.InvalidParameter())
		}
		return k4k3ruSDKAppError.Tracef("failed to validate subscription event: %w: event_type=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	data := bytes.TrimSpace(e.Data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return k4k3ruSDKAppError.Tracef("failed to validate subscription event: %w: data=empty", k4k3ruSDKAppError.InvalidParameter())
	}
	if !json.Valid(data) {
		return k4k3ruSDKAppError.Tracef("failed to validate subscription event: %w: data=invalid", k4k3ruSDKAppError.InvalidParameter())
	}
	return nil
}

func invalidJSONError(err error) error {
	return k4k3ruSDKAppError.Tracef("failed to decode subscription event: %w: %w: json=invalid", k4k3ruSDKAppError.InvalidParameter(), err)
}
