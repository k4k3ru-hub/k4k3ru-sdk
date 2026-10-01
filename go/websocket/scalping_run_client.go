package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping"
)

type scalpingRunClient struct {
	opMu      sync.Mutex
	sender    jsonRPCSender
	events    *scalpingRunEventRegistry
	lifecycle *subscriptionLifecycle
}

type ScalpingRunSubscription struct {
	registry *scalpingRunEventRegistry
	id, key  string
	params   *dto.RunConfiguration
	sequence uint64
	events   chan dto.RunEvent
	errors   chan error
	buffer   []dto.RunEvent
	closed   bool
	released bool
}

type scalpingRunEventRegistry struct {
	mu      sync.Mutex
	active  map[string]*ScalpingRunSubscription
	pending *ScalpingRunSubscription
}

func newScalpingRunEventRegistry() *scalpingRunEventRegistry {
	return &scalpingRunEventRegistry{active: make(map[string]*ScalpingRunSubscription)}
}

func newScalpingRunClient(sender jsonRPCSender, events *scalpingRunEventRegistry, lifecycle *subscriptionLifecycle) (*scalpingRunClient, error) {
	if sender == nil || events == nil || lifecycle == nil {
		return nil, fmt.Errorf("failed to create scalping websocket client: dependency=null")
	}
	return &scalpingRunClient{sender: sender, events: events, lifecycle: lifecycle}, nil
}

// Run starts an idempotent execution or resumes its candidate notifications.
// Reconnection is explicit; retain the acknowledgement's execution ID to resume.
//
// Version:
//   - 2026-10-01: Added.
func (c *scalpingRunClient) Run(ctx context.Context, params dto.RunParams) (*ScalpingRunSubscription, error) {
	if c == nil || ctx == nil {
		return nil, fmt.Errorf("failed to run scalping: dependency=null")
	}
	params = params.Normalize()
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to run scalping: %w", err)
	}
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.lifecycle.mu.Lock()
	c.lifecycle.keepOpen = true
	c.lifecycle.mu.Unlock()
	r := c.events
	r.mu.Lock()
	if s := r.active[params.ExecutionID]; s != nil {
		r.mu.Unlock()
		return s, nil
	}
	s := &ScalpingRunSubscription{registry: r, events: make(chan dto.RunEvent, 16), errors: make(chan error, 1)}
	r.pending = s
	r.mu.Unlock()
	var ack dto.RunResult
	err := c.request(ctx, rpc.MethodTradeHubScalpingRun, params, &ack)
	if err == nil && params.Params != nil && (ack.Params == nil || !reflect.DeepEqual(*params.Params, *ack.Params)) {
		err = fmt.Errorf("failed to run scalping: acknowledgement_configuration=mismatch")
	}
	if err == nil && params.ExecutionID != "" && ack.ExecutionID != params.ExecutionID {
		err = fmt.Errorf("failed to run scalping: execution_id=invalid")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pending == s {
		r.pending = nil
	}
	if err != nil {
		r.finish(s, err)
		return nil, err
	}
	if s.closed {
		return nil, fmt.Errorf("failed to run scalping: subscription interrupted")
	}
	if old := r.active[ack.ExecutionID]; old != nil {
		if old.key == ack.SubscriptionKey {
			r.finish(s, nil)
			return old, nil
		}
		r.finish(old, fmt.Errorf("failed to receive scalping event: subscription replaced"))
	}
	s.id, s.key = ack.ExecutionID, ack.SubscriptionKey
	if ack.Params != nil {
		settings := ack.Params.Normalize()
		s.params = &settings
	}
	r.active[s.id] = s
	buffer := s.buffer
	s.buffer = nil
	for _, event := range buffer {
		r.deliver(s, event)
	}
	return s, nil
}

// Unsubscribe stops Run delivery without cancelling orders or positions.
//
// Version:
//   - 2026-10-01: Added.
func (c *scalpingRunClient) Unsubscribe(ctx context.Context, s *ScalpingRunSubscription) error {
	if c == nil || ctx == nil || s == nil || s.registry != c.events {
		return fmt.Errorf("failed to unsubscribe scalping run: subscription=invalid")
	}
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.events.mu.Lock()
	released := s.released
	params := dto.UnsubscribeParams{ExecutionID: s.id, SubscriptionKey: s.key}
	c.events.mu.Unlock()
	if released {
		return nil
	}
	var ack dto.UnsubscribeResult
	if err := c.request(ctx, rpc.MethodTradeHubScalpingUnsubscribe, params, &ack); err != nil {
		return err
	}
	if ack.ExecutionID != params.ExecutionID || ack.SubscriptionKey != params.SubscriptionKey {
		return fmt.Errorf("failed to unsubscribe scalping run: acknowledgement=invalid")
	}
	c.events.mu.Lock()
	defer c.events.mu.Unlock()
	s.released = true
	c.events.finish(s, nil)
	return nil
}

// Events returns complete candidate snapshots and server stream errors.
// Retryable stream errors withdraw prior candidates until a new snapshot arrives.
//
// Version:
//   - 2026-10-01: Added.
func (s *ScalpingRunSubscription) Events() <-chan dto.RunEvent {
	if s == nil {
		return nil
	}
	return s.events
}

// Errors returns local transport and delivery interruptions.
//
// Version:
//   - 2026-10-01: Added.
func (s *ScalpingRunSubscription) Errors() <-chan error {
	if s == nil {
		return nil
	}
	return s.errors
}

// Reference returns detached settings and the durable and connection-specific IDs.
//
// Version:
//   - 2026-10-01: Added.
func (s *ScalpingRunSubscription) Reference() dto.RunResult {
	if s == nil {
		return dto.RunResult{}
	}
	s.registry.mu.Lock()
	defer s.registry.mu.Unlock()
	ref := dto.RunResult{ExecutionID: s.id, SubscriptionKey: s.key}
	if s.params != nil {
		settings := s.params.Normalize()
		ref.Params = &settings
	}
	return ref
}

func (c *scalpingRunClient) request(ctx context.Context, method rpc.Method, params, result any) error {
	encoded, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to encode scalping request: %w", err)
	}
	response, err := c.sender.send(ctx, method, encoded)
	if err != nil {
		return fmt.Errorf("failed to send scalping request: %w", err)
	}
	if response == nil {
		return fmt.Errorf("failed to send scalping request: response=null")
	}
	if response.Error != nil {
		return fmt.Errorf("failed to send scalping request: %w", response.Error)
	}
	if err := json.Unmarshal(response.Result, result); err != nil {
		return fmt.Errorf("failed to decode scalping acknowledgement: %w", err)
	}
	return nil
}

func (r *scalpingRunEventRegistry) route(event dto.RunEvent) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if s := r.active[event.ExecutionID]; s != nil && s.key == event.SubscriptionKey {
		r.deliver(s, event)
		return
	}
	if s := r.pending; s != nil && !s.closed {
		if len(s.buffer) >= 16 {
			r.finish(s, fmt.Errorf("failed to receive scalping event: acknowledgement buffer full: %w", ErrSubscriptionOverflow))
			return
		}
		s.buffer = append(s.buffer, event)
	}
}

func (r *scalpingRunEventRegistry) deliver(s *ScalpingRunSubscription, event dto.RunEvent) {
	if s.closed || s.id != event.ExecutionID || s.key != event.SubscriptionKey || event.Sequence <= s.sequence {
		return
	}
	s.sequence = event.Sequence
	select {
	case s.events <- event:
	default:
		r.finish(s, fmt.Errorf("failed to receive scalping event: event buffer full: %w", ErrSubscriptionOverflow))
		return
	}
	if event.Error != nil && !event.Error.Retryable {
		r.finish(s, nil)
	}
}

func (r *scalpingRunEventRegistry) finish(s *ScalpingRunSubscription, err error) {
	if s.closed {
		return
	}
	s.closed = true
	s.buffer = nil
	if r.active[s.id] == s {
		delete(r.active, s.id)
	}
	if r.pending == s {
		r.pending = nil
	}
	if err != nil {
		s.errors <- err
	}
	close(s.errors)
	close(s.events)
}

func (r *scalpingRunEventRegistry) interrupt() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pending != nil {
		r.pending.released = true
		r.finish(r.pending, errWebSocketConnectionClosed)
	}
	for _, s := range r.active {
		s.released = true
		r.finish(s, errWebSocketConnectionClosed)
	}
}
