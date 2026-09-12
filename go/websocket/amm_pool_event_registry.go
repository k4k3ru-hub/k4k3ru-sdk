package websocket

import (
	"fmt"
	"sync"

	k4k3ruSDKAMMPool "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool"
)

type ammPoolEventSubscription struct {
	key    string
	events chan k4k3ruSDKAMMPool.Result
}
type ammPoolEventRegistry struct {
	mu            sync.RWMutex
	subscriptions map[string]*ammPoolEventSubscription
}

func newAMMPoolEventRegistry() *ammPoolEventRegistry {
	return &ammPoolEventRegistry{subscriptions: make(map[string]*ammPoolEventSubscription)}
}
func (r *ammPoolEventRegistry) register(params k4k3ruSDKAMMPool.Params) (*ammPoolEventSubscription, error) {
	if r == nil {
		return nil, fmt.Errorf("failed to register amm pool event subscription: event_registry=null")
	}
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to register amm pool event subscription: %w", err)
	}
	item := &ammPoolEventSubscription{key: key, events: make(chan k4k3ruSDKAMMPool.Result, 1)}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.subscriptions[key]; ok {
		return nil, fmt.Errorf("failed to register amm pool event subscription: subscription_key=duplicate")
	}
	r.subscriptions[key] = item
	return item, nil
}
func (r *ammPoolEventRegistry) unregister(item *ammPoolEventSubscription) {
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
func (r *ammPoolEventRegistry) route(result k4k3ruSDKAMMPool.Result) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("failed to route amm pool event: event_registry=null")
	}
	params := result.Params()
	key, err := params.SubscriptionKey()
	if err != nil {
		return false, fmt.Errorf("failed to route amm pool event: %w", err)
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
