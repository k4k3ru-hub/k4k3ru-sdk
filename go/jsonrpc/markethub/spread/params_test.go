package spread

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParamsNormalizeValidateAndSubscriptionKey(t *testing.T) {
	params := Params{Symbol: " btc/usdc ", BaseAsset: " btc ", Quantity: "0.100", SourceFilter: &SourceFilter{VenueCategories: []VenueCategory{"DEX", "cex"}}}
	normalized := params.Normalize()
	if normalized.AssetClass != AssetClassCrypto || normalized.Symbol != "BTC/USDC" || normalized.BaseAsset != "BTC" || normalized.Quantity != "0.1" || normalized.MinimumGrossSpreadBps != "0" || len(normalized.RouteFamilies) != 4 {
		t.Fatalf("Normalize() = %#v", normalized)
	}
	if err := normalized.Validate(); err != nil {
		t.Fatal(err)
	}
	key, err := normalized.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"MarketHub.Spread", "s=BTC/USDC", "ba=BTC", "q=0.1", "rf=PERP-PERP,PERP-SPOT,SPOT-PERP,SPOT-SPOT", "VC=CEX,DEX"} {
		if !strings.Contains(key, value) {
			t.Fatalf("key = %q, want %q", key, value)
		}
	}
}

func TestParamsValidateRejectsInvalidValues(t *testing.T) {
	tests := []Params{
		{Symbol: "BTC/USDC", BaseAsset: "ETH", Quantity: "1"},
		{Symbol: "BTC/USDC", BaseAsset: "BTC", Quantity: "0"},
		{Symbol: "BTC/USDC", BaseAsset: "BTC", Quantity: "1e2"},
		{Symbol: "BTC/USDC", BaseAsset: "BTC", Quantity: "1", MinimumGrossSpreadBps: "-1"},
		{Symbol: "BTC/USDC", BaseAsset: "BTC", Quantity: "1", RouteFamilies: []RouteFamily{"invalid"}},
	}
	for _, params := range tests {
		if err := params.Validate(); err == nil {
			t.Fatalf("Validate(%#v) error = nil", params)
		}
	}
}

func TestParamsUnmarshalRejectsUnknownField(t *testing.T) {
	var params Params
	if err := json.Unmarshal([]byte(`{"symbol":"BTC/USDC","baseAsset":"BTC","quantity":"1","unknown":true}`), &params); err == nil {
		t.Fatal("Unmarshal() error = nil")
	}
}

// TestMaxAgeSeconds verifies wire validation, defaults and subscription isolation.
// Version:
//   - 2026-09-12: Added.
func TestMaxAgeSeconds(t *testing.T) {
	p := Params{Symbol: "BTC/USDC", BaseAsset: "BTC", Quantity: "1"}
	defaultKey, err := p.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	p.MaxAgeSeconds = 5
	explicitKey, err := p.SubscriptionKey()
	if err != nil || explicitKey != defaultKey {
		t.Fatalf("default mismatch: %s %v", explicitKey, err)
	}
	p.MaxAgeSeconds = 60
	key, err := p.SubscriptionKey()
	if err != nil || key == defaultKey {
		t.Fatalf("age collision: %s %v", key, err)
	}
	for _, value := range []string{"-1", "1.5", "4294967296", "\"60\""} {
		var decoded Params
		if err := json.Unmarshal([]byte(`{"symbol":"BTC/USDC","baseAsset":"BTC","quantity":"1","maxAgeSeconds":`+value+`}`), &decoded); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Params
	if err = json.Unmarshal(data, &decoded); err != nil || decoded.MaxAgeSeconds != 60 {
		t.Fatalf("round trip: %+v %v", decoded, err)
	}
}
