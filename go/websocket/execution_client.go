package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
)

type ExecutionClient struct {
	opMu      sync.Mutex
	sender    jsonRPCSender
	events    *executionEventRegistry
	lifecycle *subscriptionLifecycle
}
type ExecutionSubscription struct {
	registry *executionEventRegistry
	id, key  string
	sequence uint64
	events   chan dto.SubscriptionEvent
	errors   chan error
	buffer   []dto.SubscriptionEvent
	closed   bool
}
type executionEventRegistry struct {
	mu     sync.Mutex
	active map[string]*ExecutionSubscription
}

func newExecutionEventRegistry() *executionEventRegistry {
	return &executionEventRegistry{active: make(map[string]*ExecutionSubscription)}
}
func newExecutionClient(sender jsonRPCSender, events *executionEventRegistry, lifecycle *subscriptionLifecycle) (*ExecutionClient, error) {
	if sender == nil || events == nil || lifecycle == nil {
		return nil, fmt.Errorf("failed to create execution websocket client: dependency=null")
	}
	return &ExecutionClient{sender: sender, events: events, lifecycle: lifecycle}, nil
}

// Subscribe observes a submitted execution without sending any chain transaction.
// Events are buffered across the acknowledgement; duplicate active subscriptions reuse the handle.
//
// Version:
//   - 2026-09-16: Added.
func (c *ExecutionClient) Subscribe(ctx context.Context, params dto.SubscribeParams) (*ExecutionSubscription, error) {
	if c == nil || ctx == nil {
		return nil, fmt.Errorf("failed to subscribe execution: dependency=null")
	}
	params = params.Normalize()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.lifecycle.mu.Lock()
	c.lifecycle.keepOpen = true
	c.lifecycle.mu.Unlock()
	r := c.events
	r.mu.Lock()
	s := r.active[params.ExecutionID]
	if s != nil {
		r.mu.Unlock()
		return s, nil
	}
	s = &ExecutionSubscription{registry: r, id: params.ExecutionID, events: make(chan dto.SubscriptionEvent, 16), errors: make(chan error, 1)}
	r.active[s.id] = s
	r.mu.Unlock()
	var ack dto.SubscribeResult
	err := c.request(ctx, rpc.MethodTradeHubExecutionSubscribe, params, &ack)
	if err == nil {
		err = ack.Validate()
	}
	if err == nil && ack.ExecutionID != params.ExecutionID {
		err = fmt.Errorf("failed to subscribe execution: execution_id=mismatch")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		r.finish(s, err)
		return nil, err
	}
	if s.closed {
		return nil, fmt.Errorf("failed to subscribe execution: observation interrupted")
	}
	s.key = ack.SubscriptionKey
	buffer := s.buffer
	s.buffer = nil
	for _, e := range buffer {
		r.deliver(s, e)
	}
	return s, nil
}

// Unsubscribe releases this session's watch without closing the physical connection.
// A subscription that already delivered its terminal event needs no further release.
//
// Version:
//   - 2026-09-16: Added.
func (c *ExecutionClient) Unsubscribe(ctx context.Context, s *ExecutionSubscription) error {
	if c == nil || ctx == nil || s == nil || s.registry != c.events {
		return fmt.Errorf("failed to unsubscribe execution: subscription=invalid")
	}
	c.opMu.Lock()
	defer c.opMu.Unlock()
	c.events.mu.Lock()
	closed := s.closed
	params := dto.UnsubscribeParams{ExecutionID: s.id, SubscriptionKey: s.key}
	c.events.mu.Unlock()
	if closed {
		return nil
	}
	var ack dto.UnsubscribeParams
	if err := c.request(ctx, rpc.MethodTradeHubExecutionUnsubscribe, params, &ack); err != nil {
		return err
	}
	if ack != params {
		return fmt.Errorf("failed to unsubscribe execution: acknowledgement=mismatch")
	}
	c.events.mu.Lock()
	defer c.events.mu.Unlock()
	c.events.finish(s, nil)
	return nil
}

// Events returns ordered observation events, closing after terminal delivery or interruption.
//
// Version:
//   - 2026-09-16: Added.
func (s *ExecutionSubscription) Events() <-chan dto.SubscriptionEvent {
	if s == nil {
		return nil
	}
	return s.events
}

// Errors returns local transport or delivery interruptions, never transaction failures.
//
// Version:
//   - 2026-09-16: Added.
func (s *ExecutionSubscription) Errors() <-chan error {
	if s == nil {
		return nil
	}
	return s.errors
}

// Reference returns the execution and server-issued watch identifiers.
//
// Version:
//   - 2026-09-16: Added.
func (s *ExecutionSubscription) Reference() dto.SubscribeResult {
	if s == nil {
		return dto.SubscribeResult{}
	}
	s.registry.mu.Lock()
	defer s.registry.mu.Unlock()
	return dto.SubscribeResult{ExecutionID: s.id, SubscriptionKey: s.key}
}

func (c *ExecutionClient) request(ctx context.Context, method rpc.Method, params, result any) error {
	encoded, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to encode execution request: %w", err)
	}
	response, err := c.sender.send(ctx, method, encoded)
	if err != nil {
		return fmt.Errorf("failed to send execution request: %w", err)
	}
	if response == nil {
		return fmt.Errorf("failed to send execution request: response=null")
	}
	if response.Error != nil {
		return fmt.Errorf("failed to send execution request: %w", response.Error)
	}
	if err := json.Unmarshal(response.Result, result); err != nil {
		return fmt.Errorf("failed to decode execution acknowledgement: %w", err)
	}
	return nil
}
func (r *executionEventRegistry) route(e dto.SubscriptionEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.active[e.ExecutionID]
	if s == nil {
		return
	}
	if s.key == "" {
		if len(s.buffer) >= 16 {
			r.finish(s, fmt.Errorf("failed to receive execution observation: event buffer full"))
			return
		}
		s.buffer = append(s.buffer, e)
		return
	}
	r.deliver(s, e)
}
func (r *executionEventRegistry) deliver(s *ExecutionSubscription, e dto.SubscriptionEvent) {
	if s.closed || e.SubscriptionKey != s.key || e.Sequence <= s.sequence {
		return
	}
	s.sequence = e.Sequence
	select {
	case s.events <- e:
	default:
		r.finish(s, fmt.Errorf("failed to receive execution observation: event buffer full"))
		return
	}
	if e.Kind == dto.ExecutionEventError || e.Snapshot != nil && e.Snapshot.Status.Terminal() {
		r.finish(s, nil)
	}
}
func (r *executionEventRegistry) finish(s *ExecutionSubscription, err error) {
	if s.closed {
		return
	}
	s.closed = true
	if r.active[s.id] == s {
		delete(r.active, s.id)
	}
	if err != nil {
		s.errors <- err
	}
	close(s.errors)
	close(s.events)
}
func (r *executionEventRegistry) interrupt() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.active {
		r.finish(s, errWebSocketConnectionClosed)
	}
}
