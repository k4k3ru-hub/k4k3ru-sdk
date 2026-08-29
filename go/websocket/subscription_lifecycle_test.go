package websocket

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestSubscriptionLifecycleDisconnectsAfterLastUnsubscribeACK(t *testing.T) {
	t.Parallel()

	transport := &fakeSubscriptionTransport{}
	lifecycle, _ := newSubscriptionLifecycle(transport)
	succeed := func(context.Context) error { return nil }
	if err := lifecycle.subscribe(context.Background(), "first", succeed); err != nil {
		t.Fatalf("subscribe first error = %v", err)
	}
	if err := lifecycle.subscribe(context.Background(), "second", succeed); err != nil {
		t.Fatalf("subscribe second error = %v", err)
	}
	if err := lifecycle.unsubscribe(context.Background(), "first", succeed); err != nil {
		t.Fatalf("unsubscribe first error = %v", err)
	}
	if transport.disconnectCount() != 0 {
		t.Fatalf("disconnect count after first unsubscribe = %d", transport.disconnectCount())
	}
	if err := lifecycle.unsubscribe(context.Background(), "second", succeed); err != nil {
		t.Fatalf("unsubscribe second error = %v", err)
	}
	if transport.disconnectCount() != 1 {
		t.Fatalf("disconnect count = %d, want 1", transport.disconnectCount())
	}
}

func TestSubscriptionLifecycleDoesNotDisconnectForConcurrentSubscribe(t *testing.T) {
	t.Parallel()

	transport := &fakeSubscriptionTransport{}
	lifecycle, _ := newSubscriptionLifecycle(transport)
	succeed := func(context.Context) error { return nil }
	if err := lifecycle.subscribe(context.Background(), "first", succeed); err != nil {
		t.Fatalf("subscribe first error = %v", err)
	}

	unsubscribeStarted := make(chan struct{})
	unsubscribeACK := make(chan struct{})
	unsubscribeDone := make(chan error, 1)
	go func() {
		unsubscribeDone <- lifecycle.unsubscribe(context.Background(), "first", func(context.Context) error {
			close(unsubscribeStarted)
			<-unsubscribeACK
			return nil
		})
	}()
	<-unsubscribeStarted

	subscribeStarted := make(chan struct{})
	subscribeACK := make(chan struct{})
	subscribeDone := make(chan error, 1)
	go func() {
		subscribeDone <- lifecycle.subscribe(context.Background(), "second", func(context.Context) error {
			close(subscribeStarted)
			<-subscribeACK
			return nil
		})
	}()
	<-subscribeStarted
	close(unsubscribeACK)
	if err := <-unsubscribeDone; err != nil {
		t.Fatalf("unsubscribe error = %v", err)
	}
	if transport.disconnectCount() != 0 {
		t.Fatalf("disconnect count = %d, want 0", transport.disconnectCount())
	}
	close(subscribeACK)
	if err := <-subscribeDone; err != nil {
		t.Fatalf("subscribe second error = %v", err)
	}
}

func TestSubscriptionLifecyclePreservesActiveStateAfterUnsubscribeError(t *testing.T) {
	t.Parallel()

	transport := &fakeSubscriptionTransport{}
	lifecycle, _ := newSubscriptionLifecycle(transport)
	succeed := func(context.Context) error { return nil }
	if err := lifecycle.subscribe(context.Background(), "subscription", succeed); err != nil {
		t.Fatalf("subscribe error = %v", err)
	}
	wantErr := errors.New("gateway rejected unsubscribe")
	err := lifecycle.unsubscribe(context.Background(), "subscription", func(context.Context) error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("unsubscribe error = %v", err)
	}
	if transport.disconnectCount() != 0 {
		t.Fatalf("disconnect count = %d, want 0", transport.disconnectCount())
	}
	if err := lifecycle.unsubscribe(context.Background(), "subscription", succeed); err != nil {
		t.Fatalf("retry unsubscribe error = %v", err)
	}
}

func TestSubscriptionLifecycleDisconnectsAfterFailedOnlySubscribe(t *testing.T) {
	t.Parallel()

	transport := &fakeSubscriptionTransport{}
	lifecycle, _ := newSubscriptionLifecycle(transport)
	wantErr := errors.New("gateway rejected subscribe")
	err := lifecycle.subscribe(context.Background(), "subscription", func(context.Context) error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("subscribe error = %v", err)
	}
	if transport.disconnectCount() != 1 {
		t.Fatalf("disconnect count = %d, want 1", transport.disconnectCount())
	}
}

func TestSubscriptionLifecycleValidatesInputs(t *testing.T) {
	t.Parallel()

	if _, err := newSubscriptionLifecycle(nil); err == nil || !strings.Contains(err.Error(), "transport=null") {
		t.Fatalf("nil transport error = %v", err)
	}
	lifecycle, _ := newSubscriptionLifecycle(&fakeSubscriptionTransport{})
	succeed := func(context.Context) error { return nil }
	if err := lifecycle.subscribe(nil, "key", succeed); err == nil || !strings.Contains(err.Error(), "context=null") {
		t.Fatalf("nil context error = %v", err)
	}
	if err := lifecycle.subscribe(context.Background(), "", succeed); err == nil || !strings.Contains(err.Error(), "subscription_key=empty") {
		t.Fatalf("empty key error = %v", err)
	}
	if err := lifecycle.subscribe(context.Background(), "key", nil); err == nil || !strings.Contains(err.Error(), "operation=null") {
		t.Fatalf("nil operation error = %v", err)
	}
	if err := lifecycle.subscribe(context.Background(), "key", succeed); err != nil {
		t.Fatalf("subscribe error = %v", err)
	}
	if err := lifecycle.subscribe(context.Background(), "key", succeed); err == nil || !strings.Contains(err.Error(), "subscription_key=duplicate") {
		t.Fatalf("duplicate subscribe error = %v", err)
	}
}

type fakeSubscriptionTransport struct {
	mu          sync.Mutex
	disconnects int
	err         error
}

func (f *fakeSubscriptionTransport) disconnect() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.disconnects++
	return f.err
}

func (f *fakeSubscriptionTransport) disconnectCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.disconnects
}
