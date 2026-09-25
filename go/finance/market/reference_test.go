package market_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	onchain "github.com/k4k3ru-hub/onchain/go/core"
)

// TestMarketRefOnchainIdentity verifies owning types, open network names and the JSON contract.
//
// Version:
//   - 2026-09-25: Added.
func TestMarketRefOnchainIdentity(t *testing.T) {
	reference := market.MarketRef{
		Venue: market.Cetus, Chain: onchain.ChainSui,
		Network: onchain.NetworkTestnet, PoolID: "PoolCase",
	}
	encoded, err := json.Marshal(reference)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"venue":"cetus","network":"testnet","chain":"sui","poolId":"PoolCase"}`
	if string(encoded) != want {
		t.Fatalf("JSON = %s, want %s", encoded, want)
	}
	reference.Chain = " SUI "
	reference.Network = " CUSTOM-TESTNET "
	normalized := reference.Normalize()
	if normalized.Chain != onchain.ChainSui || normalized.Network != onchain.Network("custom-testnet") || normalized.PoolID != "PoolCase" {
		t.Fatalf("incorrect normalized scopes: %+v", normalized)
	}
	if err := normalized.Validate(); err != nil {
		t.Fatalf("custom network rejected: %v", err)
	}
	var decoded market.MarketRef
	if err := json.Unmarshal([]byte(strings.Replace(want, "testnet", "custom-testnet", 1)), &decoded); err != nil || decoded != normalized {
		t.Fatalf("custom network did not round-trip: %+v %v", decoded, err)
	}
	for _, chain := range []onchain.Chain{onchain.ChainPolygon, onchain.ChainAvalanche} {
		// A valid reference does not establish whether an adapter supports its pool.
		r := market.MarketRef{Venue: market.UniswapV3, Chain: chain, Network: onchain.NetworkMainnet, PoolID: "pool"}
		if err := r.Validate(); err != nil {
			t.Fatalf("onchain chain rejected: %q %v", chain, err)
		}
	}
	native := market.MarketRef{Venue: market.Hyperliquid, Network: onchain.NetworkMainnet, VenueSymbol: "SUI"}
	if err := native.Validate(); err != nil {
		t.Fatalf("native instrument without chain rejected: %v", err)
	}
	native.Network = onchain.Network(strings.Repeat("a", 16))
	if err := native.Validate(); err != nil {
		t.Fatalf("16-byte network rejected: %v", err)
	}
}

// TestMarketRefRejectsInvalidOnchainScopes verifies errors and atomic JSON decoding.
//
// Version:
//   - 2026-09-25: Added.
func TestMarketRefRejectsInvalidOnchainScopes(t *testing.T) {
	valid := market.MarketRef{Venue: market.Cetus, Chain: onchain.ChainSui, Network: onchain.NetworkTestnet, PoolID: "pool"}
	for name, mutate := range map[string]func(*market.MarketRef){
		"empty network":      func(r *market.MarketRef) { r.Network = "" },
		"long network":       func(r *market.MarketRef) { r.Network = onchain.Network(strings.Repeat("a", 17)) },
		"network space":      func(r *market.MarketRef) { r.Network = "custom net" },
		"network control":    func(r *market.MarketRef) { r.Network = "custom\x00net" },
		"missing pool chain": func(r *market.MarketRef) { r.Chain = "" },
		"none chain":         func(r *market.MarketRef) { r.Chain = "none" },
		"unknown chain":      func(r *market.MarketRef) { r.Chain = "unknown" },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := valid
			mutate(&invalid)
			if err := invalid.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("invalid scopes accepted or lost error classification: %v", err)
			}
			encoded, err := json.Marshal(invalid)
			if err != nil {
				t.Fatal(err)
			}
			decoded := valid
			if err := json.Unmarshal(encoded, &decoded); !errors.Is(err, apperror.InvalidParameter()) || !reflect.DeepEqual(decoded, valid) {
				t.Fatalf("invalid scopes changed the receiver or lost error classification: %v", err)
			}
		})
	}
}
