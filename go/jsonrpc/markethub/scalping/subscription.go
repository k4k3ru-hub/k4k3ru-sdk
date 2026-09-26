package scalping

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
)

const SubscriptionIntervalMS uint64 = 1000
const subscriptionNamespace = "MarketHub.Scalping:"

type SubscribeResult struct {
	SubscriptionKey string `json:"subscriptionKey"`
	IntervalMS      uint64 `json:"intervalMs"`
}

type SubscriptionEvent struct {
	SubscriptionKey string `json:"subscriptionKey"`
	Snapshot        Result `json:"snapshot"`
}

type UnsubscribeParams struct {
	SubscriptionKey string `json:"subscriptionKey"`
}

type UnsubscribeResult struct {
	SubscriptionKey string `json:"subscriptionKey"`
}

// SubscriptionKey identifies normalized observation conditions independently of target order.
// The opaque key is not an authorization token or a durable execution identifier.
//
// Version:
//   - 2026-09-26: Added.
func (p Params) SubscriptionKey() (string, error) {
	p = p.Normalize()
	if err := p.Validate(); err != nil {
		return "", fmt.Errorf("failed to identify scalping subscription: %w", err)
	}
	sort.Slice(p.Markets, func(i, j int) bool {
		a, b := p.Markets[i], p.Markets[j]
		for _, pair := range [][2]string{{string(a.Venue), string(b.Venue)}, {string(a.Network), string(b.Network)}, {string(a.Chain), string(b.Chain)}, {a.PoolID, b.PoolID}, {a.VenueSymbol, b.VenueSymbol}} {
			if pair[0] != pair[1] {
				return pair[0] < pair[1]
			}
		}
		return false
	})
	if p.BaseQuantity != nil {
		p.BaseQuantity.Amount = strings.TrimLeft(p.BaseQuantity.Amount, "0")
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", fmt.Errorf("failed to identify scalping subscription: %w", err)
	}
	digest := sha256.Sum256(b)
	return subscriptionNamespace + hex.EncodeToString(digest[:]), nil
}

// ValidateSubscriptionKey validates the namespaced opaque subscription identifier.
//
// Version:
//   - 2026-09-26: Added.
func ValidateSubscriptionKey(key string) error {
	if key == "" {
		return invalid("subscription_key", "empty")
	}
	if len(key) != len(subscriptionNamespace)+sha256.Size*2 || !strings.HasPrefix(key, subscriptionNamespace) {
		return invalid("subscription_key", "invalid")
	}
	hash := strings.TrimPrefix(key, subscriptionNamespace)
	if _, err := hex.DecodeString(hash); err != nil || hash != strings.ToLower(hash) {
		return invalid("subscription_key", "invalid")
	}
	return nil
}

// Validate validates a successful subscription acknowledgement.
//
// Version:
//   - 2026-09-26: Added.
func (r SubscribeResult) Validate() error {
	if err := ValidateSubscriptionKey(r.SubscriptionKey); err != nil {
		return err
	}
	if r.IntervalMS != SubscriptionIntervalMS {
		return invalid("interval_ms", "invalid")
	}
	return nil
}

// Validate validates a complete observation event envelope.
//
// Version:
//   - 2026-09-26: Added.
func (e SubscriptionEvent) Validate() error {
	if err := ValidateSubscriptionKey(e.SubscriptionKey); err != nil {
		return err
	}
	if e.Snapshot.EvaluatedAt <= 0 || e.Snapshot.Buy == nil || e.Snapshot.Sell == nil {
		return invalid("snapshot", "invalid")
	}
	return nil
}

// Validate validates a subscription removal request.
//
// Version:
//   - 2026-09-26: Added.
func (p UnsubscribeParams) Validate() error { return ValidateSubscriptionKey(p.SubscriptionKey) }

// UnmarshalJSON decodes a strict subscription removal request.
//
// Version:
//   - 2026-09-26: Added.
func (p *UnsubscribeParams) UnmarshalJSON(data []byte) error {
	if p == nil {
		return invalid("destination", "null")
	}
	type wire UnsubscribeParams
	var value wire
	if err := jsonobject.Decode(data, &value, "subscriptionKey"); err != nil {
		return fmt.Errorf("failed to decode scalping unsubscribe: %w", err)
	}
	if err := UnsubscribeParams(value).Validate(); err != nil {
		return err
	}
	*p = UnsubscribeParams(value)
	return nil
}
