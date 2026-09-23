package executionrule

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// TestAssetReferenceIdentity verifies scope exclusivity and identifier preservation.
//
// Version:
//   - 2026-09-23: Added.
func TestAssetReferenceIdentity(t *testing.T) {
	input := AssetRef{Chain: " SUI ", Network: " MAINNET ", AssetID: " 0x2::sui::SUI "}
	normalized := input.Normalize()
	if normalized.Chain != "sui" || normalized.Network != "mainnet" || normalized.AssetID != "0x2::sui::SUI" {
		t.Fatalf("unexpected normalization: %+v", normalized)
	}
	if input.AssetID != " 0x2::sui::SUI " {
		t.Fatal("input was mutated")
	}
	for name, ref := range map[string]AssetRef{
		"both scopes":        {Chain: "sui", Venue: "hyperliquid", Network: "mainnet", AssetID: "SUI"},
		"no scope":           {Network: "mainnet", AssetID: "SUI"},
		"missing network":    {Chain: "sui", AssetID: "SUI"},
		"missing identity":   {Chain: "sui", Network: "mainnet"},
		"oversized identity": {Chain: "sui", Network: "mainnet", AssetID: strings.Repeat("x", 257)},
	} {
		t.Run(name, func(t *testing.T) {
			if !errors.Is(ref.Validate(), apperror.InvalidParameter()) {
				t.Fatal("expected inspectable validation error")
			}
		})
	}
	if err := (AssetRef{Venue: "hyperliquid", Network: "mainnet", AssetID: "token-usdc"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestMarketReferenceIdentity verifies pool requirements and native symbol handling.
//
// Version:
//   - 2026-09-23: Added.
func TestMarketReferenceIdentity(t *testing.T) {
	pool := MarketRef{Venue: "cetus", Network: "mainnet", Chain: "sui", PoolID: "PoolCase"}
	book := MarketRef{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: "xyz:SUI"}
	for _, ref := range []MarketRef{pool, book} {
		if err := ref.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	if book.Normalize().VenueSymbol != "xyz:SUI" {
		t.Fatal("native symbol changed")
	}
	noChain := pool
	noChain.Chain = ""
	ambiguous := book
	ambiguous.PoolID = "pool"
	for _, ref := range []MarketRef{noChain, ambiguous, {Venue: "cetus", Network: "mainnet"}} {
		if err := ref.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid reference accepted: %+v", ref)
		}
	}
	duplicate := pool
	duplicate.Venue = " CETUS "
	if err := ValidateMarkets([]MarketRef{pool, duplicate}); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("duplicate normalized pool accepted")
	}
	if err := ValidateMarkets([]MarketRef{pool, book}); err != nil {
		t.Fatal(err)
	}
}

// TestReferenceJSONRejectsAmbiguity verifies strict asset and market objects.
//
// Version:
//   - 2026-09-23: Added.
func TestReferenceJSONRejectsAmbiguity(t *testing.T) {
	for _, data := range []string{
		`null`, `[]`, `{}`, `{"chain":"sui","network":null,"assetId":"SUI"}`,
		`{"chain":"sui","network":"mainnet","assetId":"SUI","AssetId":"OTHER"}`,
		`{"chain":"sui","network":"mainnet","assetId":"SUI","marketType":"spot"}`,
		`{"chain":"sui","network":"mainnet","assetId":"SUI"} {}`,
	} {
		var ref AssetRef
		if err := json.Unmarshal([]byte(data), &ref); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
	var ref MarketRef
	if err := json.Unmarshal([]byte(`{"venue":"hyperliquid","network":"mainnet","venueSymbol":"SUI","marketType":"perp"}`), &ref); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("nested market type accepted")
	}
}
