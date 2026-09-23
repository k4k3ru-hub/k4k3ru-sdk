package scalping

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
)

func pointer[T any](value T) *T { return &value }

func spotParams() Params {
	return Params{
		MarketType: rule.MarketTypeSpot,
		BaseAsset:  rule.AssetRef{Chain: "sui", Network: "mainnet", AssetID: "0x2::sui::SUI"},
		QuoteAsset: rule.AssetRef{Chain: "sui", Network: "mainnet", AssetID: "usdc-asset-id"},
		Markets:    []rule.MarketRef{{Venue: "cetus", Chain: "sui", Network: "mainnet", PoolID: "sui-usdc-pool"}},
		Conditions: Conditions{WindowMS: 30000, MaximumDataAgeMS: 2000, TradeCount: &CountRange{Minimum: pointer(uint64(3))}},
		ExecutionRule: rule.Rule{
			Open:  rule.OpenRule{Spot: &rule.SpotOpenRule{Amount: "1000000"}, LimitPrice: pointer("2"), MaximumSlippageBPS: pointer(uint64(0)), ExecutionTTLMS: 30000},
			Close: rule.CloseRule{TakeProfit: &rule.Trigger{Type: rule.TriggerTypePrice, Value: "2.1"}, StopLoss: &rule.Trigger{Type: rule.TriggerTypePrice, Value: "1.9"}, MaximumSlippageBPS: pointer(uint64(100)), ExecutionTTLMS: 30000, Spot: &rule.SpotCloseRule{Markets: []rule.MarketRef{{Venue: "uniswap-v3", Chain: "base", Network: "mainnet", PoolID: "other-chain-pool"}}}},
		},
	}
}

func perpParams() Params {
	p := spotParams()
	p.MarketType = rule.MarketTypePerp
	p.Markets = []rule.MarketRef{{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: "SUI"}}
	p.ExecutionRule.Open.Spot = nil
	p.ExecutionRule.Open.Perp = &rule.PerpOpenRule{Side: rule.PositionSideShort, Quantity: "1000000000", Leverage: 1, MarginMode: rule.MarginModeIsolated}
	p.ExecutionRule.Close.Spot = nil
	p.ExecutionRule.Close.TakeProfit.Value = "1.9"
	p.ExecutionRule.Close.StopLoss.Value = "2.1"
	return p
}

// TestParamsRoundTrip verifies both market types and explicit zero slippage.
//
// Version:
//   - 2026-09-23: Added.
func TestParamsRoundTrip(t *testing.T) {
	for _, p := range []Params{spotParams(), perpParams()} {
		if err := p.Validate(); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), `"maximumSlippageBps":0`) {
			t.Fatal("explicit zero was omitted")
		}
		var decoded Params
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(p, decoded) {
			t.Fatal("request changed on round trip")
		}
	}
}

// TestParamsValidation verifies conflicting rules and exact numeric bounds.
//
// Version:
//   - 2026-09-23: Added.
func TestParamsValidation(t *testing.T) {
	tests := map[string]func(*Params){
		"missing type":      func(p *Params) { p.MarketType = "" },
		"wrong type":        func(p *Params) { p.MarketType = rule.MarketTypePerp },
		"mixed variants":    func(p *Params) { p.ExecutionRule.Open.Perp = perpParams().ExecutionRule.Open.Perp },
		"no close":          func(p *Params) { p.ExecutionRule.Close = rule.CloseRule{} },
		"no trigger":        func(p *Params) { p.ExecutionRule.Close.TakeProfit = nil; p.ExecutionRule.Close.StopLoss = nil },
		"no close market":   func(p *Params) { p.ExecutionRule.Close.Spot = nil },
		"empty markets":     func(p *Params) { p.Markets = nil },
		"same assets":       func(p *Params) { p.QuoteAsset = p.BaseAsset },
		"no slippage":       func(p *Params) { p.ExecutionRule.Open.MaximumSlippageBPS = nil },
		"invalid slippage":  func(p *Params) { p.ExecutionRule.Close.MaximumSlippageBPS = pointer(uint64(10001)) },
		"no ttl":            func(p *Params) { p.ExecutionRule.Open.ExecutionTTLMS = 0 },
		"zero holding":      func(p *Params) { p.ExecutionRule.Close.MaximumHoldingMS = pointer(uint64(0)) },
		"exponent amount":   func(p *Params) { p.ExecutionRule.Open.Spot.Amount = "1e6" },
		"fractional amount": func(p *Params) { p.ExecutionRule.Open.Spot.Amount = "1.1" },
		"zero amount":       func(p *Params) { p.ExecutionRule.Open.Spot.Amount = "0" },
		"negative amount":   func(p *Params) { p.ExecutionRule.Open.Spot.Amount = "-1" },
		"no limit":          func(p *Params) { p.ExecutionRule.Open.LimitPrice = pointer("") },
		"invalid price":     func(p *Params) { p.ExecutionRule.Close.TakeProfit.Value = "NaN" },
		"no threshold":      func(p *Params) { p.Conditions.TradeCount = nil },
		"empty range":       func(p *Params) { p.Conditions.PriceChangeBPS = &DecimalRange{} },
		"fractional volume": func(p *Params) { p.Conditions.QuoteVolume = &IntegerRange{Minimum: pointer("0.5")} },
		"reversed exact range": func(p *Params) {
			p.Conditions.PriceChangeBPS = &DecimalRange{Minimum: pointer("9007199254740993"), Maximum: pointer("9007199254740992")}
		},
		"reversed count": func(p *Params) {
			p.Conditions.TradeCount = &CountRange{Minimum: pointer(uint64(2)), Maximum: pointer(uint64(1))}
		},
		"invalid ratio": func(p *Params) { p.Conditions.BuyVolumeRatioBPS = &DecimalRange{Minimum: pointer("10000.01")} },
		"no window":     func(p *Params) { p.Conditions.WindowMS = 0 },
		"no age":        func(p *Params) { p.Conditions.MaximumDataAgeMS = 0 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			p := spotParams()
			mutate(&p)
			if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("want validation error, got %v", err)
			}
		})
	}
	p := spotParams()
	p.ExecutionRule.Open.Spot.Amount = "9007199254740993"
	p.Conditions.PriceChangeBPS = &DecimalRange{Maximum: pointer("-0.25")}
	p.Conditions.BuyVolumeRatioBPS = &DecimalRange{Minimum: pointer("0"), Maximum: pointer("10000")}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Params){
		func(p *Params) { p.ExecutionRule.Open.Perp.Leverage = 0 },
		func(p *Params) { p.ExecutionRule.Open.Perp.MarginMode = "automatic" },
		func(p *Params) { p.ExecutionRule.Close.Spot = spotParams().ExecutionRule.Close.Spot },
	} {
		p := perpParams()
		mutate(&p)
		if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatal("invalid perp accepted")
		}
	}
}

// TestParamsJSONRejectsOverrides verifies required objects and immutable failures.
//
// Version:
//   - 2026-09-23: Added.
func TestParamsJSONRejectsOverrides(t *testing.T) {
	data, err := json.Marshal(spotParams())
	if err != nil {
		t.Fatal(err)
	}
	for _, alter := range []func(map[string]any){
		func(m map[string]any) { delete(m, "marketType") },
		func(m map[string]any) { m["marketType"] = nil },
		func(m map[string]any) { delete(m["executionRule"].(map[string]any), "close") },
		func(m map[string]any) { m["executionRule"].(map[string]any)["close"] = nil },
		func(m map[string]any) { m["executionRule"].(map[string]any)["close"] = map[string]any{} },
		func(m map[string]any) { m["executionRule"].(map[string]any)["marketType"] = "spot" },
		func(m map[string]any) {
			m["executionRule"].(map[string]any)["open"].(map[string]any)["marketType"] = "spot"
		},
		func(m map[string]any) {
			m["executionRule"].(map[string]any)["close"].(map[string]any)["limitPrice"] = "1"
		},
		func(m map[string]any) { m["conditions"].(map[string]any)["unexpected"] = true },
	} {
		var object map[string]any
		if err := json.Unmarshal(data, &object); err != nil {
			t.Fatal(err)
		}
		alter(object)
		wire, err := json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		previous := spotParams()
		decoded := previous.Normalize()
		if err := json.Unmarshal(wire, &decoded); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid json accepted: %v", err)
		}
		if !reflect.DeepEqual(previous, decoded) {
			t.Fatal("failed decode changed destination")
		}
	}
	for _, wire := range []string{"null", "[]", "{}", strings.Replace(string(data), `"marketType":"spot"`, `"marketType":"spot","MarketType":"perp"`, 1)} {
		var decoded Params
		if err := json.Unmarshal([]byte(wire), &decoded); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("accepted invalid object: %v", err)
		}
	}
}

// TestNormalizeDoesNotAlias verifies request copies and absence of trading defaults.
//
// Version:
//   - 2026-09-23: Added.
func TestNormalizeDoesNotAlias(t *testing.T) {
	p := spotParams()
	p.MarketType = " SPOT "
	p.Conditions.PriceChangeBPS = &DecimalRange{Minimum: pointer(" 0.25 ")}
	before, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	n := p.Normalize()
	if n.MarketType != rule.MarketTypeSpot || *n.Conditions.PriceChangeBPS.Minimum != "0.25" {
		t.Fatal("normalization failed")
	}
	n.Markets[0].PoolID = "changed"
	n.ExecutionRule.Close.Spot.Markets[0].PoolID = "changed"
	n.ExecutionRule.Close.TakeProfit.Value = "changed"
	*n.ExecutionRule.Open.LimitPrice = "changed"
	*n.ExecutionRule.Open.MaximumSlippageBPS = 50
	*n.Conditions.PriceChangeBPS.Minimum = "changed"
	*n.Conditions.TradeCount.Minimum = 100
	after, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("normalization aliased input")
	}
	empty := (Params{}).Normalize()
	if empty.ExecutionRule.Open.MaximumSlippageBPS != nil || empty.Conditions.PriceChangeBPS != nil || empty.Conditions.WindowMS != 0 {
		t.Fatal("defaults were invented")
	}
}
