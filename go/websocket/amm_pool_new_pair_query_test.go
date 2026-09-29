package websocket

import (
	"testing"

	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/ammpool/newpair"
)

// TestAMMPoolNewPairQueryRouting verifies query isolation and detached subscription parameters.
//
// Version:
//   - 2026-09-29: Added.
func TestAMMPoolNewPairQueryRouting(t *testing.T) {
	registry := newAMMPoolNewPairEventRegistry()
	sender := &ammPoolNewPairSender{}
	lifecycle, err := newSubscriptionLifecycle(&fakeSubscriptionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	client, err := newAMMPoolNewPairClient(sender, lifecycle, registry)
	if err != nil {
		t.Fatal(err)
	}
	params := dto.Params{Chain: "base", Query: &dto.Query{Conditions: dto.QueryConditions{MinSwapCount: "01"}}}
	first, err := client.Subscribe(t.Context(), params)
	if err != nil {
		t.Fatal(err)
	}
	params.Query.Conditions.MinSwapCount = "2"
	second, err := client.Subscribe(t.Context(), params)
	if err != nil {
		t.Fatal(err)
	}
	copy := first.Params()
	copy.Query.Conditions.MinSwapCount = "99"
	if first.Params().Query.Conditions.MinSwapCount != "1" {
		t.Fatal("subscription query was aliased")
	}
	if ok, err := registry.route(dto.Result{Filter: params.Normalize(), QueryResult: &dto.QueryResult{MatchedCount: "0"}}); err != nil || !ok {
		t.Fatal(ok, err)
	}
	select {
	case <-first.Events():
		t.Fatal("different query received event")
	default:
	}
	select {
	case result := <-second.Events():
		if result.QueryResult == nil || result.QueryResult.ExpiresAt != nil {
			t.Fatal("live metadata changed")
		}
	default:
		t.Fatal("query event was lost")
	}
	if err := client.Unsubscribe(t.Context(), first); err != nil {
		t.Fatal(err)
	}
	if err := client.Unsubscribe(t.Context(), second); err != nil {
		t.Fatal(err)
	}
}
