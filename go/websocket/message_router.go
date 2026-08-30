package websocket

import (
	"bytes"
	"encoding/json"
	"fmt"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKJSONRPCAggregation "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/aggregation"
	k4k3ruSDKSubscription "github.com/k4k3ru-hub/k4k3ru-sdk/go/subscription"
)

type messageRouter struct {
	requests *requestTracker
	events   *aggregationEventRegistry
	errors   chan error
}

func newMessageRouter(requests *requestTracker, events *aggregationEventRegistry) (*messageRouter, error) {
	if requests == nil {
		return nil, fmt.Errorf("failed to create websocket message router: request_tracker=null")
	}
	if events == nil {
		return nil, fmt.Errorf("failed to create websocket message router: event_registry=null")
	}
	return &messageRouter{requests: requests, events: events, errors: make(chan error, 1)}, nil
}

func (r *messageRouter) HandleMessage(message []byte) {
	if r == nil {
		return
	}
	if err := r.route(message); err != nil {
		r.report(err)
	}
}

func (r *messageRouter) HandleClose() {
	if r == nil || r.requests == nil {
		return
	}
	r.requests.failAll(errWebSocketConnectionClosed)
}

func (r *messageRouter) route(message []byte) error {
	if len(bytes.TrimSpace(message)) == 0 {
		return fmt.Errorf("failed to route websocket message: message=empty")
	}
	var envelope struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(message, &envelope); err != nil {
		return fmt.Errorf("failed to route websocket message: failed to decode json: %w", err)
	}
	if len(bytes.TrimSpace(envelope.ID)) > 0 && !bytes.Equal(bytes.TrimSpace(envelope.ID), []byte("null")) {
		var response k4k3ruSDKJSONRPC.Response
		if err := json.Unmarshal(message, &response); err != nil {
			return fmt.Errorf("failed to route websocket response: failed to decode json: %w", err)
		}
		if len(response.Result) == 0 && response.Error == nil {
			return fmt.Errorf("failed to route websocket response: response_payload=empty")
		}
		if len(response.Result) > 0 && response.Error != nil {
			return fmt.Errorf("failed to route websocket response: response_payload=invalid")
		}
		if _, err := r.requests.resolve(response); err != nil {
			return fmt.Errorf("failed to route websocket response: %w", err)
		}
		return nil
	}
	var event k4k3ruSDKSubscription.Event
	if err := json.Unmarshal(message, &event); err != nil {
		return fmt.Errorf("failed to route websocket event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("failed to route websocket event: %w", err)
	}
	switch event.Type {
	case k4k3ruSDKSubscription.EventTypeAggregation:
		var result k4k3ruSDKJSONRPCAggregation.Result
		if err := json.Unmarshal(event.Data, &result); err != nil {
			return fmt.Errorf("failed to route aggregation event: failed to decode data: %w", err)
		}
		if result.Price == "" {
			return fmt.Errorf("failed to route aggregation event: price=empty")
		}
		if _, err := r.events.route(result); err != nil {
			return fmt.Errorf("failed to route websocket message: %w", err)
		}
	case k4k3ruSDKSubscription.EventTypeArbitrage:
		return fmt.Errorf("failed to route websocket event: event_type=unsupported: event_type=%q", event.Type)
	}
	return nil
}

func (r *messageRouter) report(err error) {
	if r == nil || err == nil {
		return
	}
	select {
	case r.errors <- err:
	default:
		select {
		case <-r.errors:
		default:
		}
		select {
		case r.errors <- err:
		default:
		}
	}
}
