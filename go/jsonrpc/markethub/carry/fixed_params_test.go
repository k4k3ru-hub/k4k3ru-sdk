package carry

import (
	"encoding/json"
	"testing"
)

func fixedParams() Params {
	return Params{Symbol: "BTC/USDC", BaseAsset: "BTC", Quantity: "0.1", HoldingPeriodMinutes: 1440, Route: RouteSelector{Buy: MarketSelector{Venue: "binance", MarketType: MarketTypeSpot}, Sell: MarketSelector{Venue: "hyperliquid", MarketType: MarketTypePerp}}}
}
func TestFixedIdentityAndResultRoundTrip(t *testing.T) {
	p := fixedParams()
	id, err := p.RouteID()
	if err != nil {
		t.Fatal(err)
	}
	key, err := p.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	same := p
	same.Quantity = "0.1000"
	same.Route.Buy.Venue = " BINANCE "
	sameKey, err := same.SubscriptionKey()
	if err != nil || sameKey != key {
		t.Fatalf("normalization: %s %v", sameKey, err)
	}
	for _, change := range []func(*Params){func(p *Params) { p.Quantity = "1" }, func(p *Params) { p.HoldingPeriodMinutes = 60 }} {
		next := p
		change(&next)
		nextID, err := next.RouteID()
		if err != nil || nextID != id {
			t.Fatal("market pair identity changed")
		}
		nextKey, err := next.SubscriptionKey()
		if err != nil || nextKey == key {
			t.Fatal("evaluation identity collision")
		}
	}
	next := p
	next.Route.Buy, next.Route.Sell = next.Route.Sell, next.Route.Buy
	nextID, err := next.RouteID()
	if err != nil || nextID == id {
		t.Fatal("direction identity collision")
	}
	r := Result{AssetClass: p.AssetClass, Symbol: p.Symbol, BaseAsset: p.BaseAsset, Quantity: p.Quantity, HoldingPeriodMinutes: p.HoldingPeriodMinutes, Route: p.Route, Status: "unavailable"}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Result
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	got, err := decoded.Params().SubscriptionKey()
	if err != nil || got != key {
		t.Fatal("unavailable event lost routing identity")
	}
}
func TestFixedParamsRejectSearchFiltersAndInvalidSelectors(t *testing.T) {
	for _, field := range []string{`"minimumEstimatedFundingBps":"3"`, `"routeFamilies":["spot-perp"]`, `"sourceFilter":{}`, `"interval":"1s"`} {
		var p Params
		if err := json.Unmarshal([]byte(`{"symbol":"BTC/USDC",`+field+`}`), &p); err == nil {
			t.Fatalf("accepted %s", field)
		}
	}
	for _, change := range []func(*Params){func(p *Params) { p.Route = RouteSelector{} }, func(p *Params) { p.Route.Sell = p.Route.Buy }, func(p *Params) { p.Route.Sell.MarketType = MarketTypeSpot }, func(p *Params) { p.Route.Buy.PoolID = "pool" }, func(p *Params) { p.Route.Buy.Chain = ChainBase }, func(p *Params) { p.HoldingPeriodMinutes = 0 }} {
		p := fixedParams()
		change(&p)
		if p.Validate() == nil {
			t.Fatalf("accepted %#v", p)
		}
	}
}
