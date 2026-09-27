package swap

import (
	"encoding/json"
	"errors"
	"testing"

	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// TestPrepareWithoutSimulationRequiresExplicitLimits checks the default and its constraints.
//
// Version:
//   - 2026-09-27: Added.
func TestPrepareWithoutSimulationRequiresExplicitLimits(t *testing.T) {
	p := validPrepareParams()
	p.Simulate, p.MaximumSlippageBPS = false, nil
	p.AmountLimit, p.EVM = "000900", &EVMPrepareParams{GasLimit: 150000}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded PrepareParams
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Simulate || decoded.Normalize().AmountLimit != "900" {
		t.Fatal("incorrect default or limit")
	}
	if err := decoded.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*PrepareParams){
		"missing amount limit":     func(p *PrepareParams) { p.AmountLimit = "" },
		"zero amount limit":        func(p *PrepareParams) { p.AmountLimit = "0" },
		"missing gas limit":        func(p *PrepareParams) { p.EVM = nil },
		"zero gas limit":           func(p *PrepareParams) { p.EVM = &EVMPrepareParams{} },
		"slippage without a quote": func(p *PrepareParams) { v := uint64(100); p.MaximumSlippageBPS = &v },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := decoded
			mutate(&invalid)
			if err := invalid.Validate(); !errors.Is(err, app.InvalidParameter()) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
	sui := suiPrepareParams()
	sui.Simulate, sui.MaximumSlippageBPS, sui.AmountLimit = false, nil, "12"
	if err := sui.Validate(); err != nil {
		t.Fatal(err)
	}
	sui.AmountLimit = "18446744073709551616"
	if err := sui.Validate(); err == nil {
		t.Fatal("accepted overflowing Sui amount limit")
	}
}

// TestPrepareResultOmitsUnknownAmounts prevents reporting an unsimulated estimate as an output.
//
// Version:
//   - 2026-09-27: Added.
func TestPrepareResultOmitsUnknownAmounts(t *testing.T) {
	r := validPrepareResult(PrepareStatusReady)
	r.Simulated, r.AmountIn, r.AmountLimit = false, "1000", "900"
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["amountOut"]; exists {
		t.Fatal("unknown output is present")
	}
	r.AmountOut = "950"
	if err := r.Validate(); err == nil {
		t.Fatal("unsimulated result has two asserted amounts")
	}
	r.Simulated = true
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.Simulated, r.AmountIn = false, ""
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	} // Exact-output request has only known output.
}
