package subscription

import (
	"encoding/json"
	"errors"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestEventJSONRoundTrip(t *testing.T) {
	t.Parallel()

	want := Event{Type: EventTypeAggregation, Data: json.RawMessage(`{"symbol":"BTC/USDC"}`)}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"e":"ag","data":{"symbol":"BTC/USDC"}}` {
		t.Fatalf("Marshal() = %s", data)
	}
	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Type != want.Type || string(got.Data) != string(want.Data) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestEventRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	var event Event
	err := json.Unmarshal([]byte(`{"e":"ag","data":{},"unknown":true}`), &event)
	if err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
		t.Fatalf("Unmarshal() error = %v, want invalid parameter", err)
	}
}

func TestEventValidate(t *testing.T) {
	t.Parallel()

	valid := []Event{
		{Type: EventTypeAggregation, Data: json.RawMessage(`{}`)},
		{Type: EventTypeOrderBook, Data: json.RawMessage(`{}`)},
		{Type: EventTypeArbitrage, Data: json.RawMessage(`{"arbitrageType":"atomic"}`)},
	}
	for _, event := range valid {
		if err := event.Validate(); err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
	}
	invalid := []Event{
		{},
		{Type: "unknown", Data: json.RawMessage(`{}`)},
		{Type: EventTypeAggregation},
		{Type: EventTypeAggregation, Data: json.RawMessage(`invalid`)},
	}
	for _, event := range invalid {
		if err := event.Validate(); err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
			t.Fatalf("Validate() error = %v, want invalid parameter: event=%#v", err, event)
		}
	}
}
