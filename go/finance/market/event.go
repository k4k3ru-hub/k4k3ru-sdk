package market

import (
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// EventType identifies legacy finance data events, not public RPC notifications.
type EventType string

const (
	EventMarketData EventType = "md"
	EventTrade      EventType = "tr"
	EventTicker     EventType = "tk"
	EventCandle     EventType = "kl"
	EventMidPrice   EventType = "mp"
)

// Validate preserves the accepted event types of the copied subscription model.
// Constants for other event kinds do not imply validation support.
//
// Version:
//   - 2026-09-25: Copied the required finance dependency without server RPC routing.
func (t EventType) Validate() error {
	if t == "" {
		return fmt.Errorf("failed to validate finance event type: %w: event_type=empty", apperror.InvalidParameter())
	}
	if len(t) > 8 {
		return fmt.Errorf("failed to validate finance event type: %w: event_type=too_long actual_length=%d max_length=8", apperror.InvalidParameter(), len(t))
	}
	switch t {
	case EventMarketData, EventTrade:
		return nil
	default:
		return fmt.Errorf("failed to validate finance event type: %w: event_type=invalid", apperror.InvalidParameter())
	}
}
