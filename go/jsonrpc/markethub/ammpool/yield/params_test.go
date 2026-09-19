package yield

import (
	"encoding/json"
	"errors"
	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"testing"
)

// TestStrictParameters verifies validation and normalized subscription identities.
//
// Version:
//   - 2026-09-19: Added.
func TestStrictParameters(t *testing.T) {
	for _, raw := range []string{`null`, `{"apy":true}`, `{"chain":"a:b"}`, `{"poolId":"0x1"}`} {
		var p Params
		if err := json.Unmarshal([]byte(raw), &p); err == nil || !errors.Is(err, app.InvalidParameter()) {
			t.Fatalf("accepted %s: %v", raw, err)
		}
	}
	a, err := (Params{Chain: " SUI ", Network: "MAINNET", Venue: " BlueFin "}).SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	b, err := (Params{Chain: "sui", Network: "mainnet", Venue: "bluefin"}).SubscriptionKey()
	if err != nil || a != b || a != "MarketHub.AMMPool.Yield:c=sui:n=mainnet:v=bluefin" {
		t.Fatalf("bad identity: %s %s %v", a, b, err)
	}
	for _, raw := range []string{`{}`, `null`, `{"chain":"sui","network":"mainnet","venue":"bluefin","poolId":"x","bad":1}`} {
		var p GetParams
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	for _, raw := range []string{`{"limit":201}`, `{"limit":-1}`, `{"filter":null}`, `{"bad":1}`} {
		var p ListParams
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if (ListParams{}).Normalize().Limit != 100 {
		t.Fatal("default limit")
	}
}

// TestMetricWireContract distinguishes unavailable data from a reported zero.
//
// Version:
//   - 2026-09-19: Added.
func TestMetricWireContract(t *testing.T) {
	zero := "0"
	raw, err := json.Marshal(APR{Fee: Metric{Status: "available", Value: &zero}, Reward: Metric{Status: "unavailable", Reason: "not_reported"}, Total: Metric{Status: "unavailable"}})
	if err != nil {
		t.Fatal(err)
	}
	var decoded APR
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Fee.Value == nil || *decoded.Fee.Value != "0" || decoded.Reward.Value != nil {
		t.Fatalf("lossy wire: %s", raw)
	}
}
