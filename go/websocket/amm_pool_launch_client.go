package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKAMMPoolLaunch "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool/launch"
)

// AMMPoolLaunchClient manages Market Hub AMMPoolLaunch WebSocket subscriptions.
type AMMPoolLaunchClient struct {
	sender      jsonRPCSender
	lifecycle   *subscriptionLifecycle
	eventRouter *ammPoolLaunchEventRegistry
}

// AMMPoolLaunchSubscription represents one active AMMPoolLaunch subscription.
type AMMPoolLaunchSubscription struct {
	params   k4k3ruSDKAMMPoolLaunch.Params
	local    *ammPoolLaunchEventSubscription
	registry *ammPoolLaunchEventRegistry
}

func newAMMPoolLaunchClient(sender jsonRPCSender, lifecycle *subscriptionLifecycle, eventRouter *ammPoolLaunchEventRegistry) (*AMMPoolLaunchClient, error) {
	if sender == nil {
		return nil, fmt.Errorf("failed to create amm pool launch websocket client: sender=null")
	}
	if lifecycle == nil {
		return nil, fmt.Errorf("failed to create amm pool launch websocket client: subscription_lifecycle=null")
	}
	if eventRouter == nil {
		return nil, fmt.Errorf("failed to create amm pool launch websocket client: event_registry=null")
	}
	return &AMMPoolLaunchClient{sender: sender, lifecycle: lifecycle, eventRouter: eventRouter}, nil
}

// Subscribe subscribes to one normalized Market Hub AMMPoolLaunch stream.
//
// Version:
//   - 2026-09-15: Added.
func (c *AMMPoolLaunchClient) Subscribe(ctx context.Context, params k4k3ruSDKAMMPoolLaunch.Params) (*AMMPoolLaunchSubscription, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool launch: amm_pool_launch_client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool launch: context=null")
	}
	params = params.Normalize()
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool launch: %w", err)
	}
	local, err := c.eventRouter.register(params)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool launch: %w", err)
	}
	if err := c.lifecycle.subscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubAMMPoolLaunchSubscribe, params, key)
	}); err != nil {
		c.eventRouter.unregister(local)
		return nil, fmt.Errorf("failed to subscribe market hub amm pool launch: %w", err)
	}
	return &AMMPoolLaunchSubscription{params: params, local: local, registry: c.eventRouter}, nil
}

// Unsubscribe unsubscribes an active Market Hub AMMPoolLaunch stream.
//
// Version:
//   - 2026-09-15: Added.
func (c *AMMPoolLaunchClient) Unsubscribe(ctx context.Context, subscription *AMMPoolLaunchSubscription) error {
	if c == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool launch: amm_pool_launch_client=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool launch: context=null")
	}
	if subscription == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool launch: subscription=null")
	}
	if subscription.registry != c.eventRouter || subscription.local == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool launch: subscription=invalid")
	}
	key := subscription.local.key
	if err := c.lifecycle.unsubscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubAMMPoolLaunchUnsubscribe, subscription.params, key)
	}); err != nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool launch: %w", err)
	}
	c.eventRouter.unregister(subscription.local)
	return nil
}

// Events returns a latest-value event channel for the AMMPoolLaunch subscription.
//
// Version:
//   - 2026-09-15: Added.
func (s *AMMPoolLaunchSubscription) Events() <-chan k4k3ruSDKAMMPoolLaunch.Result {
	if s == nil || s.local == nil {
		return nil
	}
	return s.local.events
}

// Params returns normalized AMMPoolLaunch subscription parameters.
//
// Version:
//   - 2026-09-15: Added.
func (s *AMMPoolLaunchSubscription) Params() k4k3ruSDKAMMPoolLaunch.Params {
	if s == nil {
		return k4k3ruSDKAMMPoolLaunch.Params{}
	}
	return s.params.Normalize()
}

func (c *AMMPoolLaunchClient) sendAndValidateACK(ctx context.Context, method k4k3ruSDKJSONRPC.Method, params k4k3ruSDKAMMPoolLaunch.Params, wantKey string) error {
	encoded, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to process amm pool launch acknowledgement: failed to encode parameters: %w", err)
	}
	response, err := c.sender.send(ctx, method, encoded)
	if err != nil {
		return fmt.Errorf("failed to process amm pool launch acknowledgement: %w", err)
	}
	if response == nil {
		return fmt.Errorf("failed to process amm pool launch acknowledgement: response=null")
	}
	if response.Error != nil {
		return fmt.Errorf("failed to process amm pool launch acknowledgement: %w", response.Error)
	}
	var acknowledged k4k3ruSDKAMMPoolLaunch.Params
	if err := json.Unmarshal(response.Result, &acknowledged); err != nil {
		return fmt.Errorf("failed to process amm pool launch acknowledgement: failed to decode result: %w", err)
	}
	key, err := acknowledged.SubscriptionKey()
	if err != nil {
		return fmt.Errorf("failed to process amm pool launch acknowledgement: %w", err)
	}
	if key != wantKey {
		return fmt.Errorf("failed to process amm pool launch acknowledgement: subscription_key=invalid")
	}
	return nil
}
