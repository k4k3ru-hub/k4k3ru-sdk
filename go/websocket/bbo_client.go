package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKJSONRPCBBO "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/bbo"
)

type jsonRPCSender interface {
	send(context.Context, k4k3ruSDKJSONRPC.Method, json.RawMessage) (*k4k3ruSDKJSONRPC.Response, error)
}

// BBOClient manages Market Hub BBO WebSocket subscriptions.
type BBOClient struct {
	sender      jsonRPCSender
	lifecycle   *subscriptionLifecycle
	eventRouter *bboEventRegistry
}

// BBOSubscription represents one active BBO subscription.
type BBOSubscription struct {
	params   k4k3ruSDKJSONRPCBBO.Params
	local    *bboEventSubscription
	registry *bboEventRegistry
}

func newBBOClient(sender jsonRPCSender, lifecycle *subscriptionLifecycle, eventRouter *bboEventRegistry) (*BBOClient, error) {
	if sender == nil {
		return nil, fmt.Errorf("failed to create bbo websocket client: sender=null")
	}
	if lifecycle == nil {
		return nil, fmt.Errorf("failed to create bbo websocket client: subscription_lifecycle=null")
	}
	if eventRouter == nil {
		return nil, fmt.Errorf("failed to create bbo websocket client: event_registry=null")
	}
	return &BBOClient{sender: sender, lifecycle: lifecycle, eventRouter: eventRouter}, nil
}

// Subscribe subscribes to one normalized Market Hub BBO stream.
//
// Version:
//   - 2026-09-05: Added.
func (c *BBOClient) Subscribe(ctx context.Context, params k4k3ruSDKJSONRPCBBO.Params) (*BBOSubscription, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to subscribe market hub bbo: bbo_client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to subscribe market hub bbo: context=null")
	}
	params = params.Normalize()
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub bbo: %w", err)
	}
	local, err := c.eventRouter.register(params)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub bbo: %w", err)
	}
	if err := c.lifecycle.subscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubBBOSubscribe, params, key)
	}); err != nil {
		c.eventRouter.unregister(local)
		return nil, fmt.Errorf("failed to subscribe market hub bbo: %w", err)
	}
	return &BBOSubscription{params: params, local: local, registry: c.eventRouter}, nil
}

// Unsubscribe unsubscribes an active Market Hub BBO stream.
//
// Version:
//   - 2026-09-05: Added.
func (c *BBOClient) Unsubscribe(ctx context.Context, subscription *BBOSubscription) error {
	if c == nil {
		return fmt.Errorf("failed to unsubscribe market hub bbo: bbo_client=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to unsubscribe market hub bbo: context=null")
	}
	if subscription == nil {
		return fmt.Errorf("failed to unsubscribe market hub bbo: subscription=null")
	}
	if subscription.registry != c.eventRouter || subscription.local == nil {
		return fmt.Errorf("failed to unsubscribe market hub bbo: subscription=invalid")
	}
	key := subscription.local.key
	if err := c.lifecycle.unsubscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubBBOUnsubscribe, subscription.params, key)
	}); err != nil {
		return fmt.Errorf("failed to unsubscribe market hub bbo: %w", err)
	}
	c.eventRouter.unregister(subscription.local)
	return nil
}

// Events returns a latest-value event channel for the BBO subscription.
//
// Version:
//   - 2026-09-05: Added.
func (s *BBOSubscription) Events() <-chan k4k3ruSDKJSONRPCBBO.Result {
	if s == nil || s.local == nil {
		return nil
	}
	return s.local.events
}

// Params returns a normalized copy of the BBO subscription parameters.
//
// Version:
//   - 2026-09-05: Added.
func (s *BBOSubscription) Params() k4k3ruSDKJSONRPCBBO.Params {
	if s == nil {
		return k4k3ruSDKJSONRPCBBO.Params{}
	}
	return s.params.Normalize()
}

func (c *BBOClient) sendAndValidateACK(ctx context.Context, method k4k3ruSDKJSONRPC.Method, params k4k3ruSDKJSONRPCBBO.Params, wantKey string) error {
	encodedParams, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to process bbo acknowledgement: failed to encode parameters: %w", err)
	}
	response, err := c.sender.send(ctx, method, encodedParams)
	if err != nil {
		return fmt.Errorf("failed to process bbo acknowledgement: %w", err)
	}
	if response == nil {
		return fmt.Errorf("failed to process bbo acknowledgement: response=null")
	}
	if response.Error != nil {
		return fmt.Errorf("failed to process bbo acknowledgement: %w", response.Error)
	}
	var acknowledged k4k3ruSDKJSONRPCBBO.Params
	if err := json.Unmarshal(response.Result, &acknowledged); err != nil {
		return fmt.Errorf("failed to process bbo acknowledgement: failed to decode result: %w", err)
	}
	acknowledgedKey, err := acknowledged.SubscriptionKey()
	if err != nil {
		return fmt.Errorf("failed to process bbo acknowledgement: %w", err)
	}
	if acknowledgedKey != wantKey {
		return fmt.Errorf("failed to process bbo acknowledgement: subscription_key=invalid")
	}
	return nil
}
