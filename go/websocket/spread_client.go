package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKSpread "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/spread"
)

// SpreadClient manages Market Hub Spread WebSocket subscriptions.
type SpreadClient struct {
	sender      jsonRPCSender
	lifecycle   *subscriptionLifecycle
	eventRouter *spreadEventRegistry
}

// SpreadSubscription represents one active Spread subscription.
type SpreadSubscription struct {
	params   k4k3ruSDKSpread.Params
	local    *spreadEventSubscription
	registry *spreadEventRegistry
}

func newSpreadClient(sender jsonRPCSender, lifecycle *subscriptionLifecycle, eventRouter *spreadEventRegistry) (*SpreadClient, error) {
	if sender == nil {
		return nil, fmt.Errorf("failed to create spread websocket client: sender=null")
	}
	if lifecycle == nil {
		return nil, fmt.Errorf("failed to create spread websocket client: subscription_lifecycle=null")
	}
	if eventRouter == nil {
		return nil, fmt.Errorf("failed to create spread websocket client: event_registry=null")
	}
	return &SpreadClient{sender: sender, lifecycle: lifecycle, eventRouter: eventRouter}, nil
}

// Subscribe subscribes to one normalized Market Hub Spread stream.
//
// Version:
//   - 2026-09-05: Added.
func (c *SpreadClient) Subscribe(ctx context.Context, params k4k3ruSDKSpread.Params) (*SpreadSubscription, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to subscribe market hub spread: spread_client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to subscribe market hub spread: context=null")
	}
	params = params.Normalize()
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub spread: %w", err)
	}
	local, err := c.eventRouter.register(params)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub spread: %w", err)
	}
	if err := c.lifecycle.subscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubSpreadSubscribe, params, key)
	}); err != nil {
		c.eventRouter.unregister(local)
		return nil, fmt.Errorf("failed to subscribe market hub spread: %w", err)
	}
	return &SpreadSubscription{params: params, local: local, registry: c.eventRouter}, nil
}

// Unsubscribe unsubscribes an active Market Hub Spread stream.
//
// Version:
//   - 2026-09-05: Added.
func (c *SpreadClient) Unsubscribe(ctx context.Context, subscription *SpreadSubscription) error {
	if c == nil {
		return fmt.Errorf("failed to unsubscribe market hub spread: spread_client=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to unsubscribe market hub spread: context=null")
	}
	if subscription == nil {
		return fmt.Errorf("failed to unsubscribe market hub spread: subscription=null")
	}
	if subscription.registry != c.eventRouter || subscription.local == nil {
		return fmt.Errorf("failed to unsubscribe market hub spread: subscription=invalid")
	}
	key := subscription.local.key
	if err := c.lifecycle.unsubscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubSpreadUnsubscribe, subscription.params, key)
	}); err != nil {
		return fmt.Errorf("failed to unsubscribe market hub spread: %w", err)
	}
	c.eventRouter.unregister(subscription.local)
	return nil
}

// Events returns a latest-value event channel for the Spread subscription.
//
// Version:
//   - 2026-09-05: Added.
func (s *SpreadSubscription) Events() <-chan k4k3ruSDKSpread.Result {
	if s == nil || s.local == nil {
		return nil
	}
	return s.local.events
}

// Params returns normalized Spread subscription parameters.
//
// Version:
//   - 2026-09-05: Added.
func (s *SpreadSubscription) Params() k4k3ruSDKSpread.Params {
	if s == nil {
		return k4k3ruSDKSpread.Params{}
	}
	return s.params.Normalize()
}

func (c *SpreadClient) sendAndValidateACK(ctx context.Context, method k4k3ruSDKJSONRPC.Method, params k4k3ruSDKSpread.Params, wantKey string) error {
	encoded, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to process spread acknowledgement: failed to encode parameters: %w", err)
	}
	response, err := c.sender.send(ctx, method, encoded)
	if err != nil {
		return fmt.Errorf("failed to process spread acknowledgement: %w", err)
	}
	if response == nil {
		return fmt.Errorf("failed to process spread acknowledgement: response=null")
	}
	if response.Error != nil {
		return fmt.Errorf("failed to process spread acknowledgement: %w", response.Error)
	}
	var acknowledged k4k3ruSDKSpread.Params
	if err := json.Unmarshal(response.Result, &acknowledged); err != nil {
		return fmt.Errorf("failed to process spread acknowledgement: failed to decode result: %w", err)
	}
	key, err := acknowledged.SubscriptionKey()
	if err != nil {
		return fmt.Errorf("failed to process spread acknowledgement: %w", err)
	}
	if key != wantKey {
		return fmt.Errorf("failed to process spread acknowledgement: subscription_key=invalid")
	}
	return nil
}
