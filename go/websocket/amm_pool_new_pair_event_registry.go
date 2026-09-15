package websocket

import (
	"fmt"
	"sync"

	k4k3ruSDKAMMPoolNewPair "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool/newpair"
)

type ammPoolNewPairEventSubscription struct {
	key    string
	events chan k4k3ruSDKAMMPoolNewPair.Result
}
type ammPoolNewPairEventRegistry struct {
	mu            sync.RWMutex
	subscriptions map[string]*ammPoolNewPairEventSubscription
}

func newAMMPoolNewPairEventRegistry() *ammPoolNewPairEventRegistry {
	return &ammPoolNewPairEventRegistry{subscriptions: make(map[string]*ammPoolNewPairEventSubscription)}
}
func (r *ammPoolNewPairEventRegistry) register(params k4k3ruSDKAMMPoolNewPair.Params) (*ammPoolNewPairEventSubscription, error) {
	if r == nil {
		return nil, fmt.Errorf("failed to register amm pool new pair event subscription: event_registry=null")
	}
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to register amm pool new pair event subscription: %w", err)
	}
	item := &ammPoolNewPairEventSubscription{key: key, events: make(chan k4k3ruSDKAMMPoolNewPair.Result, 1)}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.subscriptions[key]; ok {
		return nil, fmt.Errorf("failed to register amm pool new pair event subscription: subscription_key=duplicate")
	}
	r.subscriptions[key] = item
	return item, nil
}
func (r *ammPoolNewPairEventRegistry) unregister(item *ammPoolNewPairEventSubscription) {
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
func (r *ammPoolNewPairEventRegistry) route(result k4k3ruSDKAMMPoolNewPair.Result) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("failed to route amm pool new pair event: event_registry=null")
	}
	params := result.Params()
	key, err := params.SubscriptionKey()
	if err != nil {
		return false, fmt.Errorf("failed to route amm pool new pair event: %w", err)
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
