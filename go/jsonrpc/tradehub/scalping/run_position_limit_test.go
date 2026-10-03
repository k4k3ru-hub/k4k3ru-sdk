package scalping

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
)

func perpetualRunConfig() RunConfiguration {
	p := runConfig()
	p.MarketType = market.MarketTypePerpetual
	m := MarketSelector{Venue: "hyperliquid", VenueSymbol: "SUI"}
	p.Observation.Markets = []MarketSelector{m}
	p.ExecutionRule.Markets = []ExecutionMarket{{
		MarketSelector: m, AccountAddress: "trading-account",
		Perpetual: &PerpetualSettings{Leverage: 3, MarginMode: rule.MarginModeIsolated},
	}}
	return p
}

// TestRunPositionLimitDefaults verifies market-specific defaults and persistence.
//
// Version:
//   - 2026-10-03: Added.
func TestRunPositionLimitDefaults(t *testing.T) {
	for _, side := range []Side{SideBuy, SideSell} {
		for _, tt := range []struct {
			name     string
			settings *PerpetualRunSettings
			want     market.Quantity
		}{
			{"omitted settings", nil, market.Quantity{Amount: "100", Decimals: 1}},
			{"omitted quantity", &PerpetualRunSettings{}, market.Quantity{Amount: "100", Decimals: 1}},
			{"explicit limit", &PerpetualRunSettings{MaximumPositionQuantity: &market.Quantity{Amount: "30", Decimals: 0}}, market.Quantity{Amount: "30", Decimals: 0}},
			{"below order cap", &PerpetualRunSettings{MaximumPositionQuantity: &market.Quantity{Amount: "5", Decimals: 0}}, market.Quantity{Amount: "5", Decimals: 0}},
		} {
			t.Run(string(side)+"/"+tt.name, func(t *testing.T) {
				p := perpetualRunConfig()
				p.ExecutionRule.Entry.Side = side
				p.ExecutionRule.Perpetual = tt.settings
				var originalLimit *market.Quantity
				if tt.settings != nil {
					originalLimit = tt.settings.MaximumPositionQuantity
				}
				n := p.Normalize()
				if err := n.Validate(); err != nil {
					t.Fatal(err)
				}
				if n.ExecutionRule.MaximumUnsettledOrders != nil || n.ExecutionRule.Perpetual == nil || n.ExecutionRule.Perpetual.MaximumPositionQuantity == nil || *n.ExecutionRule.Perpetual.MaximumPositionQuantity != tt.want {
					t.Fatal("incorrect Perpetual defaults")
				}
				if !reflect.DeepEqual(n, n.Normalize()) {
					t.Fatal("normalization is not idempotent")
				}
				request := RunParams{IdempotencyKey: "position-limit", Params: &p}
				encoded, err := json.Marshal(request)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(encoded), `"maximumUnsettledOrders"`) || !strings.Contains(string(encoded), `"maximumPositionQuantity"`) {
					t.Fatal("incorrect persisted limit fields")
				}
				var restored RunParams
				if err := json.Unmarshal(encoded, &restored); err != nil {
					t.Fatal(err)
				}
				if restored.Params == nil || !reflect.DeepEqual(n, *restored.Params) {
					t.Fatal("limits changed after restore")
				}
				if p.ExecutionRule.MaximumUnsettledOrders != nil || p.ExecutionRule.Perpetual != tt.settings || tt.settings != nil && tt.settings.MaximumPositionQuantity != originalLimit {
					t.Fatal("normalization mutated input")
				}
				n.ExecutionRule.Perpetual.MaximumPositionQuantity.Amount = "999"
				if p.ExecutionRule.Entry.MaximumQuantity.Amount != "100" || n.ExecutionRule.Entry.MaximumQuantity.Amount != "100" || tt.settings != nil && tt.settings.MaximumPositionQuantity != nil && *tt.settings.MaximumPositionQuantity != tt.want {
					t.Fatal("position limit aliases input or entry quantity")
				}
			})
		}
	}
	spot := runConfig().Normalize()
	if spot.ExecutionRule.Perpetual != nil || spot.ExecutionRule.MaximumUnsettledOrders == nil || *spot.ExecutionRule.MaximumUnsettledOrders != 1 {
		t.Fatal("Spot defaults changed")
	}
}

// TestRunPositionLimitValidation rejects invalid limits and cross-product fields.
//
// Version:
//   - 2026-10-03: Added.
func TestRunPositionLimitValidation(t *testing.T) {
	for name, change := range map[string]func(*RunConfiguration){
		"empty amount": func(p *RunConfiguration) { p.ExecutionRule.Perpetual.MaximumPositionQuantity.Amount = "" },
		"zero":         func(p *RunConfiguration) { p.ExecutionRule.Perpetual.MaximumPositionQuantity.Amount = "000" },
		"fraction":     func(p *RunConfiguration) { p.ExecutionRule.Perpetual.MaximumPositionQuantity.Amount = "1.5" },
		"negative":     func(p *RunConfiguration) { p.ExecutionRule.Perpetual.MaximumPositionQuantity.Amount = "-1" },
		"too long": func(p *RunConfiguration) {
			p.ExecutionRule.Perpetual.MaximumPositionQuantity.Amount = strings.Repeat("1", 385)
		},
		"order count": func(p *RunConfiguration) { p.ExecutionRule.MaximumUnsettledOrders = pointer(uint64(1)) },
		"zero count":  func(p *RunConfiguration) { p.ExecutionRule.MaximumUnsettledOrders = pointer(uint64(0)) },
		"Spot settings": func(p *RunConfiguration) {
			p.MarketType = market.MarketTypeSpot
			p.ExecutionRule.Markets[0].Perpetual = nil
		},
		"empty Spot settings": func(p *RunConfiguration) {
			p.MarketType = market.MarketTypeSpot
			p.ExecutionRule.Markets[0].Perpetual = nil
			p.ExecutionRule.Perpetual = &PerpetualRunSettings{}
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := perpetualRunConfig().Normalize()
			change(&p)
			if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("expected invalid parameter, got %v", err)
			}
			if _, err := json.Marshal(p); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("invalid configuration serialized: %v", err)
			}
		})
	}
	// Integer strings and their scale must remain exact, beyond float64 precision.
	p := perpetualRunConfig()
	p.ExecutionRule.Perpetual = &PerpetualRunSettings{MaximumPositionQuantity: &market.Quantity{Amount: "9007199254740993", Decimals: 255}}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestRunPositionLimitJSON enforces placement, strict input and atomic decoding.
//
// Version:
//   - 2026-10-03: Added.
func TestRunPositionLimitJSON(t *testing.T) {
	base := string(runExample(t, "run_perpetual_sell"))
	for _, tt := range []struct{ name, old, replacement string }{
		{"legacy order count", `"executionRule": {`, `"executionRule": {"maximumUnsettledOrders":2,`},
		{"zero order count", `"executionRule": {`, `"executionRule": {"maximumUnsettledOrders":0,`},
		{"null order count", `"executionRule": {`, `"executionRule": {"maximumUnsettledOrders":null,`},
		{"null settings", "\"perpetual\": {\n      \"maximumPositionQuantity\": {\"amount\": \"30\", \"decimals\": 0}\n    }", `"perpetual":null`},
		{"null quantity", `{"amount": "30", "decimals": 0}`, `null`},
		{"missing decimals", `{"amount": "30", "decimals": 0}`, `{"amount":"30"}`},
		{"zero quantity", `{"amount": "30", "decimals": 0}`, `{"amount":"0","decimals":0}`},
		{"duplicate limit", `"maximumPositionQuantity":`, `"maximumPositionQuantity":{"amount":"1","decimals":0},"maximumPositionQuantity":`},
		{"unknown setting", `"maximumPositionQuantity":`, `"unknown":1,"maximumPositionQuantity":`},
		{"flat position limit", `"executionRule": {`, `"executionRule": {"maximumPositionQuantity":{"amount":"30","decimals":0},`},
		{"venue position limit", `"perpetual": {"leverage":`, `"perpetual":{"maximumPositionQuantity":{"amount":"30","decimals":0},"leverage":`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(base, tt.old) {
				t.Fatal("replacement did not match")
			}
			data := strings.Replace(base, tt.old, tt.replacement, 1)
			p := RunParams{ExecutionID: "unchanged"}
			if err := json.Unmarshal([]byte(data), &p); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("expected invalid parameter, got %v", err)
			}
			if !reflect.DeepEqual(p, RunParams{ExecutionID: "unchanged"}) {
				t.Fatal("failed decode replaced request")
			}
		})
	}
	for _, settings := range []string{`{}`, `{"maximumPositionQuantity":{"amount":"30","decimals":0}}`} {
		data := strings.Replace(runStartJSON, `"executionRule":{`, `"executionRule":{"perpetual":`+settings+`,`, 1)
		var p RunParams
		if err := json.Unmarshal([]byte(data), &p); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("Spot accepted Perpetual settings: %v", err)
		}
	}
}
