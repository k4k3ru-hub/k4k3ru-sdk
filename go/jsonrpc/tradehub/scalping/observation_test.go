package scalping

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observation "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
)

// TestObservationContract verifies optional targets, explicit quantities and absence of invented snapshot TTLs.
//
// Version:
//   - 2026-09-26: Added.
func TestObservationContract(t *testing.T) {
	p := spotParams()
	p.Markets[0].PoolID = ""
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	if p.ObservationParams().BaseQuantity != nil || p.Normalize().Conditions.MaximumSnapshotAgeMS != nil {
		t.Fatal("execution amount or TTL was inferred")
	}
	p.BaseQuantity = &market.Quantity{Amount: "1", Decimals: 9}
	p.Conditions.MaximumSnapshotAgeMS = pointer(uint64(math.MaxUint64))
	p.Conditions.QuoteVolume = &QuantityRange{Minimum: &market.Quantity{Amount: "100", Decimals: 2}, Maximum: &market.Quantity{Amount: "1000000", Decimals: 6}}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	n := p.Normalize()
	n.BaseQuantity.Amount = "2"
	n.Conditions.QuoteVolume.Minimum.Amount = "0"
	*n.Conditions.MaximumSnapshotAgeMS = 1
	if p.BaseQuantity.Amount != "1" || p.Conditions.QuoteVolume.Minimum.Amount != "100" || *p.Conditions.MaximumSnapshotAgeMS != math.MaxUint64 {
		t.Fatal("normalization aliased parameters")
	}
	wire, err := json.Marshal(SubscribeParams{IdempotencyKey: "one", Params: &p})
	if err != nil {
		t.Fatal(err)
	}
	var decoded SubscribeParams
	if err := json.Unmarshal(wire, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*decoded.Params, p) {
		t.Fatal("flattened params lost observation fields")
	}
	for _, mutate := range []func(*Params){func(p *Params) { p.Symbol = "" }, func(p *Params) { p.Conditions.MaximumSnapshotAgeMS = pointer(uint64(0)) }, func(p *Params) { p.Conditions.QuoteVolume.Minimum.Amount = "101" }, func(p *Params) { p.BaseQuantity.Amount = "0" }} {
		q := p.Normalize()
		mutate(&q)
		if err := q.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid observation accepted: %v", err)
		}
	}
}

// TestResultExpandedMarkets verifies unresolved scopes, fallback prices, units and replacement emptiness.
//
// Version:
//   - 2026-09-26: Added.
func TestResultExpandedMarkets(t *testing.T) {
	p := spotParams()
	p.Markets[0].PoolID = ""
	p.BaseQuantity = &market.Quantity{Amount: "1", Decimals: 0}
	r := matchedResult()
	r.Markets[0].Price.Status = observation.PriceStatusFallbackReference
	second := r.Markets[0]
	second.Price.Market.PoolID = "other"
	second.Candidate = &Candidate{CandidateID: "second", Revision: 1, ExpiresAt: r.Markets[0].Candidate.ExpiresAt}
	r.Markets = append(r.Markets, second)
	if err := r.ValidateFor(p); err != nil {
		t.Fatal(err)
	}
	r.Markets = []MarketEvaluation{}
	r.Metrics = nil
	if err := r.ValidateFor(p); err != nil {
		t.Fatal("empty resolved basket rejected", err)
	}
	p = spotParams()
	r = matchedResult()
	p.Markets[0].PoolID = "0x1"
	r.Markets[0].Price.Market.PoolID = "0x0000000000000000000000000000000000000000000000000000000000000001"
	if err := r.ValidateFor(p); err != nil {
		t.Fatal("equivalent Sui pool rejected", err)
	}
	r.Markets[0].Candidate.ExpiresAt++
	if err := r.ValidateFor(p); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("candidate exceeds configured data age")
	}
}
