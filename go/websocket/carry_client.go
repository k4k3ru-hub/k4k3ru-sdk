package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKCarry "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/carry"
)

// CarryClient manages Market Hub Carry WebSocket subscriptions.
type CarryClient struct {
	sender      jsonRPCSender
	lifecycle   *subscriptionLifecycle
	eventRouter *carryEventRegistry
}

// CarrySubscription represents one active Carry subscription.
type CarrySubscription struct {
	params   k4k3ruSDKCarry.Params
	local    *carryEventSubscription
	registry *carryEventRegistry
}

func newCarryClient(sender jsonRPCSender, lifecycle *subscriptionLifecycle, eventRouter *carryEventRegistry) (*CarryClient, error) {
	if sender == nil {
		return nil, fmt.Errorf("failed to create carry websocket client: sender=null")
	}
	if lifecycle == nil {
		return nil, fmt.Errorf("failed to create carry websocket client: subscription_lifecycle=null")
	}
	if eventRouter == nil {
		return nil, fmt.Errorf("failed to create carry websocket client: event_registry=null")
	}
	return &CarryClient{sender: sender, lifecycle: lifecycle, eventRouter: eventRouter}, nil
}

// Subscribe subscribes to one normalized Market Hub Carry stream.
//
// Version:
//   - 2026-09-06: Added.
func (c *CarryClient) Subscribe(ctx context.Context, params k4k3ruSDKCarry.Params) (*CarrySubscription, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to subscribe market hub carry: carry_client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to subscribe market hub carry: context=null")
	}
	params = params.Normalize()
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub carry: %w", err)
	}
	local, err := c.eventRouter.register(params)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub carry: %w", err)
	}
	if err := c.lifecycle.subscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubCarrySubscribe, params, key)
	}); err != nil {
		c.eventRouter.unregister(local)
		return nil, fmt.Errorf("failed to subscribe market hub carry: %w", err)
	}
	return &CarrySubscription{params: params, local: local, registry: c.eventRouter}, nil
}

// Unsubscribe unsubscribes an active Market Hub Carry stream.
//
// Version:
//   - 2026-09-06: Added.
func (c *CarryClient) Unsubscribe(ctx context.Context, subscription *CarrySubscription) error {
	if c == nil {
		return fmt.Errorf("failed to unsubscribe market hub carry: carry_client=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to unsubscribe market hub carry: context=null")
	}
	if subscription == nil {
		return fmt.Errorf("failed to unsubscribe market hub carry: subscription=null")
	}
	if subscription.registry != c.eventRouter || subscription.local == nil {
		return fmt.Errorf("failed to unsubscribe market hub carry: subscription=invalid")
	}
	key := subscription.local.key
	if err := c.lifecycle.unsubscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubCarryUnsubscribe, subscription.params, key)
	}); err != nil {
		return fmt.Errorf("failed to unsubscribe market hub carry: %w", err)
	}
	c.eventRouter.unregister(subscription.local)
	return nil
}

// Events returns a latest-value event channel for the Carry subscription.
//
// Version:
//   - 2026-09-06: Added.
func (s *CarrySubscription) Events() <-chan k4k3ruSDKCarry.Result {
	if s == nil || s.local == nil {
		return nil
	}
	return s.local.events
}

// Params returns normalized Carry subscription parameters.
//
// Version:
//   - 2026-09-06: Added.
func (s *CarrySubscription) Params() k4k3ruSDKCarry.Params {
	if s == nil {
		return k4k3ruSDKCarry.Params{}
	}
	return s.params.Normalize()
}

func (c *CarryClient) sendAndValidateACK(ctx context.Context, method k4k3ruSDKJSONRPC.Method, params k4k3ruSDKCarry.Params, wantKey string) error {
	encoded, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to process carry acknowledgement: failed to encode parameters: %w", err)
	}
	response, err := c.sender.send(ctx, method, encoded)
	if err != nil {
		return fmt.Errorf("failed to process carry acknowledgement: %w", err)
	}
	if response == nil {
		return fmt.Errorf("failed to process carry acknowledgement: response=null")
	}
	if response.Error != nil {
		return fmt.Errorf("failed to process carry acknowledgement: %w", response.Error)
	}
	var acknowledged k4k3ruSDKCarry.Params
	if err := json.Unmarshal(response.Result, &acknowledged); err != nil {
		return fmt.Errorf("failed to process carry acknowledgement: failed to decode result: %w", err)
	}
	key, err := acknowledged.SubscriptionKey()
	if err != nil {
		return fmt.Errorf("failed to process carry acknowledgement: %w", err)
	}
	if key != wantKey {
		return fmt.Errorf("failed to process carry acknowledgement: subscription_key=invalid")
	}
	return nil
}
