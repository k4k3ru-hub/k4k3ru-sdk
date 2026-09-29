package execution

import (
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"testing"
)

// TestExecutionPnLContract verifies optional settlement semantics and independent transaction finality.
//
// Version:
//   - 2026-09-29: Added.
func TestExecutionPnLContract(t *testing.T) {
	for _, status := range []PnLStatus{PnLStatusPending, PnLStatusNotApplicable, PnLStatusUnavailable, PnLStatusRealized} {
		p := ExecutionPnL{Status: status}
		if status == PnLStatusRealized {
			p.Settlement = &PnLSettlement{AssetID: "sui", Quantity: market.Quantity{Amount: "86646", Decimals: 9}, Currency: "USDC", CostBasis: "0.001", Proceeds: "0.00098", Amount: "-0.00002"}
		}
		if err := p.Validate(); err != nil {
			t.Fatal(err)
		}
		s := ExecutionSnapshot{Status: ObservationStatusSuccess, OMS: &ExecutionOMS{PnL: p}}
		if !s.Status.Terminal() || s.Complete() != (status != PnLStatusPending) {
			t.Fatal(status)
		}
		if status == PnLStatusRealized {
			p.Settlement = nil
		} else {
			p.Settlement = &PnLSettlement{}
		}
		if p.Validate() == nil {
			t.Fatal("accepted invalid settlement", status)
		}
	}
	for _, amount := range []string{"1e-6", "+1", "-", "0.0000000000000000001", "NaN", "01"} {
		p := ExecutionPnL{Status: PnLStatusRealized, Settlement: &PnLSettlement{AssetID: "sui", Quantity: market.Quantity{Amount: "1", Decimals: 9}, Currency: "USDC", CostBasis: "1", Proceeds: "1", Amount: amount}}
		if p.Validate() == nil {
			t.Fatal("accepted invalid amount", amount)
		}
	}
}
