package carry

import (
	"encoding/json"
	"errors"
	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"testing"
)

func TestCarryContractValidationAndIdentity(t *testing.T) {
	base := Params{Symbol: "BTC/USDC", BaseAsset: "BTC", Quantity: "0.1", HoldingPeriodMinutes: 1440}
	key, err := base.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	equivalent := base
	equivalent.Symbol = " btc/usdc "
	equivalent.Quantity = "0.100"
	equivalent.MinimumEstimatedFundingBps = "-0.00"
	if got, err := equivalent.SubscriptionKey(); err != nil || got != key {
		t.Fatalf("normalization: %q %v", got, err)
	}
	for _, period := range []uint32{1, 43200} {
		p := base
		p.HoldingPeriodMinutes = period
		if err := p.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, period := range []uint32{0, 43201} {
		p := base
		p.HoldingPeriodMinutes = period
		if err := p.Validate(); !errors.Is(err, app.InvalidParameter()) {
			t.Fatalf("period %d: %v", period, err)
		}
	}
	negative := base
	negative.MinimumEstimatedFundingBps = "-2.5"
	if err := negative.Validate(); err != nil {
		t.Fatal(err)
	}
	invalid := base
	invalid.RouteFamilies = []RouteFamily{"spot-spot"}
	if err := invalid.Validate(); err == nil {
		t.Fatal("accepted spot-spot")
	}
	changed := base
	changed.HoldingPeriodMinutes = 60
	if got, _ := changed.SubscriptionKey(); got == key {
		t.Fatal("period collision")
	}
	changed = base
	changed.MinimumEstimatedFundingBps = "1"
	if got, _ := changed.SubscriptionKey(); got == key {
		t.Fatal("threshold collision")
	}

	for _, value := range []string{"0x10", "1_000", "1e3", "1/2", "--1"} {
		invalid := base
		invalid.Quantity = value
		if err := invalid.Validate(); err == nil {
			t.Fatalf("accepted nondecimal: %s", value)
		}
	}
	var decoded Params
	for _, raw := range []string{`{"unknown":true}`, `{"holdingPeriodMinutes":1.5}`, `{"sourceFilter":{"unknown":true}}`} {
		if err := json.Unmarshal([]byte(raw), &decoded); !errors.Is(err, app.InvalidParameter()) {
			t.Fatalf("decode %s: %v", raw, err)
		}
	}
	result := Result{AssetClass: base.Normalize().AssetClass, Symbol: base.Symbol, BaseAsset: base.BaseAsset, Quantity: base.Quantity, HoldingPeriodMinutes: base.HoldingPeriodMinutes, MinimumEstimatedFundingBps: "0", RouteFamilies: base.Normalize().RouteFamilies}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip Result
	if err := json.Unmarshal(raw, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if got, err := roundtrip.Params().SubscriptionKey(); err != nil || got != key {
		t.Fatalf("roundtrip: %q %v", got, err)
	}
}
