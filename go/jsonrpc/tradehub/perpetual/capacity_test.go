package perpetual

import (
	"encoding/json"
	"testing"
)

// TestTradingCapacityWire preserves optional omission, separate directions and decimal values.
//
// Version:
//   - 2026-10-04: Added.
func TestTradingCapacityWire(t *testing.T) {
	r := AccountResult{TradingCapacity: &TradingCapacity{Buy: TradingSideCapacity{MaximumQuantity: "0", AvailableCollateral: "-0.1"}, Sell: TradingSideCapacity{MaximumQuantity: "20.5", AvailableCollateral: "30.01"}, Leverage: 3, MarginMode: "isolated", MarkPrice: "2.001", Contract: Contract{Coin: "SUI", SizeDecimals: 1, MaxLeverage: 10}, ObservedAt: 1000}}
	if err := r.TradingCapacity.Validate(); err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var got AccountResult
	if err := json.Unmarshal(wire, &got); err != nil || got.TradingCapacity == nil || *got.TradingCapacity != *r.TradingCapacity {
		t.Fatal("capacity changed on wire", err)
	}
	for _, change := range []func(*TradingCapacity){func(c *TradingCapacity) { c.Buy.MaximumQuantity = "-1" }, func(c *TradingCapacity) { c.Sell.AvailableCollateral = "NaN" }, func(c *TradingCapacity) { c.MarkPrice = "0" }, func(c *TradingCapacity) { c.ObservedAt = 0 }, func(c *TradingCapacity) { c.MarginMode = "unknown" }, func(c *TradingCapacity) { c.Leverage = 11 }} {
		c := *r.TradingCapacity
		change(&c)
		if c.Validate() == nil {
			t.Fatal("invalid capacity accepted")
		}
	}
	if err := json.Unmarshal([]byte(`{"accountMode":"disabled"}`), &got); err != nil {
		t.Fatal(err)
	}
	var legacy AccountResult
	if err := json.Unmarshal([]byte(`{"accountMode":"disabled"}`), &legacy); err != nil || legacy.TradingCapacity != nil {
		t.Fatal("missing capacity invented", err)
	}
}
