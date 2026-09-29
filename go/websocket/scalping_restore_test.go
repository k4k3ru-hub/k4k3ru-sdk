package websocket

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/scalping"
)

// TestScalpingRestoresACKSettings verifies configuration survives the real SDK adapter.
//
// Version:
//   - 2026-09-29: Added.
func TestScalpingRestoresACKSettings(t *testing.T) {
	holding := uint64(60000)
	minimum := uint64(1)
	p := dto.Params{MarketType: market.MarketTypeSpot, Symbol: "SUI/USDC", BaseAsset: rule.AssetRef{Chain: "sui", Network: "testnet", AssetID: "0x2::sui::SUI"}, QuoteAsset: rule.AssetRef{Chain: "sui", Network: "testnet", AssetID: "usdc"}, Markets: []market.MarketTarget{{Venue: "cetus", Chain: "sui", Network: "testnet", PoolID: "pool"}}, Conditions: dto.Conditions{WindowMS: 60000, MaximumDataAgeMS: 60000, TradeCount: &dto.CountRange{Minimum: &minimum}}, ExecutionRule: rule.Rule{Open: rule.OpenRule{Spot: &rule.SpotOpenRule{MaximumAmount: "1000"}, ExecutionTTLMS: 30000}, Close: rule.CloseRule{MaximumHoldingMS: &holding, ExecutionTTLMS: 30000, Spot: &rule.SpotCloseRule{Markets: []rule.MarketRef{{Venue: "cetus", Chain: "sui", Network: "testnet", PoolID: "pool"}}}}}}.Normalize()
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(dto.SubscribeResult{ExecutionID: "scalp_restore", SubscriptionKey: "key", Params: &p})
	if err != nil {
		t.Fatal(err)
	}
	l, err := newSubscriptionLifecycle(&executionTransport{})
	if err != nil {
		t.Fatal(err)
	}
	c, err := newScalpingClient(executionSender(func(context.Context, rpc.Method, json.RawMessage) (*rpc.Response, error) {
		return &rpc.Response{Result: raw}, nil
	}), newScalpingEventRegistry(), l)
	if err != nil {
		t.Fatal(err)
	}
	s, err := c.Subscribe(t.Context(), dto.SubscribeParams{ExecutionID: "scalp_restore"})
	if err != nil {
		t.Fatal(err)
	}
	ref := s.Reference()
	if ref.Params == nil || ref.Params.ExecutionRule.Open.Spot.MaximumAmount != "1000" {
		t.Fatal("restored settings discarded")
	}
	ref.Params.Markets[0].PoolID = "other"
	ref.Params.ExecutionRule.Open.Spot.MaximumAmount = "9999"
	if s.Reference().Params.Markets[0].PoolID != "pool" || s.Reference().Params.ExecutionRule.Open.Spot.MaximumAmount != "1000" {
		t.Fatal("reference aliases mutable settings")
	}
}
