package scalping

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observation "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
)

func runSnapshotFixture() RunSnapshot {
	ref := market.MarketRef{Venue: "cetus", Chain: "sui", Network: "testnet", PoolID: "pool"}
	idle := RunEvaluation{Status: EvaluationStatusNotMatched, Markets: []RunMarketEvaluation{}}
	return RunSnapshot{EvaluationID: "eval", MarketType: market.MarketTypeSpot, Symbol: "SUI/USDC", EvaluatedAt: 10000, Entry: idle, Orders: []RunOrder{
		{OrderID: "123", Revision: "r1", Status: RunOrderHolding, Market: ref, AccountAddress: "wallet", Side: SideBuy, RemainingQuantity: &market.Quantity{Amount: "4"}, EntryValue: &market.Quantity{Amount: "8"}, AcquiredAt: pointer(int64(9000)), Exit: idle},
		{OrderID: "124", Revision: "r2", Status: RunOrderPending, Market: ref, AccountAddress: "wallet", Side: SideBuy, Exit: idle},
	}}
}

// TestRunSnapshotReplacementAndOrderValidation verifies independent orders and strict full-state delivery.
//
// Version:
//   - 2026-10-01: Added.
func TestRunSnapshotReplacementAndOrderValidation(t *testing.T) {
	s := runSnapshotFixture()
	e := RunEvent{ExecutionID: "run", SubscriptionKey: "key", Sequence: 1, Kind: EventKindSnapshot, Snapshot: &s}
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var decoded RunEvent
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Snapshot.Orders) != 2 {
		t.Fatal("multiple orders lost")
	}
	s.Orders = []RunOrder{}
	raw, err = json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &decoded); err != nil || len(decoded.Snapshot.Orders) != 0 {
		t.Fatal("closed order retained", err)
	}
	for _, raw := range []string{strings.Replace(string(raw), `"orders":[]`, `"orders":null`, 1), strings.Replace(string(raw), `"orders":[]`, `"unknown":0,"orders":[]`, 1)} {
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			t.Fatal("invalid replacement accepted")
		}
	}
	for _, id := range []string{"0", "01", "1.0", "18446744073709551616", "124"} {
		s = runSnapshotFixture()
		s.Orders[0].OrderID = id
		if s.Validate() == nil {
			t.Fatal("invalid or duplicate order id accepted", id)
		}
	}
	s = runSnapshotFixture()
	s.Orders[0].Status = "closed"
	if s.Validate() == nil {
		t.Fatal("closed order included")
	}
	s = runSnapshotFixture()
	s.Orders[0].RemainingQuantity = nil
	if s.Validate() == nil {
		t.Fatal("holding without quantity")
	}
}

// TestRunSnapshotExitPriorityWithoutTradeAge verifies optional trade timestamps and mandatory exit priority.
//
// Version:
//   - 2026-10-01: Added.
func TestRunSnapshotExitPriorityWithoutTradeAge(t *testing.T) {
	s := runSnapshotFixture()
	price := observation.MarketPrice{Market: s.Orders[0].Market, Status: observation.PriceStatusReference, NetPrice: pointer("2"), ObservedAt: pointer(int64(9000))}
	m := RunMarketEvaluation{Price: price, Status: EvaluationStatusMatched, Candidate: &RunCandidate{CandidateID: "candidate", Revision: 1}}
	s.Entry = RunEvaluation{Status: EvaluationStatusMatched, Markets: []RunMarketEvaluation{m}}
	if err := s.Validate(); err != nil {
		t.Fatal("trade time unexpectedly required", err)
	}
	m.Trigger = "take_profit"
	s.Orders[0].Exit = RunEvaluation{Status: EvaluationStatusMatched, Markets: []RunMarketEvaluation{m}}
	if s.Validate() == nil {
		t.Fatal("entry not withdrawn for exit")
	}
	s.Entry = RunEvaluation{Status: EvaluationStatusNotMatched, Markets: []RunMarketEvaluation{}}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Orders[0].Status = RunOrderPending
	if s.Validate() == nil {
		t.Fatal("uncertain order actionable")
	}
}
