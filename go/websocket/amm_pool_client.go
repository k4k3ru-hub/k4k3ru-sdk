package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKAMMPool "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool"
)

// AMMPoolClient manages Market Hub AMMPool WebSocket subscriptions.
type AMMPoolClient struct {
	sender      jsonRPCSender
	lifecycle   *subscriptionLifecycle
	eventRouter *ammPoolEventRegistry
}

// AMMPoolSubscription represents one active AMMPool subscription.
type AMMPoolSubscription struct {
	params   k4k3ruSDKAMMPool.Params
	local    *ammPoolEventSubscription
	registry *ammPoolEventRegistry
}

func newAMMPoolClient(sender jsonRPCSender, lifecycle *subscriptionLifecycle, eventRouter *ammPoolEventRegistry) (*AMMPoolClient, error) {
	if sender == nil {
		return nil, fmt.Errorf("failed to create amm pool websocket client: sender=null")
	}
	if lifecycle == nil {
		return nil, fmt.Errorf("failed to create amm pool websocket client: subscription_lifecycle=null")
	}
	if eventRouter == nil {
		return nil, fmt.Errorf("failed to create amm pool websocket client: event_registry=null")
	}
	return &AMMPoolClient{sender: sender, lifecycle: lifecycle, eventRouter: eventRouter}, nil
}

// Subscribe subscribes to one normalized Market Hub AMMPool stream.
//
// Version:
//   - 2026-09-11: Added.
func (c *AMMPoolClient) Subscribe(ctx context.Context, params k4k3ruSDKAMMPool.Params) (*AMMPoolSubscription, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool: amm_pool_client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool: context=null")
	}
	params = params.Normalize()
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool: %w", err)
	}
	local, err := c.eventRouter.register(params)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool: %w", err)
	}
	if err := c.lifecycle.subscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubAMMPoolSubscribe, params, key)
	}); err != nil {
		c.eventRouter.unregister(local)
		return nil, fmt.Errorf("failed to subscribe market hub amm pool: %w", err)
	}
	return &AMMPoolSubscription{params: params, local: local, registry: c.eventRouter}, nil
}

// Unsubscribe unsubscribes an active Market Hub AMMPool stream.
//
// Version:
//   - 2026-09-11: Added.
func (c *AMMPoolClient) Unsubscribe(ctx context.Context, subscription *AMMPoolSubscription) error {
	if c == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool: amm_pool_client=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool: context=null")
	}
	if subscription == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool: subscription=null")
	}
	if subscription.registry != c.eventRouter || subscription.local == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool: subscription=invalid")
	}
	key := subscription.local.key
	if err := c.lifecycle.unsubscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubAMMPoolUnsubscribe, subscription.params, key)
	}); err != nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool: %w", err)
	}
	c.eventRouter.unregister(subscription.local)
	return nil
}

// Events returns a latest-value event channel for the AMMPool subscription.
//
// Version:
//   - 2026-09-11: Added.
func (s *AMMPoolSubscription) Events() <-chan k4k3ruSDKAMMPool.Result {
	if s == nil || s.local == nil {
		return nil
	}
	return s.local.events
}

// Params returns normalized AMMPool subscription parameters.
//
// Version:
//   - 2026-09-11: Added.
func (s *AMMPoolSubscription) Params() k4k3ruSDKAMMPool.Params {
	if s == nil {
		return k4k3ruSDKAMMPool.Params{}
	}
	return s.params.Normalize()
}

func (c *AMMPoolClient) sendAndValidateACK(ctx context.Context, method k4k3ruSDKJSONRPC.Method, params k4k3ruSDKAMMPool.Params, wantKey string) error {
	encoded, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to process amm pool acknowledgement: failed to encode parameters: %w", err)
	}
	response, err := c.sender.send(ctx, method, encoded)
	if err != nil {
		return fmt.Errorf("failed to process amm pool acknowledgement: %w", err)
	}
	if response == nil {
		return fmt.Errorf("failed to process amm pool acknowledgement: response=null")
	}
	if response.Error != nil {
		return fmt.Errorf("failed to process amm pool acknowledgement: %w", response.Error)
	}
	var acknowledged k4k3ruSDKAMMPool.Params
	if err := json.Unmarshal(response.Result, &acknowledged); err != nil {
		return fmt.Errorf("failed to process amm pool acknowledgement: failed to decode result: %w", err)
	}
	key, err := acknowledged.SubscriptionKey()
	if err != nil {
		return fmt.Errorf("failed to process amm pool acknowledgement: %w", err)
	}
	if key != wantKey {
		return fmt.Errorf("failed to process amm pool acknowledgement: subscription_key=invalid")
	}
	return nil
}
