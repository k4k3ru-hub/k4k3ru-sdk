package scalping

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observation "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/perpetual"
)

func runPositionSnapshotFixture() RunSnapshot {
	idle := RunEvaluation{Status: EvaluationStatusNotMatched, Markets: []RunMarketEvaluation{}}
	return RunSnapshot{
		EvaluationID: "eval", MarketType: market.MarketTypePerpetual, Symbol: "SUI/USDC", EvaluatedAt: 10000, Entry: idle,
		Positions: []RunPosition{{
			Market:         market.MarketRef{Venue: "hyperliquid", Network: "testnet", VenueSymbol: "SUI"},
			AccountAddress: "account", SyncStatus: RunPositionSynced, ObservedAt: pointer(int64(9000)),
			Position: &perpetual.Position{Quantity: "-10", EntryPrice: pointer("2"), Leverage: 3, MarginMode: "isolated", MarginUsed: "6.666667", UnrealizedPnL: "-0.125"},
			Exit:     idle,
		}},
	}
}

func runPositionUnavailable(reason string) RunEvaluation {
	return RunEvaluation{Status: EvaluationStatusUnavailable, Markets: []RunMarketEvaluation{}, Reasons: []string{reason}}
}

func runPositionMatched(ref market.MarketRef, trigger string) RunEvaluation {
	return RunEvaluation{Status: EvaluationStatusMatched, Markets: []RunMarketEvaluation{{
		Price:  observation.MarketPrice{Market: ref, Status: observation.PriceStatusReference, NetPrice: pointer("2"), ObservedAt: pointer(int64(9000))},
		Status: EvaluationStatusMatched, Candidate: &RunCandidate{CandidateID: "candidate", Revision: 1}, Trigger: trigger,
	}}}
}

// TestRunPositionReplacement verifies venue data survives delivery and closure clears it explicitly.
//
// Version:
//   - 2026-10-03: Added.
func TestRunPositionReplacement(t *testing.T) {
	s := runPositionSnapshotFixture()
	e := RunEvent{ExecutionID: "run", SubscriptionKey: "key", Sequence: 1, Kind: EventKindSnapshot, Snapshot: &s}
	var received RunEvent
	for _, quantity := range []string{"-10", "-4.25", "7.5"} {
		s.Positions[0].Position.Quantity = quantity
		raw, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), `"orders"`) || strings.Contains(string(raw), `"orderId"`) {
			t.Fatal("perpetual snapshot emitted synthetic order state")
		}
		if err := json.Unmarshal(raw, &received); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(e, received) {
			t.Fatal("venue quantity, margin or pnl changed during round trip")
		}
		e.Sequence++
	}
	s.Positions[0].Position = nil
	s.Positions[0].Exit = runPositionUnavailable("position_flat")
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &received); err != nil {
		t.Fatal(err)
	}
	p := received.Snapshot.Positions[0]
	if p.Position != nil || p.SyncStatus != RunPositionSynced || p.ObservedAt == nil {
		t.Fatal("confirmed flat scope did not replace prior position")
	}
	for _, status := range []RunPositionSyncStatus{RunPositionSyncing, RunPositionUnavailable} {
		s.Entry = runPositionUnavailable("position_unavailable")
		s.Positions[0].SyncStatus = status
		s.Positions[0].ObservedAt = nil
		s.Positions[0].Reasons = []string{"reconciliation_required"}
		raw, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &received); err != nil {
			t.Fatal(err)
		}
		if received.Snapshot.Positions[0].SyncStatus == RunPositionSynced {
			t.Fatal("unknown scope was represented as confirmed flat")
		}
	}
}

// TestRunPositionValidation rejects contradictory synchronization, ownership and numeric data.
//
// Version:
//   - 2026-10-03: Added.
func TestRunPositionValidation(t *testing.T) {
	for name, change := range map[string]func(*RunSnapshot){
		"missing positions": func(s *RunSnapshot) { s.Positions = nil },
		"empty positions":   func(s *RunSnapshot) { s.Positions = []RunPosition{} },
		"perpetual orders":  func(s *RunSnapshot) { s.Orders = []RunOrder{} },
		"spot positions":    func(s *RunSnapshot) { s.MarketType = market.MarketTypeSpot; s.Orders = []RunOrder{} },
		"pool position":     func(s *RunSnapshot) { s.Positions[0].Market = runSnapshotFixture().Orders[0].Market },
		"missing account":   func(s *RunSnapshot) { s.Positions[0].AccountAddress = "" },
		"missing status":    func(s *RunSnapshot) { s.Positions[0].SyncStatus = "" },
		"missing time":      func(s *RunSnapshot) { s.Positions[0].ObservedAt = nil },
		"future time":       func(s *RunSnapshot) { s.Positions[0].ObservedAt = pointer(int64(10001)) },
		"zero time":         func(s *RunSnapshot) { s.Positions[0].ObservedAt = pointer(int64(0)) },
		"synced reasons":    func(s *RunSnapshot) { s.Positions[0].Reasons = []string{"unknown"} },
		"unknown treated as idle": func(s *RunSnapshot) {
			s.Positions[0].SyncStatus = RunPositionSyncing
			s.Positions[0].Reasons = []string{"pending"}
		},
		"unknown missing reasons": func(s *RunSnapshot) {
			s.Entry = runPositionUnavailable("pending")
			s.Positions[0].SyncStatus = RunPositionSyncing
		},
		"flat idle exit":            func(s *RunSnapshot) { s.Positions[0].Position = nil },
		"zero holding":              func(s *RunSnapshot) { s.Positions[0].Position.Quantity = "-0.00" },
		"exponent quantity":         func(s *RunSnapshot) { s.Positions[0].Position.Quantity = "1e2" },
		"invalid price":             func(s *RunSnapshot) { s.Positions[0].Position.EntryPrice = pointer("0") },
		"invalid liquidation price": func(s *RunSnapshot) { s.Positions[0].Position.LiquidationPrice = pointer("-2") },
		"zero leverage":             func(s *RunSnapshot) { s.Positions[0].Position.Leverage = 0 },
		"invalid margin mode":       func(s *RunSnapshot) { s.Positions[0].Position.MarginMode = "other" },
		"negative margin":           func(s *RunSnapshot) { s.Positions[0].Position.MarginUsed = "-1" },
		"unknown pnl":               func(s *RunSnapshot) { s.Positions[0].Position.UnrealizedPnL = "NaN" },
		"duplicate scope": func(s *RunSnapshot) {
			p := s.Positions[0]
			p.Market.Venue = " Hyperliquid "
			s.Positions = append(s.Positions, p)
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := runPositionSnapshotFixture()
			change(&s)
			if err := s.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("invalid snapshot did not retain validation error: %v", err)
			}
			if _, err := json.Marshal(s); err == nil {
				t.Fatal("invalid snapshot encoded")
			}
		})
	}
}

// TestRunPositionExitScopeAndPriority verifies only reconciled venue positions can propose exits.
//
// Version:
//   - 2026-10-03: Added.
func TestRunPositionExitScopeAndPriority(t *testing.T) {
	s := runPositionSnapshotFixture()
	p := &s.Positions[0]
	p.Exit = runPositionMatched(p.Market, "take_profit")
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	s.Entry = runPositionMatched(p.Market, "")
	if s.Validate() == nil {
		t.Fatal("entry survived settlement priority")
	}
	s.Entry = runPositionUnavailable("settlement_priority")
	p.Exit.Markets[0].Price.Market.Network = "mainnet"
	if s.Validate() == nil {
		t.Fatal("exit moved to another venue scope")
	}
	p.Exit = runPositionMatched(p.Market, "stop_loss")
	p.SyncStatus, p.Reasons = RunPositionSyncing, []string{"submission_unknown"}
	if s.Validate() == nil {
		t.Fatal("unknown position remained actionable")
	}
	p.Exit = runPositionUnavailable("submission_unknown")
	if err := s.Validate(); err != nil {
		t.Fatal("last known venue position could not be shown", err)
	}
	// An unknown scope blocks new exposure, but another reconciled scope can exit.
	other := runPositionSnapshotFixture().Positions[0]
	other.Market.VenueSymbol = "dex:SUI"
	other.Exit = runPositionMatched(other.Market, "stop_loss")
	s.Positions = append(s.Positions, other)
	if err := s.Validate(); err != nil {
		t.Fatal("reconciled scope could not exit", err)
	}
}

// TestRunPositionJSON validates the published example and preserves state after malformed replacements.
//
// Version:
//   - 2026-10-03: Added.
func TestRunPositionJSON(t *testing.T) {
	raw, err := os.ReadFile("testdata/run_perpetual_snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var original RunEvent
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	for name, mutation := range map[string]string{
		"unknown field":           strings.Replace(string(raw), `"syncStatus": "synced"`, `"syncStatus": "synced", "unknown": true`, 1),
		"duplicate field":         strings.Replace(string(raw), `"syncStatus": "synced"`, `"syncStatus": "synced", "syncStatus": "syncing"`, 1),
		"explicit null":           strings.Replace(string(raw), `"entryPrice": "2"`, `"entryPrice": null`, 1),
		"wrong product array":     strings.Replace(string(raw), `"positions": [`, `"orders": [], "positions": [`, 1),
		"unknown nested position": strings.Replace(string(raw), `"quantity": "-10"`, `"quantity": "-10", "orderId": "1"`, 1),
		"invalid nested pnl":      strings.Replace(string(raw), `"unrealizedPnl": "0.1"`, `"unrealizedPnl": "Infinity"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			received := original
			if err := json.Unmarshal([]byte(mutation), &received); err == nil {
				t.Fatal("malformed snapshot accepted")
			}
			if !reflect.DeepEqual(original, received) {
				t.Fatal("failed replacement mutated previous state")
			}
		})
	}
	var destination *RunSnapshot
	if err := destination.UnmarshalJSON([]byte(`{}`)); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("nil destination not rejected", err)
	}
}
