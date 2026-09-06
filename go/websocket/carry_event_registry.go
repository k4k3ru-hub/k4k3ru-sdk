package websocket

import (
	"fmt"
	"sync"

	k4k3ruSDKCarry "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/carry"
)

type carryEventSubscription struct {
	key    string
	events chan k4k3ruSDKCarry.Result
}
type carryEventRegistry struct {
	mu            sync.RWMutex
	subscriptions map[string]*carryEventSubscription
}

func newCarryEventRegistry() *carryEventRegistry {
	return &carryEventRegistry{subscriptions: make(map[string]*carryEventSubscription)}
}
func (r *carryEventRegistry) register(params k4k3ruSDKCarry.Params) (*carryEventSubscription, error) {
	if r == nil {
		return nil, fmt.Errorf("failed to register carry event subscription: event_registry=null")
	}
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to register carry event subscription: %w", err)
	}
	item := &carryEventSubscription{key: key, events: make(chan k4k3ruSDKCarry.Result, 1)}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.subscriptions[key]; ok {
		return nil, fmt.Errorf("failed to register carry event subscription: subscription_key=duplicate")
	}
	r.subscriptions[key] = item
	return item, nil
}
func (r *carryEventRegistry) unregister(item *carryEventSubscription) {
	if r == nil || item == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if current, ok := r.subscriptions[item.key]; ok && current == item {
		delete(r.subscriptions, item.key)
		close(item.events)
	}
}
func (r *carryEventRegistry) route(result k4k3ruSDKCarry.Result) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("failed to route carry event: event_registry=null")
	}
	params := result.Params()
	key, err := params.SubscriptionKey()
	if err != nil {
		return false, fmt.Errorf("failed to route carry event: %w", err)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.subscriptions[key]
	if !ok {
		return false, nil
	}
	select {
	case item.events <- result:
	default:
		select {
		case <-item.events:
		default:
		}
		select {
		case item.events <- result:
		default:
		}
	}
	return true, nil
}
