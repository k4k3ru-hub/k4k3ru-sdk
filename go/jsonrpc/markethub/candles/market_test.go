package candles

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

// TestMarketSelectors validates optional instrument filters without mutating caller state.
//
// Version:
//   - 2026-10-19: Added.
func TestMarketSelectors(t *testing.T) {
	base := Params{AssetType: "crypto", Asset: "sui", MarketType: "spot", From: 1791417600, To: 1791417900, Interval: "5m", IncludeCurrent: true}
	for _, target := range []market.MarketTarget{
		{Venue: "cetus", Network: "mainnet"},
		{Venue: "cetus", Chain: "sui", Network: "mainnet", PoolID: "0xpool"},
		{Venue: "hyperliquid", Network: "mainnet", VenueSymbol: "SUI"},
	} {
		p := base
		p.Market = &target
		original := target
		if err := p.Validate(); err != nil || target != original {
			t.Fatalf("valid target rejected or mutated: %v", err)
		}
		encoded, err := json.Marshal(p)
		if err != nil || !strings.Contains(string(encoded), `"includeCurrent":true`) {
			t.Fatalf("developing selector missing: %s %v", encoded, err)
		}
		var decoded Params
		if err := json.Unmarshal(encoded, &decoded); err != nil || decoded.Market == nil || *decoded.Market != target || !decoded.IncludeCurrent {
			t.Fatalf("target round trip changed: %+v %v", decoded, err)
		}
	}
	for _, target := range []market.MarketTarget{
		{}, {Venue: "cetus"}, {Network: "mainnet"},
		{Venue: "cetus", Network: "mainnet", PoolID: "pool", VenueSymbol: "SUI"},
	} {
		p := base
		p.Market = &target
		if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid target accepted: %v", err)
		}
	}
	base.IncludeCurrent = false
	encoded, err := json.Marshal(base)
	if err != nil || strings.Contains(string(encoded), `"market":`) || strings.Contains(string(encoded), `"includeCurrent":`) {
		t.Fatalf("default request changed: %s %v", encoded, err)
	}
}
