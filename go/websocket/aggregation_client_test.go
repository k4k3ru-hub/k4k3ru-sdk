package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	k4k3ruSDKJSONRPC "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	k4k3ruSDKJSONRPCAggregation "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/aggregation"
)

func TestAggregationClientSubscribeAndUnsubscribe(t *testing.T) {
	t.Parallel()

	params := validAggregationParams()
	sender := &fakeJSONRPCSender{params: params}
	transport := &fakeSubscriptionTransport{}
	lifecycle, _ := newSubscriptionLifecycle(transport)
	registry := newAggregationEventRegistry()
	client, err := newAggregationClient(sender, lifecycle, registry)
	if err != nil {
		t.Fatalf("newAggregationClient() error = %v", err)
	}
	subscription, err := client.Subscribe(context.Background(), params)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if subscription == nil || subscription.Events() == nil || subscription.Params().Symbol != params.Symbol {
		t.Fatalf("Subscribe() = %#v", subscription)
	}
	if err := client.Unsubscribe(context.Background(), subscription); err != nil {
		t.Fatalf("Unsubscribe() error = %v", err)
	}
	if _, open := <-subscription.Events(); open {
		t.Fatal("subscription event channel remains open")
	}
	if transport.disconnectCount() != 1 {
		t.Fatalf("disconnect count = %d, want 1", transport.disconnectCount())
	}
	methods := sender.sentMethods()
	wantMethods := []k4k3ruSDKJSONRPC.Method{k4k3ruSDKJSONRPC.MethodMarketHubAggregationSubscribe, k4k3ruSDKJSONRPC.MethodMarketHubAggregationUnsubscribe}
	if len(methods) != len(wantMethods) || methods[0] != wantMethods[0] || methods[1] != wantMethods[1] {
		t.Fatalf("methods = %#v, want %#v", methods, wantMethods)
	}
}

func TestAggregationClientRollsBackFailedSubscribe(t *testing.T) {
	t.Parallel()

	params := validAggregationParams()
	wantErr := errors.New("subscribe rejected")
	sender := &fakeJSONRPCSender{err: wantErr}
	transport := &fakeSubscriptionTransport{}
	lifecycle, _ := newSubscriptionLifecycle(transport)
	registry := newAggregationEventRegistry()
	client, _ := newAggregationClient(sender, lifecycle, registry)
	if _, err := client.Subscribe(context.Background(), params); !errors.Is(err, wantErr) {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if transport.disconnectCount() != 1 {
		t.Fatalf("disconnect count = %d, want 1", transport.disconnectCount())
	}
	if _, err := registry.register(params); err != nil {
		t.Fatalf("event subscription was not rolled back: %v", err)
	}
}

func TestAggregationClientPreservesSubscriptionAfterFailedUnsubscribe(t *testing.T) {
	t.Parallel()

	params := validAggregationParams()
	sender := &fakeJSONRPCSender{params: params}
	transport := &fakeSubscriptionTransport{}
	lifecycle, _ := newSubscriptionLifecycle(transport)
	registry := newAggregationEventRegistry()
	client, _ := newAggregationClient(sender, lifecycle, registry)
	subscription, err := client.Subscribe(context.Background(), params)
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	wantErr := errors.New("unsubscribe rejected")
	sender.err = wantErr
	if err := client.Unsubscribe(context.Background(), subscription); !errors.Is(err, wantErr) {
		t.Fatalf("Unsubscribe() error = %v", err)
	}
	if transport.disconnectCount() != 0 {
		t.Fatalf("disconnect count = %d, want 0", transport.disconnectCount())
	}
	if routed, err := registry.route(aggregationResult(params, "100")); err != nil || !routed {
		t.Fatalf("route after failed unsubscribe = %v, %v", routed, err)
	}
}

type fakeJSONRPCSender struct {
	mu      sync.Mutex
	params  k4k3ruSDKJSONRPCAggregation.Params
	methods []k4k3ruSDKJSONRPC.Method
	err     error
}

func (f *fakeJSONRPCSender) send(_ context.Context, method k4k3ruSDKJSONRPC.Method, _ json.RawMessage) (*k4k3ruSDKJSONRPC.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.methods = append(f.methods, method)
	if f.err != nil {
		return nil, f.err
	}
	result, err := json.Marshal(f.params.Normalize())
	if err != nil {
		return nil, err
	}
	return &k4k3ruSDKJSONRPC.Response{ID: json.RawMessage(`1`), Result: result}, nil
}

func (f *fakeJSONRPCSender) sentMethods() []k4k3ruSDKJSONRPC.Method {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]k4k3ruSDKJSONRPC.Method(nil), f.methods...)
}
