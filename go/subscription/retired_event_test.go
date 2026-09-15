package subscription

import (
	"encoding/json"
	"testing"
)

// TestRetiredPoolEvent verifies only the current discovery event is accepted.
//
// Version:
//   - 2026-09-16: Added.
func TestRetiredPoolEvent(t *testing.T) {
	if err := (Event{Type: "apl", Data: json.RawMessage(`{}`)}).Validate(); err == nil {
		t.Fatal("retired event accepted")
	}
	if err := (Event{Type: EventTypeAMMPoolNewPair, Data: json.RawMessage(`{}`)}).Validate(); err != nil {
		t.Fatal(err)
	}
}
