package websocket

import (
	"context"
	"encoding/json"
	"fmt"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKJSONRPCAggregation "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/aggregation"
)

type jsonRPCSender interface {
	send(context.Context, k4k3ruSDKJSONRPC.Method, json.RawMessage) (*k4k3ruSDKJSONRPC.Response, error)
}

// AggregationClient manages Market Hub aggregation WebSocket subscriptions.
type AggregationClient struct {
	sender      jsonRPCSender
	lifecycle   *subscriptionLifecycle
	eventRouter *aggregationEventRegistry
}

// AggregationSubscription represents one active aggregation subscription.
type AggregationSubscription struct {
	params   k4k3ruSDKJSONRPCAggregation.Params
	local    *aggregationEventSubscription
	registry *aggregationEventRegistry
}

func newAggregationClient(sender jsonRPCSender, lifecycle *subscriptionLifecycle, eventRouter *aggregationEventRegistry) (*AggregationClient, error) {
	if sender == nil {
		return nil, fmt.Errorf("failed to create aggregation websocket client: sender=null")
	}
	if lifecycle == nil {
		return nil, fmt.Errorf("failed to create aggregation websocket client: subscription_lifecycle=null")
	}
	if eventRouter == nil {
		return nil, fmt.Errorf("failed to create aggregation websocket client: event_registry=null")
	}
	return &AggregationClient{sender: sender, lifecycle: lifecycle, eventRouter: eventRouter}, nil
}

// Subscribe subscribes to one normalized Market Hub aggregation stream.
//
// Parameters:
//   - ctx: request and acknowledgement context.
//   - params: aggregation stream parameters.
//
// Returns:
//   - Active aggregation subscription.
//   - Validation, credential, signing, transport, or JSON-RPC error.
//
// Version:
//   - 2026-08-29: Added.
func (c *AggregationClient) Subscribe(ctx context.Context, params k4k3ruSDKJSONRPCAggregation.Params) (*AggregationSubscription, error) {
	if c == nil {
		return nil, fmt.Errorf("failed to subscribe market hub aggregation: aggregation_client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to subscribe market hub aggregation: context=null")
	}
	params = params.Normalize()
	key, err := params.SubscriptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub aggregation: %w", err)
	}
	local, err := c.eventRouter.register(params)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe market hub aggregation: %w", err)
	}
	if err := c.lifecycle.subscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubAggregationSubscribe, params, key)
	}); err != nil {
		c.eventRouter.unregister(local)
		return nil, fmt.Errorf("failed to subscribe market hub aggregation: %w", err)
	}
	return &AggregationSubscription{params: params, local: local, registry: c.eventRouter}, nil
}

// Unsubscribe unsubscribes an active Market Hub aggregation stream.
//
// Parameters:
//   - ctx: request and acknowledgement context.
//   - subscription: subscription returned by Subscribe.
//
// Returns:
//   - Validation, credential, signing, transport, or JSON-RPC error.
//
// Version:
//   - 2026-08-29: Added.
func (c *AggregationClient) Unsubscribe(ctx context.Context, subscription *AggregationSubscription) error {
	if c == nil {
		return fmt.Errorf("failed to unsubscribe market hub aggregation: aggregation_client=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to unsubscribe market hub aggregation: context=null")
	}
	if subscription == nil {
		return fmt.Errorf("failed to unsubscribe market hub aggregation: subscription=null")
	}
	if subscription.registry != c.eventRouter || subscription.local == nil {
		return fmt.Errorf("failed to unsubscribe market hub aggregation: subscription=invalid")
	}
	key := subscription.local.key
	if err := c.lifecycle.unsubscribe(ctx, key, func(operationCtx context.Context) error {
		return c.sendAndValidateACK(operationCtx, k4k3ruSDKJSONRPC.MethodMarketHubAggregationUnsubscribe, subscription.params, key)
	}); err != nil {
		return fmt.Errorf("failed to unsubscribe market hub aggregation: %w", err)
	}
	c.eventRouter.unregister(subscription.local)
	return nil
}

// Events returns a latest-value event channel for the aggregation subscription.
//
// Returns:
//   - Read-only aggregation event channel, or nil for an invalid subscription.
//
// Version:
//   - 2026-08-29: Added.
func (s *AggregationSubscription) Events() <-chan k4k3ruSDKJSONRPCAggregation.Result {
	if s == nil || s.local == nil {
		return nil
	}
	return s.local.events
}

// Params returns a normalized copy of the aggregation subscription parameters.
//
// Returns:
//   - Normalized aggregation parameters.
//
// Version:
//   - 2026-08-29: Added.
func (s *AggregationSubscription) Params() k4k3ruSDKJSONRPCAggregation.Params {
	if s == nil {
		return k4k3ruSDKJSONRPCAggregation.Params{}
	}
	return s.params.Normalize()
}

func (c *AggregationClient) sendAndValidateACK(ctx context.Context, method k4k3ruSDKJSONRPC.Method, params k4k3ruSDKJSONRPCAggregation.Params, wantKey string) error {
	encodedParams, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to process aggregation acknowledgement: failed to encode parameters: %w", err)
	}
	response, err := c.sender.send(ctx, method, encodedParams)
	if err != nil {
		return fmt.Errorf("failed to process aggregation acknowledgement: %w", err)
	}
	if response == nil {
		return fmt.Errorf("failed to process aggregation acknowledgement: response=null")
	}
	if response.Error != nil {
		return fmt.Errorf("failed to process aggregation acknowledgement: %w", response.Error)
	}
	var acknowledged k4k3ruSDKJSONRPCAggregation.Params
	if err := json.Unmarshal(response.Result, &acknowledged); err != nil {
		return fmt.Errorf("failed to process aggregation acknowledgement: failed to decode result: %w", err)
	}
	acknowledgedKey, err := acknowledged.SubscriptionKey()
	if err != nil {
		return fmt.Errorf("failed to process aggregation acknowledgement: %w", err)
	}
	if acknowledgedKey != wantKey {
		return fmt.Errorf("failed to process aggregation acknowledgement: subscription_key=invalid")
	}
	return nil
}
