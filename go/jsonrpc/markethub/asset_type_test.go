package markethub_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/candles"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/change"
	pricerange "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/range"
)

// TestAssetTypeRequired rejects missing, null and unsupported selectors in all asset APIs.
//
// Version:
//   - 2026-10-19: Added.
func TestAssetTypeRequired(t *testing.T) {
	type validator interface{ Validate() error }
	for name, makeParams := range map[string]func() validator{
		"price":  func() validator { return &price.Params{Asset: "sui", MarketType: "spot"} },
		"change": func() validator { return &change.Params{Asset: "sui", MarketType: "spot"} },
		"range":  func() validator { return &pricerange.Params{Asset: "sui", MarketType: "spot"} },
		"candles": func() validator {
			return &candles.Params{Asset: "sui", MarketType: "spot", From: 1792326600, To: 1792326660, Interval: "1m"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, raw := range []string{`{}`, `{"assetType":null}`, `{"assetType":""}`, `{"assetType":"fx"}`, `{"assetType":"stock"}`, `{"assetType":"Crypto"}`, `{"assetType":" crypto "}`} {
				p := makeParams()
				if err := json.Unmarshal([]byte(raw), p); err != nil {
					t.Fatal(err)
				}
				if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
					t.Fatalf("accepted invalid asset type: %s: %v", raw, err)
				}
			}
			for _, raw := range []string{`{"assetType":1}`, `{"assetType":{}}`, `{"assetType":[]}`} {
				if err := json.Unmarshal([]byte(raw), makeParams()); err == nil {
					t.Fatalf("accepted wrong JSON type: %s", raw)
				}
			}
			p := makeParams()
			if err := json.Unmarshal([]byte(`{"assetType":"crypto"}`), p); err != nil {
				t.Fatal(err)
			}
			if err := p.Validate(); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(encoded), `"assetType":"crypto"`) {
				t.Fatal("required selector missing on wire")
			}
		})
	}
	if p := (change.Params{Asset: "sui", MarketType: "spot"}).Normalize(); p.AssetType != market.AssetClassUnknown || p.Validate() == nil {
		t.Fatal("change normalization silently supplied an asset type")
	}
	if p := (pricerange.Params{Asset: "sui", MarketType: "spot"}).Normalize(); p.AssetType != market.AssetClassUnknown || p.Validate() == nil {
		t.Fatal("range normalization silently supplied an asset type")
	}
}
