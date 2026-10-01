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
	observations "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
)

func runConfig() RunConfiguration {
	return RunConfiguration{
		MarketType: market.MarketTypeSpot, Symbol: market.SUIUSDC,
		Observation: Observation{Markets: []MarketSelector{{Venue: "cetus", Chain: "sui"}}},
		ExecutionRule: ExecutionRule{
			Markets: []ExecutionMarket{{MarketSelector: MarketSelector{Venue: "cetus", Chain: "sui"}, AccountAddress: "wallet"}},
			Entry: EntryRule{Side: SideBuy, MaximumQuantity: market.Quantity{Amount: "100", Decimals: 1},
				Condition: Condition{PriceChangeBPS: &DecimalRange{Minimum: pointer("20")}}},
			Exit: ExitRule{MaximumHoldingMS: pointer(uint64(60000))},
		},
	}
}

func runExample(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestRunExamples verifies the documented start payloads and canonical persistence.
//
// Version:
//   - 2026-10-01: Added.
func TestRunExamples(t *testing.T) {
	for _, name := range []string{"run_spot_buy", "run_spot_sell", "run_perpetual_sell"} {
		t.Run(name, func(t *testing.T) {
			var request RunParams
			if err := json.Unmarshal(runExample(t, name), &request); err != nil {
				t.Fatal(err)
			}
			if request.Params == nil || !request.Params.TestMode {
				t.Fatal("missing test configuration")
			}
			data, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), `"Params"`) || strings.Contains(string(data), `"params"`) {
				t.Fatal("configuration was nested")
			}
			var restored RunParams
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(request, restored) {
				t.Fatal("request changed during round trip")
			}
			configData, err := json.Marshal(request.Params)
			if err != nil {
				t.Fatal(err)
			}
			var config RunConfiguration
			if err := json.Unmarshal(configData, &config); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(*request.Params, config) {
				t.Fatal("saved settings changed")
			}
		})
	}
}

// TestRunDefaultsAndIsolation verifies omitted defaults, explicit zeros and deep copies.
//
// Version:
//   - 2026-10-01: Added.
func TestRunDefaultsAndIsolation(t *testing.T) {
	p := runConfig()
	n := p.Normalize()
	r := n.ExecutionRule
	if n.TestMode || *n.Observation.WindowMS != 60000 || *r.MinimumOrderIntervalMS != 1000 || *r.MaximumUnsettledOrders != 1 || *r.MaximumSlippageBPS != 50 || *r.ReserveBufferBPS != 100 || *r.ExecutionTTLMS != 30000 {
		t.Fatal("unexpected defaults")
	}
	if n.Observation.Buy != nil || n.Observation.Sell != nil || n.Observation.MaximumTradeAgeMS != nil || n.Observation.MaximumSnapshotAgeMS != nil {
		t.Fatal("observation settings inferred")
	}
	if p.Observation.WindowMS != nil || p.ExecutionRule.MinimumOrderIntervalMS != nil {
		t.Fatal("normalization mutated source")
	}
	p.ExecutionRule.MinimumOrderIntervalMS, p.ExecutionRule.ReserveBufferBPS, p.ExecutionRule.MaximumSlippageBPS = pointer(uint64(0)), pointer(uint64(0)), pointer(uint64(0))
	n = p.Normalize()
	if err := n.Validate(); err != nil {
		t.Fatal(err)
	}
	if *n.ExecutionRule.MinimumOrderIntervalMS != 0 || *n.ExecutionRule.ReserveBufferBPS != 0 || *n.ExecutionRule.MaximumSlippageBPS != 0 {
		t.Fatal("explicit zero defaulted")
	}

	// Fill every pointer/slice family, then mutate only the detached copy.
	p.Observation.Buy = &observations.SideParams{Quantity: &market.Quantity{Amount: "10", Decimals: 0}}
	p.Observation.Sell = &observations.SideParams{Quantity: &market.Quantity{Amount: "20", Decimals: 0}}
	p.Observation.MaximumTradeAgeMS, p.Observation.MaximumSnapshotAgeMS = pointer(uint64(10)), pointer(uint64(20))
	p.ExecutionRule.Markets[0].Perpetual = &PerpetualSettings{Leverage: 3, MarginMode: rule.MarginModeCross}
	p.ExecutionRule.Entry.LimitPrice = pointer("2")
	c := Condition{QuoteVolume: &QuantityRange{Minimum: &market.Quantity{Amount: "1", Decimals: 0}}, TradeCount: &CountRange{Maximum: pointer(uint64(4))}, SpreadBPS: &DecimalRange{Maximum: pointer("10")}}
	p.ExecutionRule.Exit.Condition = &c
	p.ExecutionRule.Exit.TakeProfit = &rule.Trigger{Type: rule.TriggerTypeReturnBPS, Value: "1"}
	n = p.Normalize()
	n.Observation.Markets[0].Venue = "turbos"
	n.Observation.Buy.Quantity.Amount, n.Observation.Sell.Quantity.Amount = "100", "200"
	*n.Observation.MaximumTradeAgeMS, *n.Observation.MaximumSnapshotAgeMS = 100, 200
	n.ExecutionRule.Markets[0].AccountAddress = "other"
	n.ExecutionRule.Markets[0].Perpetual.Leverage = 4
	*n.ExecutionRule.Entry.LimitPrice = "3"
	*n.ExecutionRule.Entry.Condition.PriceChangeBPS.Minimum = "30"
	n.ExecutionRule.Exit.Condition.QuoteVolume.Minimum.Amount = "3"
	*n.ExecutionRule.Exit.Condition.TradeCount.Maximum = 5
	*n.ExecutionRule.Exit.Condition.SpreadBPS.Maximum = "11"
	n.ExecutionRule.Exit.TakeProfit.Value = "2"
	*n.ExecutionRule.Exit.MaximumHoldingMS = 1
	*n.ExecutionRule.MinimumOrderIntervalMS = 99
	if p.Observation.Markets[0].Venue != "cetus" || p.Observation.Buy.Quantity.Amount != "10" || p.Observation.Sell.Quantity.Amount != "20" || *p.Observation.MaximumTradeAgeMS != 10 || *p.Observation.MaximumSnapshotAgeMS != 20 || p.ExecutionRule.Markets[0].AccountAddress != "wallet" || p.ExecutionRule.Markets[0].Perpetual.Leverage != 3 || *p.ExecutionRule.Entry.LimitPrice != "2" || *p.ExecutionRule.Entry.Condition.PriceChangeBPS.Minimum != "20" || c.QuoteVolume.Minimum.Amount != "1" || *c.TradeCount.Maximum != 4 || *c.SpreadBPS.Maximum != "10" || p.ExecutionRule.Exit.TakeProfit.Value != "1" || *p.ExecutionRule.Exit.MaximumHoldingMS != 60000 || *p.ExecutionRule.MinimumOrderIntervalMS != 0 {
		t.Fatal("mutable fields alias the original configuration")
	}
}

// TestRunValidation rejects malformed execution settings at the request boundary.
//
// Version:
//   - 2026-10-01: Added.
func TestRunValidation(t *testing.T) {
	tests := map[string]func(*RunConfiguration){
		"legacy perp":             func(p *RunConfiguration) { p.MarketType = "perp" },
		"missing symbol":          func(p *RunConfiguration) { p.Symbol = "" },
		"same assets":             func(p *RunConfiguration) { p.Symbol = "SUI/SUI" },
		"multiple pairs":          func(p *RunConfiguration) { p.Symbol = "SUI/USDC,BTC" },
		"empty observation":       func(p *RunConfiguration) { p.Observation.Markets = nil },
		"empty execution markets": func(p *RunConfiguration) { p.ExecutionRule.Markets = nil },
		"unobserved venue":        func(p *RunConfiguration) { p.ExecutionRule.Markets[0].Venue = "turbos" },
		"unobserved chain":        func(p *RunConfiguration) { p.ExecutionRule.Markets[0].Chain = "base" },
		"different explicit pool": func(p *RunConfiguration) {
			p.Observation.Markets[0].PoolID = "poolA"
			p.ExecutionRule.Markets[0].PoolID = "poolB"
		},
		"duplicate market": func(p *RunConfiguration) {
			p.Observation.Markets = append(p.Observation.Markets, p.Observation.Markets[0])
		},
		"duplicate account routing": func(p *RunConfiguration) {
			p.ExecutionRule.Markets = append(p.ExecutionRule.Markets, p.ExecutionRule.Markets[0])
			p.ExecutionRule.Markets[1].AccountAddress = "other"
		},
		"two identifiers": func(p *RunConfiguration) {
			p.Observation.Markets[0].PoolID = "pool"
			p.Observation.Markets[0].VenueSymbol = "SUI"
		},
		"missing account": func(p *RunConfiguration) { p.ExecutionRule.Markets[0].AccountAddress = " " },
		"spot perp settings": func(p *RunConfiguration) {
			p.ExecutionRule.Markets[0].Perpetual = &PerpetualSettings{Leverage: 1, MarginMode: rule.MarginModeCross}
		},
		"missing perp settings": func(p *RunConfiguration) { p.MarketType = market.MarketTypePerpetual },
		"perp zero leverage": func(p *RunConfiguration) {
			p.MarketType = market.MarketTypePerpetual
			p.ExecutionRule.Markets[0].Perpetual = &PerpetualSettings{MarginMode: rule.MarginModeCross}
		},
		"perp missing margin": func(p *RunConfiguration) {
			p.MarketType = market.MarketTypePerpetual
			p.ExecutionRule.Markets[0].Perpetual = &PerpetualSettings{Leverage: 1}
		},
		"side long":             func(p *RunConfiguration) { p.ExecutionRule.Entry.Side = "long" },
		"zero maximum":          func(p *RunConfiguration) { p.ExecutionRule.Entry.MaximumQuantity.Amount = "000" },
		"fractional amount":     func(p *RunConfiguration) { p.ExecutionRule.Entry.MaximumQuantity.Amount = "0.1" },
		"empty entry condition": func(p *RunConfiguration) { p.ExecutionRule.Entry.Condition = Condition{} },
		"empty exit":            func(p *RunConfiguration) { p.ExecutionRule.Exit = ExitRule{} },
		"empty exit condition":  func(p *RunConfiguration) { p.ExecutionRule.Exit.Condition = &Condition{} },
		"zero limit":            func(p *RunConfiguration) { p.ExecutionRule.Entry.LimitPrice = pointer("0") },
		"zero observation quantity": func(p *RunConfiguration) {
			p.Observation.Buy = &observations.SideParams{Quantity: &market.Quantity{Amount: "0", Decimals: 0}}
		},
		"zero window":          func(p *RunConfiguration) { p.Observation.WindowMS = pointer(uint64(0)) },
		"long window":          func(p *RunConfiguration) { p.Observation.WindowMS = pointer(uint64(60001)) },
		"zero trade age":       func(p *RunConfiguration) { p.Observation.MaximumTradeAgeMS = pointer(uint64(0)) },
		"zero snapshot age":    func(p *RunConfiguration) { p.Observation.MaximumSnapshotAgeMS = pointer(uint64(0)) },
		"zero holding":         func(p *RunConfiguration) { p.ExecutionRule.Exit.MaximumHoldingMS = pointer(uint64(0)) },
		"zero ttl":             func(p *RunConfiguration) { p.ExecutionRule.ExecutionTTLMS = pointer(uint64(0)) },
		"zero unsettled":       func(p *RunConfiguration) { p.ExecutionRule.MaximumUnsettledOrders = pointer(uint64(0)) },
		"100 percent slippage": func(p *RunConfiguration) { p.ExecutionRule.MaximumSlippageBPS = pointer(uint64(10000)) },
		"overflow duration":    func(p *RunConfiguration) { p.ExecutionRule.MinimumOrderIntervalMS = pointer(MaximumRunDurationMS + 1) },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			p := runConfig()
			change(&p)
			if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("expected invalid parameter, got %v", err)
			}
			if _, err := json.Marshal(p); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("invalid configuration serialized: %v", err)
			}
		})
	}
	// Unknown symbols are not rejected by a catalog whitelist; broad selectors
	// may resolve to the same set as explicit selectors, in either direction.
	p := runConfig()
	p.Symbol = "NEW/QUOTE"
	p.ExecutionRule.Markets[0].PoolID = "PoolCaseSensitive"
	p.ExecutionRule.ReserveBufferBPS = pointer(uint64(20000))
	p.ExecutionRule.MaximumSlippageBPS = pointer(uint64(9999))
	p.ExecutionRule.ExecutionTTLMS = pointer(MaximumRunDurationMS)
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Observation.Markets[0].PoolID, p.ExecutionRule.Markets[0].PoolID = "PoolCaseSensitive", ""
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestRunTriggerRules verifies zero normalization, signs and side-aware ordering.
//
// Version:
//   - 2026-10-01: Added.
func TestRunTriggerRules(t *testing.T) {
	trigger := func(kind rule.TriggerType, value string) *rule.Trigger {
		return &rule.Trigger{Type: kind, Value: value}
	}
	for _, tt := range []struct {
		name   string
		side   Side
		tp, sl *rule.Trigger
		valid  bool
	}{
		{"returns buy", SideBuy, trigger(rule.TriggerTypeReturnBPS, "0.1"), trigger(rule.TriggerTypeReturnBPS, "-0.1"), true},
		{"returns sell", SideSell, trigger(rule.TriggerTypeReturnBPS, "20000"), trigger(rule.TriggerTypeReturnBPS, "-20000"), true},
		{"negative tp", SideBuy, trigger(rule.TriggerTypeReturnBPS, "-1"), nil, false},
		{"positive sl", SideSell, nil, trigger(rule.TriggerTypeReturnBPS, "1"), false},
		{"price buy", SideBuy, trigger(rule.TriggerTypePrice, "3"), trigger(rule.TriggerTypePrice, "2"), true},
		{"price sell", SideSell, trigger(rule.TriggerTypePrice, "2"), trigger(rule.TriggerTypePrice, "3"), true},
		{"inverted buy", SideBuy, trigger(rule.TriggerTypePrice, "2"), trigger(rule.TriggerTypePrice, "3"), false},
		{"inverted sell", SideSell, trigger(rule.TriggerTypePrice, "3"), trigger(rule.TriggerTypePrice, "2"), false},
		{"equal prices", SideBuy, trigger(rule.TriggerTypePrice, "2.0"), trigger(rule.TriggerTypePrice, "2"), false},
		{"mixed types", SideBuy, trigger(rule.TriggerTypePrice, "0.1"), trigger(rule.TriggerTypeReturnBPS, "-100"), true},
		{"mixed sell", SideSell, trigger(rule.TriggerTypeReturnBPS, "100"), trigger(rule.TriggerTypePrice, "1"), true},
		{"zero price", SideBuy, trigger(rule.TriggerTypePrice, "0"), nil, false},
		{"unknown zero", SideBuy, trigger("other", "0"), nil, false},
		{"malformed zero", SideBuy, trigger(rule.TriggerTypeReturnBPS, "0e0"), nil, false},
		{"empty value", SideBuy, trigger(rule.TriggerTypeReturnBPS, ""), nil, false},
		{"exact decimal order", SideBuy, trigger(rule.TriggerTypePrice, "2.000000000000000001"), trigger(rule.TriggerTypePrice, "2"), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := runConfig()
			p.ExecutionRule.Entry.Side = tt.side
			p.ExecutionRule.Exit.TakeProfit, p.ExecutionRule.Exit.StopLoss = tt.tp, tt.sl
			err := p.Validate()
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%t: %v", tt.valid, err)
			}
		})
	}
	for _, zero := range []string{"0", "0.0", "-0", "-0.000"} {
		p := runConfig()
		p.ExecutionRule.Exit = ExitRule{TakeProfit: trigger(rule.TriggerTypeReturnBPS, zero), StopLoss: trigger(rule.TriggerTypeReturnBPS, zero)}
		if err := p.Validate(); err == nil {
			t.Fatal("zero-only exit accepted")
		}
		p.ExecutionRule.Exit.Condition = &Condition{TradeCount: &CountRange{Minimum: pointer(uint64(0))}}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "takeProfit") || strings.Contains(string(data), "stopLoss") {
			t.Fatal("zero trigger persisted")
		}
		if p.ExecutionRule.Exit.TakeProfit == nil {
			t.Fatal("encoding mutated trigger")
		}
		var restored RunConfiguration
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatal(err)
		}
		if restored.ExecutionRule.Exit.TakeProfit != nil || restored.ExecutionRule.Exit.StopLoss != nil {
			t.Fatal("disabled trigger revived")
		}
	}
}

// TestRunConditions verifies all nine metrics and exact range comparisons.
//
// Version:
//   - 2026-10-01: Added.
func TestRunConditions(t *testing.T) {
	for _, tt := range []struct {
		name      string
		condition Condition
		valid     bool
	}{
		{"price", Condition{PriceChangeBPS: &DecimalRange{Minimum: pointer("-20000"), Maximum: pointer("20000")}}, true},
		{"volume", Condition{QuoteVolume: &QuantityRange{Minimum: &market.Quantity{Amount: "1", Decimals: 0}, Maximum: &market.Quantity{Amount: "10", Decimals: 1}}}, true},
		{"count", Condition{TradeCount: &CountRange{Maximum: pointer(uint64(0))}}, true},
		{"ratio", Condition{BuyVolumeRatioBPS: &DecimalRange{Minimum: pointer("0"), Maximum: pointer("10000")}}, true},
		{"price delta", Condition{PriceChangeDeltaBPS: &DecimalRange{Maximum: pointer("-10")}}, true},
		{"volume change", Condition{QuoteVolumeChangeBPS: &DecimalRange{Minimum: pointer("10001")}}, true},
		{"ratio delta", Condition{BuyVolumeRatioDeltaBPS: &DecimalRange{Minimum: pointer("-10000"), Maximum: pointer("10000")}}, true},
		{"volatility", Condition{RealizedVolatilityBPS: &DecimalRange{Minimum: pointer("0")}}, true},
		{"spread", Condition{SpreadBPS: &DecimalRange{Minimum: pointer("-1")}}, true},
		{"empty", Condition{}, false},
		{"empty decimal", Condition{SpreadBPS: &DecimalRange{}}, false},
		{"empty volume", Condition{QuoteVolume: &QuantityRange{}}, false},
		{"empty count", Condition{TradeCount: &CountRange{}}, false},
		{"negative volatility", Condition{RealizedVolatilityBPS: &DecimalRange{Maximum: pointer("-1")}}, false},
		{"high ratio", Condition{BuyVolumeRatioBPS: &DecimalRange{Maximum: pointer("10000.0001")}}, false},
		{"low ratio delta", Condition{BuyVolumeRatioDeltaBPS: &DecimalRange{Minimum: pointer("-10000.0001")}}, false},
		{"reversed precise decimal", Condition{PriceChangeBPS: &DecimalRange{Minimum: pointer("9007199254740993"), Maximum: pointer("9007199254740992")}}, false},
		{"reversed scales", Condition{QuoteVolume: &QuantityRange{Minimum: &market.Quantity{Amount: "10", Decimals: 0}, Maximum: &market.Quantity{Amount: "10", Decimals: 1}}}, false},
		{"reversed count", Condition{TradeCount: &CountRange{Minimum: pointer(uint64(2)), Maximum: pointer(uint64(1))}}, false},
		{"exponent", Condition{PriceChangeBPS: &DecimalRange{Minimum: pointer("1e2")}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.condition.Validate()
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%t: %v", tt.valid, err)
			}
		})
	}
}
