package scalping_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	onchain "github.com/k4k3ru-hub/onchain/go/core"
)

const validRequest = `{"marketType":"spot","symbol":"SUI/USDC","markets":[{"venue":"cetus","network":"testnet","chain":"sui","poolId":"pool-1"}]}`

// TestScalpingFinanceParameters verifies shared types, canonical JSON and independent normalization.
//
// Version:
//   - 2026-09-25: Added.
func TestScalpingFinanceParameters(t *testing.T) {
	var p scalping.Params
	if err := json.Unmarshal([]byte(validRequest), &p); err != nil {
		t.Fatal(err)
	}
	if p.WindowMS != 60000 || p.Symbol != market.SUIUSDC || p.Markets[0].Venue != market.Cetus {
		t.Fatalf("incorrect finance parameters: %+v", p)
	}
	p.Symbol = " new/usdc "
	p.MarketType = " PERPETUAL "
	p.Markets = []market.MarketRef{{Venue: market.Hyperliquid, Network: "MAINNET", VenueSymbol: "NativeCase"}}
	n := p.Normalize()
	if n.Symbol != "NEW/USDC" || n.MarketType != market.MarketTypePerpetual || n.Markets[0].VenueSymbol != "NativeCase" || n.Validate() != nil {
		t.Fatalf("canonical dynamic-symbol request failed: %+v", n)
	}
	n.Markets[0].Network = "testnet"
	if p.Markets[0].Network != "MAINNET" {
		t.Fatal("Normalize modified the input markets")
	}
	data, err := json.Marshal(n)
	if err != nil || !strings.Contains(string(data), `"marketType":"perpetual"`) {
		t.Fatalf("incorrect wire spelling: %s %v", data, err)
	}
}

// TestScalpingRejectsAmbiguousRequests verifies invalid requests do not partially change the receiver.
//
// Version:
//   - 2026-09-25: Added.
func TestScalpingRejectsAmbiguousRequests(t *testing.T) {
	invalid := []string{
		`null`, `[]`, validRequest + `{}`,
		strings.Replace(validRequest, `"spot"`, `"perp"`, 1),
		strings.Replace(validRequest, `"spot"`, `"future"`, 1),
		strings.Replace(validRequest, `"spot"`, `null`, 1),
		strings.Replace(validRequest, `"SUI/USDC"`, `["SUI/USDC","BTC/USDC"]`, 1),
		strings.Replace(validRequest, `"SUI/USDC"`, `"SUI,ETH/USDC"`, 1),
		strings.Replace(validRequest, `"SUI/USDC"`, `"*/USDC"`, 1),
		strings.Replace(validRequest, `"SUI/USDC"`, `"SUI/USDC/BTC"`, 1),
		strings.Replace(validRequest, `"SUI/USDC"`, `"SUI/SUI"`, 1),
		strings.Replace(validRequest, `"markets":`, `"windowMs":null,"markets":`, 1),
		strings.Replace(validRequest, `"markets":`, `"windowMs":0,"markets":`, 1),
		strings.Replace(validRequest, `"markets":`, `"windowMs":60001,"markets":`, 1),
		strings.Replace(validRequest, `"markets":`, `"unknown":true,"markets":`, 1),
		strings.Replace(validRequest, `"marketType":`, `"MarketType":"perpetual","marketType":`, 1),
		strings.Replace(validRequest, `"venue":`, `"venue":"hyperliquid","venue":`, 1),
		strings.Replace(validRequest, `"poolId":`, `"venueSymbol":"SUI","poolId":`, 1),
		strings.Replace(validRequest, `"chain":"sui",`, ``, 1),
		strings.Replace(validRequest, `"cetus"`, `"unknown-venue"`, 1),
	}
	for _, raw := range invalid {
		var before scalping.Params
		if err := json.Unmarshal([]byte(validRequest), &before); err != nil {
			t.Fatal(err)
		}
		value := before.Normalize()
		err := value.UnmarshalJSON([]byte(raw))
		if !errors.Is(err, apperror.InvalidParameter()) || !reflect.DeepEqual(before, value) {
			t.Errorf("invalid request changed receiver or error contract: %s: %v", raw, err)
		}
	}
}

// TestScalpingMarketBounds verifies canonical duplicate detection and explicit Go window values.
//
// Version:
//   - 2026-09-25: Added.
func TestScalpingMarketBounds(t *testing.T) {
	var p scalping.Params
	if err := json.Unmarshal([]byte(validRequest), &p); err != nil {
		t.Fatal(err)
	}
	duplicate := p.Markets[0]
	duplicate.Venue = " CETUS "
	p.Markets = append(p.Markets, duplicate)
	if !errors.Is(p.Validate(), apperror.InvalidParameter()) {
		t.Fatal("normalized duplicate accepted")
	}
	p.Markets = p.Markets[:1]
	p.WindowMS = 0
	if p.Normalize().WindowMS != 0 || p.Validate() == nil {
		t.Fatal("explicit Go zero defaulted")
	}
	p.WindowMS = 1
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	p.Markets = make([]market.MarketRef, scalping.MaximumMarkets+1)
	if p.Validate() == nil {
		t.Fatal("market bound was lost")
	}
}

// TestScalpingOnchainNetworkValidation verifies shared network rules at the request boundary.
//
// Version:
//   - 2026-09-25: Added.
func TestScalpingOnchainNetworkValidation(t *testing.T) {
	var params scalping.Params
	request := strings.Replace(validRequest, `"testnet"`, `"custom-testnet"`, 1)
	if err := json.Unmarshal([]byte(request), &params); err != nil {
		t.Fatalf("custom network request rejected: %v", err)
	}
	if params.Markets[0].Chain != onchain.ChainSui || params.Markets[0].Network != onchain.Network("custom-testnet") {
		t.Fatalf("incorrect onchain reference: %+v", params.Markets[0])
	}
	for _, network := range []string{strings.Repeat("a", 17), "custom net"} {
		request := strings.Replace(validRequest, `"testnet"`, `"`+network+`"`, 1)
		if err := json.Unmarshal([]byte(request), &params); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid network request accepted: %v", err)
		}
	}
}
