package websocket

import (
	"fmt"
	"sync"

	k4k3ruSDKAMMPoolLaunch "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool/launch"
)

type ammPoolLaunchEventSubscription struct {
	key    string
	events chan k4k3ruSDKAMMPoolLaunch.Result
}
type ammPoolLaunchEventRegistry struct {
	mu            sync.RWMutex
	subscriptions map[string]*ammPoolLaunchEventSubscription
}

func newAMMPoolLaunchEventRegistry() *ammPoolLaunchEventRegistry {
	return &ammPoolLaunchEventRegistry{subscriptions: make(map[string]*ammPoolLaunchEventSubscription)}
}
func (r *ammPoolLaunchEventRegistry) register(params k4k3ruSDKAMMPoolLaunch.Params) (*ammPoolLaunchEventSubscription, error) {
	if r == nil {
		return nil, fmt.Errorf("failed to register amm pool launch event subscription: event_registry=null")
	}
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to register amm pool launch event subscription: %w", err)
	}
	item := &ammPoolLaunchEventSubscription{key: key, events: make(chan k4k3ruSDKAMMPoolLaunch.Result, 1)}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.subscriptions[key]; ok {
		return nil, fmt.Errorf("failed to register amm pool launch event subscription: subscription_key=duplicate")
	}
	r.subscriptions[key] = item
	return item, nil
}
func (r *ammPoolLaunchEventRegistry) unregister(item *ammPoolLaunchEventSubscription) {
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
func (r *ammPoolLaunchEventRegistry) route(result k4k3ruSDKAMMPoolLaunch.Result) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("failed to route amm pool launch event: event_registry=null")
	}
	params := result.Params()
	key, err := params.SubscriptionKey()
	if err != nil {
		return false, fmt.Errorf("failed to route amm pool launch event: %w", err)
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
