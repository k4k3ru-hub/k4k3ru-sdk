package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
)

type executionSender func(context.Context, rpc.Method, json.RawMessage) (*rpc.Response, error)

func (f executionSender) send(c context.Context, m rpc.Method, p json.RawMessage) (*rpc.Response, error) {
	return f(c, m, p)
}

type executionTransport struct{ disconnects int }

func (f *executionTransport) disconnect() error { f.disconnects++; return nil }
func executionEvent(key string, seq uint64, status dto.ObservationStatus) dto.SubscriptionEvent {
	hash := "0x" + strings.Repeat("1", 64)
	s := &dto.ExecutionSnapshot{Status: status, ObservedAt: 1, Onchain: &dto.OnchainExecution{ChainFamily: dto.ChainFamilyEVM, Chain: "base", Network: "sepolia", TransactionID: hash}}
	if status.Terminal() {
		n := uint64(5)
		s.Onchain.BlockNumber = &n
		s.Onchain.BlockHash = hash
	}
	return dto.SubscriptionEvent{ExecutionID: "exec_one", SubscriptionKey: key, Sequence: seq, Kind: dto.ExecutionEventSnapshot, Snapshot: s}
}

// TestExecutionACKRaceStaleKeySequenceAndTerminal verifies execution ackrace stale key sequence and terminal.
//
// Version:
//   - 2026-09-16: Added.
func TestExecutionACKRaceStaleKeySequenceAndTerminal(t *testing.T) {
	r := newExecutionEventRegistry()
	transport := &executionTransport{}
	l, _ := newSubscriptionLifecycle(transport)
	sender := executionSender(func(context.Context, rpc.Method, json.RawMessage) (*rpc.Response, error) {
		r.route(executionEvent("sub_old", 50, dto.ObservationStatusSuccess))
		r.route(executionEvent("sub_new", 1, dto.ObservationStatusPending))
		return &rpc.Response{Result: json.RawMessage(`{"executionId":"exec_one","subscriptionKey":"sub_new"}`)}, nil
	})
	c, err := newExecutionClient(sender, r, l)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Subscribe(t.Context(), dto.SubscribeParams{ExecutionID: "exec_one"})
	if err != nil {
		t.Fatal(err)
	}
	if e := <-s.Events(); e.SubscriptionKey != "sub_new" || e.Sequence != 1 {
		t.Fatal(e)
	}
	r.route(executionEvent("sub_new", 1, dto.ObservationStatusPending))
	r.route(executionEvent("sub_old", 60, dto.ObservationStatusSuccess))
	r.route(executionEvent("sub_new", 2, dto.ObservationStatusSuccess))
	if e := <-s.Events(); e.Sequence != 2 || !e.Snapshot.Status.Terminal() {
		t.Fatal(e)
	}
	if _, ok := <-s.Events(); ok {
		t.Fatal("terminal channel retained")
	}
	if err := c.Unsubscribe(t.Context(), s); err != nil {
		t.Fatal(err)
	}
	if transport.disconnects != 0 || !l.keepOpen {
		t.Fatal("terminal closed session")
	}
}

// TestExecutionInterruptionAndDuplicateReuse verifies execution interruption and duplicate reuse.
//
// Version:
//   - 2026-09-16: Added.
func TestExecutionInterruptionAndDuplicateReuse(t *testing.T) {
	r := newExecutionEventRegistry()
	l, _ := newSubscriptionLifecycle(&executionTransport{})
	calls := 0
	sender := executionSender(func(context.Context, rpc.Method, json.RawMessage) (*rpc.Response, error) {
		calls++
		return &rpc.Response{Result: json.RawMessage(`{"executionId":"exec_one","subscriptionKey":"sub_one"}`)}, nil
	})
	c, _ := newExecutionClient(sender, r, l)
	a, err := c.Subscribe(t.Context(), dto.SubscribeParams{ExecutionID: "exec_one"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.Subscribe(t.Context(), dto.SubscribeParams{ExecutionID: "exec_one"})
	if err != nil || a != b || calls != 1 {
		t.Fatalf("duplicate: %v", err)
	}
	r.interrupt()
	if err := <-a.Errors(); !errors.Is(err, errWebSocketConnectionClosed) {
		t.Fatal(err)
	}
	if _, ok := <-a.Events(); ok {
		t.Fatal("silently waiting after disconnect")
	}
	if len(r.active) != 0 {
		t.Fatal("interrupted registry retained")
	}
}

// TestExecutionUnsubscribeKeepsOtherSubscriptionsAndConnection verifies execution unsubscribe keeps other subscriptions and connection.
//
// Version:
//   - 2026-09-16: Added.
func TestExecutionUnsubscribeKeepsOtherSubscriptionsAndConnection(t *testing.T) {
	r := newExecutionEventRegistry()
	transport := &executionTransport{}
	l, _ := newSubscriptionLifecycle(transport)
	sender := executionSender(func(_ context.Context, m rpc.Method, p json.RawMessage) (*rpc.Response, error) {
		if m == rpc.MethodTradeHubExecutionUnsubscribe {
			return &rpc.Response{Result: p}, nil
		}
		return &rpc.Response{Result: json.RawMessage(`{"executionId":"exec_one","subscriptionKey":"sub_one"}`)}, nil
	})
	c, _ := newExecutionClient(sender, r, l)
	s, err := c.Subscribe(t.Context(), dto.SubscribeParams{ExecutionID: "exec_one"})
	if err != nil {
		t.Fatal(err)
	}
	if err := l.subscribe(t.Context(), "other", func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := l.unsubscribe(t.Context(), "other", func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := c.Unsubscribe(t.Context(), s); err != nil {
		t.Fatal(err)
	}
	if transport.disconnects != 0 {
		t.Fatal("connection closed before idle deadline")
	}
}
