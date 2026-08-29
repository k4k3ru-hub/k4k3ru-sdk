package websocket

import (
	"context"
	"fmt"
	"sync"
)

type subscriptionTransport interface {
	disconnect() error
}

type subscriptionState uint8

const (
	subscriptionStateSubscribing subscriptionState = iota + 1
	subscriptionStateActive
	subscriptionStateUnsubscribing
)

type subscriptionLifecycle struct {
	mu        sync.Mutex
	transport subscriptionTransport
	states    map[string]subscriptionState
}

func newSubscriptionLifecycle(transport subscriptionTransport) (*subscriptionLifecycle, error) {
	if transport == nil {
		return nil, fmt.Errorf("failed to create websocket subscription lifecycle: transport=null")
	}
	return &subscriptionLifecycle{
		transport: transport,
		states:    make(map[string]subscriptionState),
	}, nil
}

func (l *subscriptionLifecycle) subscribe(ctx context.Context, key string, operation func(context.Context) error) error {
	if l == nil {
		return fmt.Errorf("failed to subscribe websocket lifecycle: subscription_lifecycle=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to subscribe websocket lifecycle: context=null")
	}
	if key == "" {
		return fmt.Errorf("failed to subscribe websocket lifecycle: subscription_key=empty")
	}
	if operation == nil {
		return fmt.Errorf("failed to subscribe websocket lifecycle: operation=null")
	}

	l.mu.Lock()
	if _, exists := l.states[key]; exists {
		l.mu.Unlock()
		return fmt.Errorf("failed to subscribe websocket lifecycle: subscription_key=duplicate")
	}
	l.states[key] = subscriptionStateSubscribing
	l.mu.Unlock()

	err := operation(ctx)
	l.mu.Lock()
	defer l.mu.Unlock()
	if err != nil {
		delete(l.states, key)
		if len(l.states) == 0 {
			if disconnectErr := l.transport.disconnect(); disconnectErr != nil {
				return fmt.Errorf("failed to subscribe websocket lifecycle: %w: %w", err, disconnectErr)
			}
		}
		return fmt.Errorf("failed to subscribe websocket lifecycle: %w", err)
	}
	l.states[key] = subscriptionStateActive
	return nil
}

func (l *subscriptionLifecycle) unsubscribe(ctx context.Context, key string, operation func(context.Context) error) error {
	if l == nil {
		return fmt.Errorf("failed to unsubscribe websocket lifecycle: subscription_lifecycle=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to unsubscribe websocket lifecycle: context=null")
	}
	if key == "" {
		return fmt.Errorf("failed to unsubscribe websocket lifecycle: subscription_key=empty")
	}
	if operation == nil {
		return fmt.Errorf("failed to unsubscribe websocket lifecycle: operation=null")
	}

	l.mu.Lock()
	state, exists := l.states[key]
	if !exists {
		l.mu.Unlock()
		return nil
	}
	if state != subscriptionStateActive {
		l.mu.Unlock()
		return fmt.Errorf("failed to unsubscribe websocket lifecycle: subscription_state=invalid")
	}
	l.states[key] = subscriptionStateUnsubscribing
	l.mu.Unlock()

	if err := operation(ctx); err != nil {
		l.mu.Lock()
		l.states[key] = subscriptionStateActive
		l.mu.Unlock()
		return fmt.Errorf("failed to unsubscribe websocket lifecycle: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.states, key)
	if len(l.states) != 0 {
		return nil
	}
	if err := l.transport.disconnect(); err != nil {
		return fmt.Errorf("failed to unsubscribe websocket lifecycle: %w", err)
	}
	return nil
}
