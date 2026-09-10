package ammpool

import (
	"encoding/json"
	"errors"
	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"testing"
)

// TestParamsWireAndIdentity verifies strict seconds and canonical subscription isolation.
//
// Version:
//   - 2026-09-10: Added.
func TestParamsWireAndIdentity(t *testing.T) {
	var p Params
	if err := json.Unmarshal([]byte(`{"symbol":" weth/usdc ","maxAgeSeconds":30}`), &p); err != nil {
		t.Fatal(err)
	}
	key, err := p.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	if key != "MarketHub.AMMPool:s=WETH/USDC:age=30" {
		t.Fatal(key)
	}
	other := p
	other.MaxAgeSeconds = 31
	otherKey, err := other.SubscriptionKey()
	if err != nil || key == otherKey {
		t.Fatal("age does not isolate subscriptions")
	}
	for _, raw := range []string{`null`, `{}`, `{"symbol":"WETH/USDC","maxAgeSeconds":0}`, `{"symbol":"WETH/USDC","maxAgeSeconds":-1}`, `{"symbol":"WETH/USDC","maxAgeSeconds":1.5}`, `{"symbol":"WETH/USDC","maxAgeMs":30}`, `{"symbol":"WETH/USDC","maxAgeSeconds":4294967296}`, `{"symbol":"WETH/USDC","maxAgeSeconds":30,"extra":1}`} {
		if err := json.Unmarshal([]byte(raw), &p); !errors.Is(err, app.InvalidParameter()) {
			t.Fatalf("accepted invalid request %s: %v", raw, err)
		}
	}
	result := Result{Symbol: "WETH/USDC", MaxAgeSeconds: 30, Pools: []Pool{}}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if v, exists := decoded["compositeMid"]; !exists || v != nil {
		t.Fatal("unavailable composite must be explicit null")
	}
	if _, exists := decoded["maxAgeSeconds"]; !exists {
		t.Fatal("missing selectors")
	}
}
