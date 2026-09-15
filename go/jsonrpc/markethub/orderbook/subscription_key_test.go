package orderbook

import (
	"errors"
	sdkError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"reflect"
	"testing"
)

func TestParseSubscriptionKeyRoundTrip(t *testing.T) {
	for _, symbol := range []Symbol{" btc/usdt ", "A:B|C=D", "A:src=*:d=2", "Ξ/USDC"} {
		for _, filter := range []*SourceFilter{nil, {},
			{VenueCategories: []VenueCategory{VenueCategoryCEX}},
			{LiquidityModels: []LiquidityModel{LiquidityModelOrderBook}},
			{VenueCategories: []VenueCategory{VenueCategoryDEX, VenueCategoryCEX}, LiquidityModels: []LiquidityModel{LiquidityModelOrderBook, LiquidityModelAMM}, AMMPoolChains: []Chain{ChainSui, ChainEthereum, ChainBase, ChainSolana, ChainBNB}},
		} {
			for _, market := range []MarketType{MarketTypeSpot, MarketTypePerp} {
				p := Params{Symbol: symbol, MarketType: market, SourceFilter: filter}
				key, err := p.SubscriptionKey()
				if err != nil {
					t.Fatal(err)
				}
				got, err := ParseSubscriptionKey(key)
				if err != nil {
					t.Fatalf("parse %q: %v", key, err)
				}
				if !reflect.DeepEqual(got, p.Normalize()) {
					t.Fatalf("round trip: got %#v, want %#v", got, p.Normalize())
				}
			}
		}
	}
}

func TestParseSubscriptionKeyRejectsNoncanonicalKeys(t *testing.T) {
	prefix := "MarketHub.OrderBook:ac=CRYPTO:mt=SPOT:s=BTC/USDT:d=3:src="
	for _, key := range []string{"", "MarketHub.Other:ac=CRYPTO:mt=SPOT:s=BTC/USDT:src=*", prefix,
		prefix + "VC=", prefix + "VC=UNKNOWN", prefix + "VC=CEX,CEX", prefix + "VC=DEX,CEX",
		prefix + "VC=CEX|VC=CEX", prefix + "LM=AMM|VC=DEX", prefix + "APC=BASE", prefix + "*|VC=CEX",
		prefix + "VC=cex", prefix + "VC=CEX ", prefix + "UNKNOWN=X",
		"MarketHub.OrderBook:ac=crypto:mt=SPOT:s=BTC/USDT:d=3:src=*",
		"MarketHub.OrderBook:ac=CRYPTO:mt=SPOT:s=btc/usdt:d=3:src=*",
		"MarketHub.OrderBook:ac=CRYPTO:mt=SPOT:s=:d=3:src=*",
	} {
		if _, err := ParseSubscriptionKey(key); !errors.Is(err, sdkError.InvalidParameter()) {
			t.Errorf("parse %q: %v", key, err)
		}
	}
}

func FuzzParseSubscriptionKey(f *testing.F) {
	f.Add("MarketHub.OrderBook:ac=CRYPTO:mt=SPOT:s=BTC/USDT:d=3:src=*")
	f.Add("")
	f.Fuzz(func(t *testing.T, key string) {
		p, err := ParseSubscriptionKey(key)
		if err != nil {
			return
		}
		regenerated, err := p.SubscriptionKey()
		if err != nil || regenerated != key {
			t.Fatalf("noncanonical result: %q, %v", regenerated, err)
		}
	})
}

func TestParseSubscriptionKeyDepth(t *testing.T) {
	for depth := uint16(1); depth <= 20; depth++ {
		p := Params{MarketType: MarketTypeSpot, Symbol: "BTC/USDT", Depth: depth}
		key, err := p.SubscriptionKey()
		if err != nil {
			t.Fatal(err)
		}
		got, err := ParseSubscriptionKey(key)
		if err != nil || got.Depth != depth {
			t.Fatalf("depth %d: %#v, %v", depth, got, err)
		}
	}
	for _, depth := range []string{"0", "21", "03", "-1", "65536", "x", ""} {
		key := "MarketHub.OrderBook:ac=CRYPTO:mt=SPOT:s=BTC/USDT:d=" + depth + ":src=*"
		if _, err := ParseSubscriptionKey(key); !errors.Is(err, sdkError.InvalidParameter()) {
			t.Errorf("depth %q: %v", depth, err)
		}
	}
}
