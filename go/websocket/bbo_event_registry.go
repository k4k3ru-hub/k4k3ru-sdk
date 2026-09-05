package websocket

import (
	"fmt"
	"sync"

	k4k3ruSDKJSONRPCBBO "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/bbo"
)

type bboEventSubscription struct {
	key    string
	events chan k4k3ruSDKJSONRPCBBO.Result
}

type bboEventRegistry struct {
	mu            sync.RWMutex
	subscriptions map[string]*bboEventSubscription
}

func newBBOEventRegistry() *bboEventRegistry {
	return &bboEventRegistry{subscriptions: make(map[string]*bboEventSubscription)}
}

func (r *bboEventRegistry) register(params k4k3ruSDKJSONRPCBBO.Params) (*bboEventSubscription, error) {
	if r == nil {
		return nil, fmt.Errorf("failed to register bbo event subscription: event_registry=null")
	}
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to register bbo event subscription: %w", err)
	}
	subscription := &bboEventSubscription{key: key, events: make(chan k4k3ruSDKJSONRPCBBO.Result, 1)}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.subscriptions[key]; exists {
		return nil, fmt.Errorf("failed to register bbo event subscription: subscription_key=duplicate")
	}
	r.subscriptions[key] = subscription
	return subscription, nil
}

func (r *bboEventRegistry) unregister(subscription *bboEventSubscription) {
	if r == nil || subscription == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if current, exists := r.subscriptions[subscription.key]; exists && current == subscription {
		delete(r.subscriptions, subscription.key)
		close(subscription.events)
	}
}

func (r *bboEventRegistry) route(result k4k3ruSDKJSONRPCBBO.Result) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("failed to route bbo event: event_registry=null")
	}
	params := k4k3ruSDKJSONRPCBBO.Params{AssetClass: result.AssetClass, MarketType: result.MarketType, Symbol: result.Symbol, AggregationMode: result.AggregationMode, SourceFilter: result.SourceFilter}
	key, err := params.SubscriptionKey()
	if err != nil {
		return false, fmt.Errorf("failed to route bbo event: %w", err)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	subscription, exists := r.subscriptions[key]
	if !exists {
		return false, nil
	}
	select {
	case subscription.events <- result:
	default:
		select {
		case <-subscription.events:
		default:
		}
		select {
		case subscription.events <- result:
		default:
		}
	}
	return true, nil
}
