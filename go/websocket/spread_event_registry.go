package websocket

import (
	"fmt"
	"sync"

	k4k3ruSDKSpread "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/spread"
)

type spreadEventSubscription struct {
	key    string
	events chan k4k3ruSDKSpread.Result
}
type spreadEventRegistry struct {
	mu            sync.RWMutex
	subscriptions map[string]*spreadEventSubscription
}

func newSpreadEventRegistry() *spreadEventRegistry {
	return &spreadEventRegistry{subscriptions: make(map[string]*spreadEventSubscription)}
}
func (r *spreadEventRegistry) register(params k4k3ruSDKSpread.Params) (*spreadEventSubscription, error) {
	if r == nil {
		return nil, fmt.Errorf("failed to register spread event subscription: event_registry=null")
	}
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to register spread event subscription: %w", err)
	}
	item := &spreadEventSubscription{key: key, events: make(chan k4k3ruSDKSpread.Result, 1)}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.subscriptions[key]; ok {
		return nil, fmt.Errorf("failed to register spread event subscription: subscription_key=duplicate")
	}
	r.subscriptions[key] = item
	return item, nil
}
func (r *spreadEventRegistry) unregister(item *spreadEventSubscription) {
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
func (r *spreadEventRegistry) route(result k4k3ruSDKSpread.Result) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("failed to route spread event: event_registry=null")
	}
	params := k4k3ruSDKSpread.Params{AssetClass: result.AssetClass, Symbol: result.Symbol, BaseAsset: result.BaseAsset, Quantity: result.Quantity, MinimumGrossSpreadBps: result.MinimumGrossSpreadBps, RouteFamilies: result.RouteFamilies, SourceFilter: result.SourceFilter}
	key, err := params.SubscriptionKey()
	if err != nil {
		return false, fmt.Errorf("failed to route spread event: %w", err)
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
