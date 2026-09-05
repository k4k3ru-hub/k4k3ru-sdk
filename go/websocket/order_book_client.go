package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKJSONRPCOrderBook "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/orderbook"
)

// OrderBookClient manages Market Hub OrderBook WebSocket subscriptions.
type OrderBookClient struct {
	sender      jsonRPCSender
	lifecycle   *subscriptionLifecycle
	eventRouter *orderBookEventRegistry
}

// OrderBookSubscription represents one active OrderBook subscription.
type OrderBookSubscription struct {
	params   k4k3ruSDKJSONRPCOrderBook.Params
	local    *orderBookEventSubscription
	registry *orderBookEventRegistry
}

func newOrderBookClient(sender jsonRPCSender, lifecycle *subscriptionLifecycle, eventRouter *orderBookEventRegistry) (*OrderBookClient, error) {
	if sender == nil {
		return nil, fmt.Errorf("failed to create order book websocket client: sender=null")
	}
	if lifecycle == nil {
		return nil, fmt.Errorf("failed to create order book websocket client: subscription_lifecycle=null")
	}
	if eventRouter == nil {
		return nil, fmt.Errorf("failed to create order book websocket client: event_registry=null")
	}
	return &OrderBookClient{sender: sender, lifecycle: lifecycle, eventRouter: eventRouter}, nil
}

// Subscribe subscribes to one normalized Market Hub OrderBook stream.
//
// Version:
//   - 2026-09-05: Added.
func (c *OrderBookClient) Subscribe(ctx context.Context, params k4k3ruSDKJSONRPCOrderBook.Params) (*OrderBookSubscription, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to subscribe market hub order book: order_book_client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to subscribe market hub order book: context=null")
	}
	params = params.Normalize()
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub order book: %w", err)
	}
	local, err := c.eventRouter.register(params)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub order book: %w", err)
	}
	if err := c.lifecycle.subscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubOrderBookSubscribe, params, key)
	}); err != nil {
		c.eventRouter.unregister(local)
		return nil, fmt.Errorf("failed to subscribe market hub order book: %w", err)
	}
	return &OrderBookSubscription{params: params, local: local, registry: c.eventRouter}, nil
}

// Unsubscribe unsubscribes an active Market Hub OrderBook stream.
//
// Version:
//   - 2026-09-05: Added.
func (c *OrderBookClient) Unsubscribe(ctx context.Context, subscription *OrderBookSubscription) error {
	if c == nil {
		return fmt.Errorf("failed to unsubscribe market hub order book: order_book_client=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to unsubscribe market hub order book: context=null")
	}
	if subscription == nil {
		return fmt.Errorf("failed to unsubscribe market hub order book: subscription=null")
	}
	if subscription.registry != c.eventRouter || subscription.local == nil {
		return fmt.Errorf("failed to unsubscribe market hub order book: subscription=invalid")
	}
	key := subscription.local.key
	if err := c.lifecycle.unsubscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubOrderBookUnsubscribe, subscription.params, key)
	}); err != nil {
		return fmt.Errorf("failed to unsubscribe market hub order book: %w", err)
	}
	c.eventRouter.unregister(subscription.local)
	return nil
}

// Events returns a latest-value event channel for the OrderBook subscription.
//
// Version:
//   - 2026-09-05: Added.
func (s *OrderBookSubscription) Events() <-chan k4k3ruSDKJSONRPCOrderBook.Result {
	if s == nil || s.local == nil {
		return nil
	}
	return s.local.events
}

// Params returns a normalized copy of the OrderBook subscription parameters.
//
// Version:
//   - 2026-09-05: Added.
func (s *OrderBookSubscription) Params() k4k3ruSDKJSONRPCOrderBook.Params {
	if s == nil {
		return k4k3ruSDKJSONRPCOrderBook.Params{}
	}
	return s.params.Normalize()
}

func (c *OrderBookClient) sendAndValidateACK(ctx context.Context, method k4k3ruSDKJSONRPC.Method, params k4k3ruSDKJSONRPCOrderBook.Params, wantKey string) error {
	encodedParams, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to process order book acknowledgement: failed to encode parameters: %w", err)
	}
	response, err := c.sender.send(ctx, method, encodedParams)
	if err != nil {
		return fmt.Errorf("failed to process order book acknowledgement: %w", err)
	}
	if response == nil {
		return fmt.Errorf("failed to process order book acknowledgement: response=null")
	}
	if response.Error != nil {
		return fmt.Errorf("failed to process order book acknowledgement: %w", response.Error)
	}
	var acknowledged k4k3ruSDKJSONRPCOrderBook.Params
	if err := json.Unmarshal(response.Result, &acknowledged); err != nil {
		return fmt.Errorf("failed to process order book acknowledgement: failed to decode result: %w", err)
	}
	acknowledgedKey, err := acknowledged.SubscriptionKey()
	if err != nil {
		return fmt.Errorf("failed to process order book acknowledgement: %w", err)
	}
	if acknowledgedKey != wantKey {
		return fmt.Errorf("failed to process order book acknowledgement: subscription_key=invalid")
	}
	return nil
}
