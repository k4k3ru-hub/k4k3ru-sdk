package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKAMMPoolNewPair "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool/newpair"
)

// AMMPoolNewPairClient manages Market Hub AMMPoolNewPair WebSocket subscriptions.
type AMMPoolNewPairClient struct {
	sender      jsonRPCSender
	lifecycle   *subscriptionLifecycle
	eventRouter *ammPoolNewPairEventRegistry
}

// AMMPoolNewPairSubscription represents one active AMMPoolNewPair subscription.
type AMMPoolNewPairSubscription struct {
	params   k4k3ruSDKAMMPoolNewPair.Params
	local    *ammPoolNewPairEventSubscription
	registry *ammPoolNewPairEventRegistry
}

func newAMMPoolNewPairClient(sender jsonRPCSender, lifecycle *subscriptionLifecycle, eventRouter *ammPoolNewPairEventRegistry) (*AMMPoolNewPairClient, error) {
	if sender == nil {
		return nil, fmt.Errorf("failed to create amm pool new pair websocket client: sender=null")
	}
	if lifecycle == nil {
		return nil, fmt.Errorf("failed to create amm pool new pair websocket client: subscription_lifecycle=null")
	}
	if eventRouter == nil {
		return nil, fmt.Errorf("failed to create amm pool new pair websocket client: event_registry=null")
	}
	return &AMMPoolNewPairClient{sender: sender, lifecycle: lifecycle, eventRouter: eventRouter}, nil
}

// Subscribe subscribes to one normalized Market Hub AMMPoolNewPair stream.
//
// Version:
//   - 2026-09-16: Added.
func (c *AMMPoolNewPairClient) Subscribe(ctx context.Context, params k4k3ruSDKAMMPoolNewPair.Params) (*AMMPoolNewPairSubscription, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool new pair: amm_pool_new_pair_client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool new pair: context=null")
	}
	params = params.Normalize()
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool new pair: %w", err)
	}
	local, err := c.eventRouter.register(params)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub amm pool new pair: %w", err)
	}
	if err := c.lifecycle.subscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubAMMPoolNewPairSubscribe, params, key)
	}); err != nil {
		c.eventRouter.unregister(local)
		return nil, fmt.Errorf("failed to subscribe market hub amm pool new pair: %w", err)
	}
	return &AMMPoolNewPairSubscription{params: params, local: local, registry: c.eventRouter}, nil
}

// Unsubscribe unsubscribes an active Market Hub AMMPoolNewPair stream.
//
// Version:
//   - 2026-09-16: Added.
func (c *AMMPoolNewPairClient) Unsubscribe(ctx context.Context, subscription *AMMPoolNewPairSubscription) error {
	if c == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool new pair: amm_pool_new_pair_client=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool new pair: context=null")
	}
	if subscription == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool new pair: subscription=null")
	}
	if subscription.registry != c.eventRouter || subscription.local == nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool new pair: subscription=invalid")
	}
	key := subscription.local.key
	if err := c.lifecycle.unsubscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubAMMPoolNewPairUnsubscribe, subscription.params, key)
	}); err != nil {
		return fmt.Errorf("failed to unsubscribe market hub amm pool new pair: %w", err)
	}
	c.eventRouter.unregister(subscription.local)
	return nil
}

// Events returns a latest-value event channel for the AMMPoolNewPair subscription.
//
// Version:
//   - 2026-09-16: Added.
func (s *AMMPoolNewPairSubscription) Events() <-chan k4k3ruSDKAMMPoolNewPair.Result {
	if s == nil || s.local == nil {
		return nil
	}
	return s.local.events
}

// Params returns normalized AMMPoolNewPair subscription parameters.
//
// Version:
//   - 2026-09-16: Added.
func (s *AMMPoolNewPairSubscription) Params() k4k3ruSDKAMMPoolNewPair.Params {
	if s == nil {
		return k4k3ruSDKAMMPoolNewPair.Params{}
	}
	return s.params.Normalize()
}

func (c *AMMPoolNewPairClient) sendAndValidateACK(ctx context.Context, method k4k3ruSDKJSONRPC.Method, params k4k3ruSDKAMMPoolNewPair.Params, wantKey string) error {
	encoded, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to process amm pool new pair acknowledgement: failed to encode parameters: %w", err)
	}
	response, err := c.sender.send(ctx, method, encoded)
	if err != nil {
		return fmt.Errorf("failed to process amm pool new pair acknowledgement: %w", err)
	}
	if response == nil {
		return fmt.Errorf("failed to process amm pool new pair acknowledgement: response=null")
	}
	if response.Error != nil {
		return fmt.Errorf("failed to process amm pool new pair acknowledgement: %w", response.Error)
	}
	var acknowledged k4k3ruSDKAMMPoolNewPair.Params
	if err := json.Unmarshal(response.Result, &acknowledged); err != nil {
		return fmt.Errorf("failed to process amm pool new pair acknowledgement: failed to decode result: %w", err)
	}
	key, err := acknowledged.SubscriptionKey()
	if err != nil {
		return fmt.Errorf("failed to process amm pool new pair acknowledgement: %w", err)
	}
	if key != wantKey {
		return fmt.Errorf("failed to process amm pool new pair acknowledgement: subscription_key=invalid")
	}
	return nil
}
