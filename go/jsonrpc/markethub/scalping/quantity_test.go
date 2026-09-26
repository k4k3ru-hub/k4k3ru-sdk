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
	for _, quantity := range []string{"", `,"baseQuantity":{"amount":"1000000000","decimals":9}`, `,"baseQuantity":{"amount":"1","decimals":0}`} {
		var p scalping.Params
		if err := json.Unmarshal([]byte(strings.TrimSuffix(request, "}")+quantity+"}"), &p); err != nil {
			t.Fatal(err)
		}
		if p.Markets[0].PoolID != "" || p.Markets[0].VenueSymbol != "" {
			t.Fatal("instrument filter was invented")
		}
		copy := p.Normalize()
		if copy.BaseQuantity != nil {
			copy.BaseQuantity.Amount = "2"
			if p.BaseQuantity.Amount == "2" {
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
		err := json.Unmarshal([]byte(strings.TrimSuffix(request, "}")+`,"baseQuantity":`+quantity+"}"), &p)
		if !errors.Is(err, apperror.InvalidParameter()) {
			t.Errorf("invalid quantity accepted: %s: %v", quantity, err)
		}
	}
	zero := market.Quantity{Amount: "0", Decimals: 6}
	if err := zero.Validate(); err != nil {
		t.Fatalf("zero output quantity rejected: %v", err)
	}
}
