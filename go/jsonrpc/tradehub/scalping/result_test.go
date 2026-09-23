package scalping

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
)

func matchedResult() Result {
	p := spotParams()
	const now int64 = 1700000030000
	return Result{
		EvaluationID: "evaluation-1", MarketType: p.MarketType, EvaluatedAt: now,
		BaseAsset:  AssetMetadata{Reference: p.BaseAsset, Symbol: "SUI", Decimals: 9},
		QuoteAsset: AssetMetadata{Reference: p.QuoteAsset, Symbol: "USDC", Decimals: 6},
		Markets: []MarketEvaluation{{Market: p.Markets[0], Status: EvaluationStatusMatched,
			Metrics:   &Metrics{WindowStart: now - 30000, WindowEnd: now, LastObservedAt: now - 100, TradeCount: pointer(uint64(10))},
			Candidate: &Candidate{CandidateID: "candidate-1", Revision: 1, ExpiresAt: now + 2000},
		}},
	}
}

// TestResultStates verifies eligible, unmatched, and unavailable snapshots.
//
// Version:
//   - 2026-09-23: Added.
func TestResultStates(t *testing.T) {
	for _, status := range []EvaluationStatus{EvaluationStatusMatched, EvaluationStatusNotMatched, EvaluationStatusUnavailable} {
		r := matchedResult()
		e := &r.Markets[0]
		e.Status = status
		if status != EvaluationStatusMatched {
			e.Candidate = nil
		}
		if status == EvaluationStatusUnavailable {
			e.Metrics = nil
			e.Reasons = []string{"data_gap"}
		}
		if err := r.ValidateFor(spotParams()); err != nil {
			t.Fatal(err)
		}
		wire, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var decoded Result
		if err := json.Unmarshal(wire, &decoded); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(r, decoded) {
			t.Fatal("snapshot changed")
		}
	}
}

// TestResultRejectsInvalidCandidates verifies candidate identity and availability.
//
// Version:
//   - 2026-09-23: Added.
func TestResultRejectsInvalidCandidates(t *testing.T) {
	tests := map[string]func(*Result){
		"missing candidate": func(r *Result) { r.Markets[0].Candidate = nil },
		"expired candidate": func(r *Result) { r.Markets[0].Candidate.ExpiresAt = r.EvaluatedAt },
		"missing revision":  func(r *Result) { r.Markets[0].Candidate.Revision = 0 },
		"unavailable candidate": func(r *Result) {
			r.Markets[0].Status = EvaluationStatusUnavailable
			r.Markets[0].Reasons = []string{"stale_data"}
		},
		"unmatched candidate":        func(r *Result) { r.Markets[0].Status = EvaluationStatusNotMatched },
		"unavailable without reason": func(r *Result) { r.Markets[0].Status = EvaluationStatusUnavailable; r.Markets[0].Candidate = nil },
		"future observation":         func(r *Result) { r.Markets[0].Metrics.LastObservedAt = r.EvaluatedAt + 1 },
		"invalid window":             func(r *Result) { r.Markets[0].Metrics.WindowStart = r.Markets[0].Metrics.WindowEnd },
		"missing metrics":            func(r *Result) { r.Markets[0].Metrics = nil },
		"empty metrics":              func(r *Result) { r.Markets[0].Metrics.TradeCount = nil },
		"fractional raw volume":      func(r *Result) { r.Markets[0].Metrics.QuoteVolume = pointer("0.1") },
		"invalid ratio":              func(r *Result) { r.Markets[0].Metrics.BuyVolumeRatioBPS = pointer("10001") },
		"duplicate candidate": func(r *Result) {
			other := r.Markets[0]
			other.Market.PoolID = "another-pool"
			r.Markets = append(r.Markets, other)
		},
		"empty markets": func(r *Result) { r.Markets = nil },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			r := matchedResult()
			mutate(&r)
			if err := r.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("invalid result accepted: %v", err)
			}
		})
	}
	r := matchedResult()
	r.Markets[0].Metrics.TradeCount = pointer(uint64(0))
	r.Markets[0].Status = EvaluationStatusNotMatched
	r.Markets[0].Candidate = nil
	if err := r.Validate(); err != nil {
		t.Fatal("explicit zero rejected:", err)
	}
}

// TestResultMatchesRequest verifies identity, required metrics, and evaluation freshness.
//
// Version:
//   - 2026-09-23: Added.
func TestResultMatchesRequest(t *testing.T) {
	normalized := matchedResult()
	normalized.MarketType = " SPOT "
	normalized.BaseAsset.Reference.Chain = " SUI "
	normalized.Markets[0].Market.Venue = " CETUS "
	if err := normalized.ValidateFor(spotParams()); err != nil {
		t.Fatal("equivalent references rejected:", err)
	}
	for name, mutate := range map[string]func(*Result){
		"wrong market type":     func(r *Result) { r.MarketType = rule.MarketTypePerp },
		"wrong reference asset": func(r *Result) { r.BaseAsset.Reference.AssetID = "other-token" },
		"wrong pool":            func(r *Result) { r.Markets[0].Market.PoolID = "other-pool" },
		"missing requested metric": func(r *Result) {
			r.Markets[0].Metrics.TradeCount = nil
			r.Markets[0].Metrics.PriceChangeBPS = pointer("1")
		},
		"stale data": func(r *Result) { r.Markets[0].Metrics.LastObservedAt = r.EvaluatedAt - 2001 },
	} {
		t.Run(name, func(t *testing.T) {
			r := matchedResult()
			mutate(&r)
			if err := r.ValidateFor(spotParams()); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("mismatched result accepted: %v", err)
			}
		})
	}
}

// TestMetadataRequiresDecimalCount distinguishes missing metadata from zero decimals.
//
// Version:
//   - 2026-09-23: Added.
func TestMetadataRequiresDecimalCount(t *testing.T) {
	for _, value := range []string{``, `,"decimals":null`, `,"decimals":256`} {
		var metadata AssetMetadata
		wire := `{"reference":{"chain":"sui","network":"mainnet","assetId":"token"},"symbol":"TOKEN"` + value + `}`
		if err := json.Unmarshal([]byte(wire), &metadata); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid metadata accepted: %v", err)
		}
	}
	var metadata AssetMetadata
	if err := json.Unmarshal([]byte(`{"reference":{"chain":"sui","network":"mainnet","assetId":"token"},"symbol":"TOKEN","decimals":0}`), &metadata); err != nil {
		t.Fatal(err)
	}
}

// TestSubscriptionEvents verifies snapshot and error exclusivity and strict decoding.
//
// Version:
//   - 2026-09-23: Added.
func TestSubscriptionEvents(t *testing.T) {
	r := matchedResult()
	events := []SubscriptionEvent{
		{SubscriptionKey: "subscription-1", Sequence: 1, Kind: EventKindSnapshot, Snapshot: &r},
		{SubscriptionKey: "subscription-1", Sequence: 2, Kind: EventKindError, Error: &StreamError{Code: "stream_unavailable", Retryable: true}},
	}
	for _, event := range events {
		wire, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		var decoded SubscriptionEvent
		if err := json.Unmarshal(wire, &decoded); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(event, decoded) {
			t.Fatal("event changed")
		}
	}
	for _, event := range []SubscriptionEvent{
		{SubscriptionKey: "subscription-1", Sequence: 0, Kind: EventKindSnapshot, Snapshot: &r},
		{SubscriptionKey: "subscription-1", Sequence: 1, Kind: EventKindSnapshot, Snapshot: &r, Error: &StreamError{Code: "error"}},
		{SubscriptionKey: "subscription-1", Sequence: 1, Kind: EventKindError},
	} {
		if err := event.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatal("invalid event accepted")
		}
	}
	var event SubscriptionEvent
	if err := json.Unmarshal([]byte(`{"subscriptionKey":"key","sequence":1,"kind":"error","error":{"code":"unavailable"}}`), &event); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("unspecified retryability accepted")
	}
	var ack SubscribeResult
	if err := json.Unmarshal([]byte(`{"subscriptionKey":"key"}`), &ack); err != nil {
		t.Fatal(err)
	}
	var unsubscribe UnsubscribeParams
	if err := json.Unmarshal([]byte(`{"subscriptionKey":"key","cancelOrders":true}`), &unsubscribe); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("unknown cancellation field accepted")
	}
	var result UnsubscribeResult
	if err := json.Unmarshal([]byte(`{"subscriptionKey":"key"}`), &result); err != nil {
		t.Fatal(err)
	}
}
