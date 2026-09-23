package scalping

import (
	"fmt"
	"strings"

	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// Validate validates a subscription acknowledgement.
//
// Version:
//   - 2026-09-23: Added.
func (r SubscribeResult) Validate() error {
	return v.Text("validate scalping subscription", "subscription_key", r.SubscriptionKey, 128)
}

// UnmarshalJSON decodes a validated subscription acknowledgement.
//
// Version:
//   - 2026-09-23: Added.
func (r *SubscribeResult) UnmarshalJSON(data []byte) error {
	if r == nil {
		return v.Invalid("decode scalping subscription", "destination", "null")
	}
	type wire SubscribeResult
	var decoded wire
	if err := v.Decode(data, &decoded, "subscriptionKey"); err != nil {
		return fmt.Errorf("failed to decode scalping subscription: %w", err)
	}
	value := SubscribeResult(decoded)
	if err := value.Validate(); err != nil {
		return err
	}
	*r = value
	return nil
}

// Normalize trims the opaque subscription key without changing its case.
//
// Version:
//   - 2026-09-23: Added.
func (p UnsubscribeParams) Normalize() UnsubscribeParams {
	p.SubscriptionKey = strings.TrimSpace(p.SubscriptionKey)
	return p
}

// Validate validates the subscription to stop without cancelling any orders.
//
// Version:
//   - 2026-09-23: Added.
func (p UnsubscribeParams) Validate() error {
	p = p.Normalize()
	return v.Text("validate scalping unsubscription", "subscription_key", p.SubscriptionKey, 128)
}

// UnmarshalJSON decodes an error with an explicit retryability decision.
//
// Version:
//   - 2026-09-23: Added.
func (e *StreamError) UnmarshalJSON(data []byte) error {
	if e == nil {
		return v.Invalid("decode scalping stream error", "destination", "null")
	}
	type wire StreamError
	var decoded wire
	if err := v.Decode(data, &decoded, "code", "retryable"); err != nil {
		return fmt.Errorf("failed to decode scalping stream error: %w", err)
	}
	if err := v.Text("decode scalping stream error", "code", decoded.Code, 64); err != nil {
		return err
	}
	*e = StreamError(decoded)
	return nil
}

// UnmarshalJSON decodes a validated unsubscription request.
//
// Version:
//   - 2026-09-23: Added.
func (p *UnsubscribeParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return v.Invalid("decode scalping unsubscription", "destination", "null")
	}
	type wire UnsubscribeParams
	var decoded wire
	if err := v.Decode(data, &decoded, "subscriptionKey"); err != nil {
		return fmt.Errorf("failed to decode scalping unsubscription: %w", err)
	}
	value := UnsubscribeParams(decoded).Normalize()
	if err := value.Validate(); err != nil {
		return err
	}
	*p = value
	return nil
}

// Validate validates an unsubscription acknowledgement.
//
// Version:
//   - 2026-09-23: Added.
func (r UnsubscribeResult) Validate() error {
	return v.Text("validate scalping unsubscription result", "subscription_key", r.SubscriptionKey, 128)
}

// UnmarshalJSON decodes a validated unsubscription acknowledgement.
//
// Version:
//   - 2026-09-23: Added.
func (r *UnsubscribeResult) UnmarshalJSON(data []byte) error {
	if r == nil {
		return v.Invalid("decode scalping unsubscription result", "destination", "null")
	}
	type wire UnsubscribeResult
	var decoded wire
	if err := v.Decode(data, &decoded, "subscriptionKey"); err != nil {
		return fmt.Errorf("failed to decode scalping unsubscription result: %w", err)
	}
	value := UnsubscribeResult(decoded)
	if err := value.Validate(); err != nil {
		return err
	}
	*r = value
	return nil
}

// Validate validates exactly one snapshot or stream error and its sequence.
// Sequence ordering and connection ownership are checked by the event consumer.
//
// Version:
//   - 2026-09-23: Added.
func (e SubscriptionEvent) Validate() error {
	const op = "validate scalping event"
	if err := v.Text(op, "subscription_key", e.SubscriptionKey, 128); err != nil {
		return err
	}
	if e.Sequence == 0 {
		return v.Invalid(op, "sequence", "empty")
	}
	switch e.Kind {
	case EventKindSnapshot:
		if e.Snapshot == nil || e.Error != nil {
			return v.Invalid(op, "snapshot", "invalid")
		}
		if err := e.Snapshot.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping event: %w", err)
		}
	case EventKindError:
		if e.Error == nil || e.Snapshot != nil {
			return v.Invalid(op, "error", "invalid")
		}
		if err := v.Text(op, "code", e.Error.Code, 64); err != nil {
			return err
		}
	default:
		return v.Invalid(op, "kind", "invalid")
	}
	return nil
}

// UnmarshalJSON decodes a validated snapshot or error event.
//
// Version:
//   - 2026-09-23: Added.
func (e *SubscriptionEvent) UnmarshalJSON(data []byte) error {
	if e == nil {
		return v.Invalid("decode scalping event", "destination", "null")
	}
	type wire SubscriptionEvent
	var decoded wire
	if err := v.Decode(data, &decoded, "subscriptionKey", "sequence", "kind"); err != nil {
		return fmt.Errorf("failed to decode scalping event: %w", err)
	}
	value := SubscriptionEvent(decoded)
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping event: %w", err)
	}
	*e = value
	return nil
}
