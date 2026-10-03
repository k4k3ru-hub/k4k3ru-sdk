package websocket

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping"
	event "github.com/k4k3ru-hub/k4k3ru-sdk/go/subscription"
)

// TestScalpingRunPositionRouting verifies a resume ACK and complete venue-position replacements.
//
// Version:
//   - 2026-10-03: Added.
func TestScalpingRunPositionRouting(t *testing.T) {
	m, err := newModule(t.Context(), validModuleConfig("wss://api.k4k3ru.com/"), validModuleDeps())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	raw, err := os.ReadFile("../jsonrpc/tradehub/scalping/testdata/run_perpetual_sell.json")
	if err != nil {
		t.Fatal(err)
	}
	var params dto.RunParams
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatal(err)
	}
	raw, err = os.ReadFile("../jsonrpc/tradehub/scalping/testdata/run_perpetual_snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var notification dto.RunEvent
	if err := json.Unmarshal(raw, &notification); err != nil {
		t.Fatal(err)
	}
	route := func() {
		t.Helper()
		data, err := json.Marshal(notification)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := json.Marshal(event.Event{Type: event.EventTypeScalpingRun, Data: data})
		if err != nil {
			t.Fatal(err)
		}
		if err := m.router.route(wire); err != nil {
			t.Fatal(err)
		}
	}
	client, err := newScalpingRunClient(executionSender(func(_ context.Context, method rpc.Method, request json.RawMessage) (*rpc.Response, error) {
		if method != rpc.MethodTradeHubScalpingRun || string(request) != `{"executionId":"example-perpetual-run"}` {
			t.Fatalf("unexpected resume request: %s %s", method, request)
		}
		// Exercise delivery before acknowledgement through the real typed router.
		route()
		ack, err := json.Marshal(dto.RunResult{ExecutionID: notification.ExecutionID, SubscriptionKey: notification.SubscriptionKey, Params: params.Params})
		if err != nil {
			return nil, err
		}
		return &rpc.Response{Result: ack}, nil
	}), m.router.scalpingRunEvents, m.subscriptions)
	if err != nil {
		t.Fatal(err)
	}
	s, err := client.Run(t.Context(), dto.RunParams{ExecutionID: notification.ExecutionID})
	if err != nil {
		t.Fatal(err)
	}
	first := <-s.Events()
	if first.Snapshot == nil || len(first.Snapshot.Positions) != 1 || first.Snapshot.Orders != nil || first.Snapshot.Positions[0].Position.Quantity != "-10" {
		t.Fatal("typed venue position did not survive routing")
	}
	notification.Sequence++
	notification.Snapshot.Positions[0].Position = nil
	notification.Snapshot.Positions[0].Exit = dto.RunEvaluation{Status: dto.EvaluationStatusUnavailable, Markets: []dto.RunMarketEvaluation{}, Reasons: []string{"position_flat"}}
	route()
	closed := <-s.Events()
	if closed.Sequence != 2 || closed.Snapshot.Positions[0].Position != nil || closed.Snapshot.Positions[0].SyncStatus != dto.RunPositionSynced {
		t.Fatal("flat replacement did not clear venue position")
	}
	if first.Snapshot.Positions[0].Position == nil {
		t.Fatal("new notification mutated previously delivered data")
	}
}
