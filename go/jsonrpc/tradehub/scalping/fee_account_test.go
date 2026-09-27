package scalping

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observation "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
)

// TestFeeAccountObservationAndPersistence verifies fee identity survives normalization, forwarding and saved JSON.
//
// Version:
//   - 2026-09-28: Added.
func TestFeeAccountObservationAndPersistence(t *testing.T) {
	p := perpParams()
	p.FeeAccounts = []observation.FeeAccount{{Venue: market.Hyperliquid, Network: "mainnet", Address: "0x1111111111111111111111111111111111111111"}}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	n := p.Normalize()
	n.FeeAccounts[0].Address = "0x2222222222222222222222222222222222222222"
	if p.FeeAccounts[0].Address == n.FeeAccounts[0].Address {
		t.Fatal("normalization aliased account identity")
	}
	forwarded := p.ObservationParams()
	if !reflect.DeepEqual(forwarded.FeeAccounts, p.FeeAccounts) {
		t.Fatal("account context not forwarded")
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var restored Params
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.FeeAccounts, p.FeeAccounts) {
		t.Fatal("saved configuration lost account identity")
	}
}
