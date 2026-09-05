package websocket

import (
	"fmt"
	"sync"

	k4k3ruSDKJSONRPCOrderBook "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/orderbook"
)

type orderBookEventSubscription struct {
	key    string
	events chan k4k3ruSDKJSONRPCOrderBook.Result
}

type orderBookEventRegistry struct {
	mu            sync.RWMutex
	subscriptions map[string]*orderBookEventSubscription
}

func newOrderBookEventRegistry() *orderBookEventRegistry {
	return &orderBookEventRegistry{subscriptions: make(map[string]*orderBookEventSubscription)}
}

func (r *orderBookEventRegistry) register(params k4k3ruSDKJSONRPCOrderBook.Params) (*orderBookEventSubscription, error) {
	if r == nil {
		return nil, fmt.Errorf("failed to register order book event subscription: event_registry=null")
	}
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to register order book event subscription: %w", err)
	}
	subscription := &orderBookEventSubscription{key: key, events: make(chan k4k3ruSDKJSONRPCOrderBook.Result, 1)}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.subscriptions[key]; exists {
		return nil, fmt.Errorf("failed to register order book event subscription: subscription_key=duplicate")
	}
	r.subscriptions[key] = subscription
	return subscription, nil
}

func (r *orderBookEventRegistry) unregister(subscription *orderBookEventSubscription) {
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

func (r *orderBookEventRegistry) route(result k4k3ruSDKJSONRPCOrderBook.Result) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("failed to route order book event: event_registry=null")
	}
	params := k4k3ruSDKJSONRPCOrderBook.Params{
		AssetClass: result.AssetClass, MarketType: result.MarketType, Symbol: result.Symbol,
		Depth: result.Depth, SourceFilter: result.SourceFilter,
	}
	key, err := params.SubscriptionKey()
	if err != nil {
		return false, fmt.Errorf("failed to route order book event: %w", err)
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
