package scalping_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
)

// TestScalpingOptionalResolutionAndQuantity verifies explicit units and optional instruments.
//
// Version:
//   - 2026-09-26: Added.
func TestScalpingOptionalResolutionAndQuantity(t *testing.T) {
	request := strings.Replace(validRequest, `,"chain":"sui","poolId":"pool-1"`, "", 1)
	for _, quantity := range []string{"", `,"buy":{"quantity":{"amount":"1000000000","decimals":9}}`, `,"buy":{"quantity":{"amount":"1","decimals":0}}`} {
		var p scalping.Params
		if err := json.Unmarshal([]byte(strings.TrimSuffix(request, "}")+quantity+"}"), &p); err != nil {
			t.Fatal(err)
		}
		if p.Markets[0].PoolID != "" || p.Markets[0].VenueSymbol != "" {
			t.Fatal("instrument filter was invented")
		}
		copy := p.Normalize()
		if copy.Buy != nil {
			copy.Buy.Quantity.Amount = "2"
			if p.Buy.Quantity.Amount == "2" {
				t.Fatal("normalization shared request quantity")
			}
		}
	}
	for _, quantity := range []string{
		`{"amount":"1"}`, `{"amount":"1","decimals":null}`, `{"amount":"0","decimals":9}`,
		`{"amount":"-1","decimals":9}`, `{"amount":"1.5","decimals":9}`, `{"amount":"1e9","decimals":9}`,
		`{"amount":"1","decimals":256}`, `{"amount":"1","decimals":1.5}`, `{"amount":1,"decimals":9}`,
		`{"amount":"1","decimals":9,"extra":true}`,
	} {
		var p scalping.Params
		err := json.Unmarshal([]byte(strings.TrimSuffix(request, "}")+`,"buy":{"quantity":`+quantity+"}}"), &p)
		if !errors.Is(err, apperror.InvalidParameter()) {
			t.Errorf("invalid quantity accepted: %s: %v", quantity, err)
		}
	}
	zero := market.Quantity{Amount: "0", Decimals: 6}
	if err := zero.Validate(); err != nil {
		t.Fatalf("zero output quantity rejected: %v", err)
	}
}

// TestDirectionalQuantities verifies independent units, strict fields and normalized ownership.
//
// Version:
//   - 2026-09-26: Added.
func TestDirectionalQuantities(t *testing.T) {
	raw := strings.TrimSuffix(validRequest, "}") + `,"buy":{"quantity":{"amount":"100000000","decimals":6}},"sell":{"quantity":{"amount":"1000000000","decimals":9}}}`
	var p scalping.Params
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatal(err)
	}
	if p.Buy.Quantity.Amount != "100000000" || p.Sell.Quantity.Decimals != 9 {
		t.Fatal("direction or scale lost")
	}
	clone := p.Normalize()
	clone.Sell.Quantity.Amount = "2"
	if p.Sell.Quantity.Amount != "1000000000" {
		t.Fatal("sell quantity aliased")
	}
	buyOnly, sellOnly := p.Normalize(), p.Normalize()
	buyOnly.Sell = nil
	sellOnly.Buy, sellOnly.Sell = nil, buyOnly.Buy
	a, err := buyOnly.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	b, err := sellOnly.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("buy and sell input conflated")
	}
	for _, extra := range []string{`,"baseQuantity":{"amount":"1","decimals":0}`, `,"buy":{"amount":"1","decimals":0}`, `,"sell":{"quantity":{"amount":"0","decimals":9}}`, `,"buy":{"quantity":{"amount":"1","decimals":0},"unknown":1}`} {
		var invalid scalping.Params
		if err := json.Unmarshal([]byte(strings.TrimSuffix(validRequest, "}")+extra+"}"), &invalid); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid direction accepted: %v", err)
		}
	}
	empty := p.Normalize()
	empty.Buy, empty.Sell = nil, &scalping.SideParams{}
	n := empty.Normalize()
	if n.Sell != nil {
		t.Fatal("empty side was not canonicalized")
	}
}
