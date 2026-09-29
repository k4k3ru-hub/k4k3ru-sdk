package scalping

import (
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observation "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
)

// TestSettlementContract rejects unrelated markets and incomplete settlement quantities.
//
// Version:
//   - 2026-09-29: Added.
func TestSettlementContract(t *testing.T) {
	p := spotParams()
	p.ExecutionRule.Close.Spot.Markets = []rule.MarketRef{{Chain: "sui", Network: "mainnet", Venue: "cetus", PoolID: "sui-usdc-pool"}}
	fixture := func() Result {
		r := matchedResult()
		price := r.Markets[0].Price
		price.Status = observation.PriceStatusVWAP
		price.NetReceiveQuantity = &market.Quantity{Amount: "2000000", Decimals: 6}
		r.Markets = []MarketEvaluation{}
		r.State = &ExecutionState{Status: "holding", Revision: strings.Repeat("a", 64), Owner: "wallet", ExecutionID: "exec", TransactionID: "tx", Quantity: &market.Quantity{Amount: "1000000000", Decimals: 9}, AcquiredAt: pointer(r.EvaluatedAt - 1000)}
		r.Settlement = &Settlement{Quantity: *r.State.Quantity, Markets: []SettlementMarket{{Price: price, Trigger: "take_profit"}}}
		return r
	}
	if err := fixture().ValidateFor(p); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Result){
		"quantity": func(r *Result) { r.Settlement.Quantity.Amount = "1" },
		"state":    func(r *Result) { r.State.Status = "pending" },
		"revision": func(r *Result) { r.State.Revision = "invalid" },
		"market":   func(r *Result) { r.Settlement.Markets[0].Price.Market.PoolID = "unrelated" },
		"reference": func(r *Result) {
			r.Settlement.Markets[0].Price.Status = observation.PriceStatusReference
			r.Settlement.Markets[0].Price.NetReceiveQuantity = nil
		},
		"duplicate": func(r *Result) { r.Settlement.Markets = append(r.Settlement.Markets, r.Settlement.Markets[0]) },
	} {
		t.Run(name, func(t *testing.T) {
			r := fixture()
			mutate(&r)
			if err := r.ValidateFor(p); err == nil {
				t.Fatal("invalid settlement accepted")
			}
		})
	}
}
