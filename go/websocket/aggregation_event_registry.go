package websocket

import (
	"fmt"
	"sync"

	k4k3ruSDKJSONRPCAggregation "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/aggregation"
)

type aggregationEventSubscription struct {
	key    string
	events chan k4k3ruSDKJSONRPCAggregation.Result
}

type aggregationEventRegistry struct {
	mu            sync.RWMutex
	subscriptions map[string]*aggregationEventSubscription
}

func newAggregationEventRegistry() *aggregationEventRegistry {
	return &aggregationEventRegistry{subscriptions: make(map[string]*aggregationEventSubscription)}
}

func (r *aggregationEventRegistry) register(params k4k3ruSDKJSONRPCAggregation.Params) (*aggregationEventSubscription, error) {
	if r == nil {
		return nil, fmt.Errorf("failed to register aggregation event subscription: event_registry=null")
	}
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to register aggregation event subscription: %w", err)
	}
	subscription := &aggregationEventSubscription{
		key:    key,
		events: make(chan k4k3ruSDKJSONRPCAggregation.Result, 1),
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.subscriptions[key]; exists {
		return nil, fmt.Errorf("failed to register aggregation event subscription: subscription_key=duplicate")
	}
	r.subscriptions[key] = subscription
	return subscription, nil
}

func (r *aggregationEventRegistry) unregister(subscription *aggregationEventSubscription) {
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

func (r *aggregationEventRegistry) route(result k4k3ruSDKJSONRPCAggregation.Result) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("failed to route aggregation event: event_registry=null")
	}
	params := k4k3ruSDKJSONRPCAggregation.Params{
		AssetClass:      result.AssetClass,
		MarketType:      result.MarketType,
		Symbol:          result.Symbol,
		AggregationMode: result.AggregationMode,
		SourceFilter:    result.SourceFilter,
	}
	key, err := params.SubscriptionKey()
	if err != nil {
		return false, fmt.Errorf("failed to route aggregation event: %w", err)
	}
	r.mu.RLock()
	subscription, exists := r.subscriptions[key]
	if !exists {
		r.mu.RUnlock()
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
	r.mu.RUnlock()
	return true, nil
}
