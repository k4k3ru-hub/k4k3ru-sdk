package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
)

type MarketHubScalpingClient struct {
	opMu      sync.Mutex
	sender    jsonRPCSender
	events    *marketHubScalpingEvents
	lifecycle *subscriptionLifecycle
}

type MarketHubScalpingSubscription struct {
	registry *marketHubScalpingEvents
	key      string
	events   chan dto.Result
	errors   chan error
	closed   bool
}

type marketHubScalpingEvents struct {
	mu     sync.Mutex
	active map[string]*MarketHubScalpingSubscription
}

func newMarketHubScalpingEvents() *marketHubScalpingEvents {
	return &marketHubScalpingEvents{active: make(map[string]*MarketHubScalpingSubscription)}
}

func newMarketHubScalpingClient(sender jsonRPCSender, events *marketHubScalpingEvents, lifecycle *subscriptionLifecycle) (*MarketHubScalpingClient, error) {
	if sender == nil || events == nil || lifecycle == nil {
		return nil, fmt.Errorf("failed to create market hub scalping client: dependency=null")
	}
	return &MarketHubScalpingClient{sender: sender, events: events, lifecycle: lifecycle}, nil
}

// Subscribe receives complete latest-value observations for one normalized request.
// Repeated calls with the same active conditions return the same subscription.
// After disconnection callers explicitly subscribe again; no history is replayed.
//
// Version:
//   - 2026-09-26: Added.
func (c *MarketHubScalpingClient) Subscribe(ctx context.Context, params dto.Params) (*MarketHubScalpingSubscription, error) {
	if c == nil || ctx == nil {
		return nil, fmt.Errorf("failed to subscribe market hub scalping: dependency=null")
	}
	params = params.Normalize()
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub scalping: %w", err)
	}
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.lifecycle.mu.Lock()
	c.lifecycle.keepOpen = true
	c.lifecycle.mu.Unlock()
	r := c.events
	r.mu.Lock()
	if existing := r.active[key]; existing != nil {
		r.mu.Unlock()
		return existing, nil
	}
	s := &MarketHubScalpingSubscription{registry: r, key: key, events: make(chan dto.Result, 1), errors: make(chan error, 1)}
	r.active[key] = s
	r.mu.Unlock()
	var ack dto.SubscribeResult
	err = c.request(ctx, rpc.MethodMarketHubScalpingSubscribe, params, &ack)
	if err == nil {
		err = ack.Validate()
	}
	if err == nil && ack.SubscriptionKey != key {
		err = fmt.Errorf("failed to subscribe market hub scalping: acknowledgement=invalid")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		r.finish(s, err)
		return nil, err
	}
	if s.closed {
		return nil, fmt.Errorf("failed to subscribe market hub scalping: subscription interrupted")
	}
	return s, nil
}

// Unsubscribe stops a subscription after its acknowledgement, keeping other streams open.
//
// Version:
//   - 2026-09-26: Added.
func (c *MarketHubScalpingClient) Unsubscribe(ctx context.Context, s *MarketHubScalpingSubscription) error {
	if c == nil || ctx == nil || s == nil || s.registry != c.events {
		return fmt.Errorf("failed to unsubscribe market hub scalping: subscription=invalid")
	}
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.events.mu.Lock()
	closed := s.closed
	c.events.mu.Unlock()
	if closed {
		return nil
	}
	var ack dto.UnsubscribeResult
	if err := c.request(ctx, rpc.MethodMarketHubScalpingUnsubscribe, dto.UnsubscribeParams{SubscriptionKey: s.key}, &ack); err != nil {
		return err
	}
	if ack.SubscriptionKey != s.key {
		return fmt.Errorf("failed to unsubscribe market hub scalping: acknowledgement=invalid")
	}
	c.events.mu.Lock()
	defer c.events.mu.Unlock()
	c.events.finish(s, nil)
	return nil
}

// Events returns full observations; a slow reader retains only the latest unread value.
// Replace the previous Result, including fields omitted from the new observation.
//
// Version:
//   - 2026-09-26: Added.
func (s *MarketHubScalpingSubscription) Events() <-chan dto.Result {
	if s == nil {
		return nil
	}
	return s.events
}

// Errors reports termination or transport failure; discard the previous observation on error.
//
// Version:
//   - 2026-09-26: Added.
func (s *MarketHubScalpingSubscription) Errors() <-chan error {
	if s == nil {
		return nil
	}
	return s.errors
}

// Reference returns the subscription key and fixed target delivery interval.
//
// Version:
//   - 2026-09-26: Added.
func (s *MarketHubScalpingSubscription) Reference() dto.SubscribeResult {
	if s == nil {
		return dto.SubscribeResult{}
	}
	return dto.SubscribeResult{SubscriptionKey: s.key, IntervalMS: dto.SubscriptionIntervalMS}
}

func (c *MarketHubScalpingClient) request(ctx context.Context, method rpc.Method, params, result any) error {
	encoded, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to encode market hub scalping request: %w", err)
	}
	response, err := c.sender.send(ctx, method, encoded)
	if err != nil {
		return fmt.Errorf("failed to send market hub scalping request: %w", err)
	}
	if response == nil {
		return fmt.Errorf("failed to receive market hub scalping acknowledgement: response=null")
	}
	if response.Error != nil {
		return fmt.Errorf("failed to receive market hub scalping acknowledgement: %w", response.Error)
	}
	if err := json.Unmarshal(response.Result, result); err != nil {
		return fmt.Errorf("failed to decode market hub scalping acknowledgement: %w", err)
	}
	return nil
}

func (r *marketHubScalpingEvents) route(event dto.SubscriptionEvent) error {
	if r == nil {
		return fmt.Errorf("failed to route market hub scalping event: registry=null")
	}
	if err := event.Validate(); err != nil {
		err = fmt.Errorf("failed to route market hub scalping event: %w", err)
		r.terminate(event.SubscriptionKey, err)
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if s := r.active[event.SubscriptionKey]; s != nil {
		select {
		case <-s.events:
		default:
		}
		s.events <- event.Snapshot
	}
	return nil
}

func (r *marketHubScalpingEvents) terminate(key string, err error) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if s := r.active[key]; s != nil {
		r.finish(s, err)
	}
}

func (r *marketHubScalpingEvents) interrupt() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.active {
		r.finish(s, errWebSocketConnectionClosed)
	}
}

// finish requires the registry lock; draining prevents stale buffered values after termination.
func (r *marketHubScalpingEvents) finish(s *MarketHubScalpingSubscription, err error) {
	if s.closed {
		return
	}
	s.closed = true
	if r.active[s.key] == s {
		delete(r.active, s.key)
	}
	select {
	case <-s.events:
	default:
	}
	if err != nil {
		s.errors <- err
	}
	close(s.events)
	close(s.errors)
}
