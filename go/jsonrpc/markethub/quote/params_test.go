package quote

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

func validParams() Params {
	return Params{AssetType: "crypto", MarketType: "spot", Symbol: "SUI/USDC", Side: SideBuy,
		Quantity: market.Quantity{Amount: "100", Decimals: 0},
		Markets:  []market.MarketTarget{{Venue: market.Hyperliquid, Network: "mainnet"}},
	}
}

// TestParamsNormalization verifies independent normalization and identical quantity representation for both market types.
//
// Version:
//   - 2026-10-20: Added.
func TestParamsNormalization(t *testing.T) {
	p := validParams()
	p.MarketType, p.Symbol, p.Side = " SPOT ", " sui/usdc ", " BUY "
	p.Markets[0].Venue, p.Markets[0].Network, p.Markets[0].VenueSymbol = " Hyperliquid ", " Mainnet ", " uSUI "
	p.FeeAccounts = []FeeAccount{{Venue: " Hyperliquid ", Network: " Mainnet ", Address: " 0x" + strings.Repeat("AB", 20) + " "}}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	n := p.Normalize()
	if n.MarketType != "spot" || n.Symbol != "SUI/USDC" || n.Side != SideBuy || n.Markets[0].VenueSymbol != "uSUI" || n.FeeAccounts[0].Address != "0x"+strings.Repeat("ab", 20) {
		t.Fatal("selectors were not normalized")
	}
	n.Markets[0].Network, n.FeeAccounts[0].Address = "testnet", "changed"
	if p.Markets[0].Network != " Mainnet " || p.FeeAccounts[0].Address != " 0x"+strings.Repeat("AB", 20)+" " {
		t.Fatal("caller slices were mutated")
	}
	for _, typ := range []market.MarketType{"spot", "perpetual"} {
		for _, side := range []Side{SideBuy, SideSell} {
			p := validParams()
			p.MarketType, p.Side = typ, side
			data, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			var decoded Params
			if err := json.Unmarshal(data, &decoded); err != nil || !reflect.DeepEqual(decoded, p) {
				t.Fatalf("round trip failed: %v", err)
			}
		}
	}
}

// TestParamsValidation rejects ambiguous selectors, quantities and fee-account scopes.
//
// Version:
//   - 2026-10-20: Added.
func TestParamsValidation(t *testing.T) {
	account := FeeAccount{Venue: market.Hyperliquid, Network: "mainnet", Address: "0x" + strings.Repeat("12", 20)}
	for name, change := range map[string]func(*Params){
		"missing asset type": func(p *Params) { p.AssetType = "" },
		"unsupported asset":  func(p *Params) { p.AssetType = "fx" },
		"legacy perp":        func(p *Params) { p.MarketType = "perp" },
		"multiple pairs":     func(p *Params) { p.Symbol = "SUI/USDC,BTC" },
		"same asset":         func(p *Params) { p.Symbol = "SUI/SUI" },
		"symbol space":       func(p *Params) { p.Symbol = "SUI/U SDC" },
		"unknown side":       func(p *Params) { p.Side = "long" },
		"zero":               func(p *Params) { p.Quantity.Amount = "000" },
		"fraction":           func(p *Params) { p.Quantity.Amount = "1.2" },
		"negative":           func(p *Params) { p.Quantity.Amount = "-1" },
		"large amount":       func(p *Params) { p.Quantity.Amount = strings.Repeat("1", 385) },
		"no markets":         func(p *Params) { p.Markets = nil },
		"too many markets":   func(p *Params) { p.Markets = make([]market.MarketTarget, MaximumMarkets+1) },
		"duplicate markets": func(p *Params) {
			p.Markets = append(p.Markets, market.MarketTarget{Venue: " Hyperliquid ", Network: " MAINNET "})
		},
		"no network": func(p *Params) { p.Markets[0].Network = "" },
		"ambiguous instrument": func(p *Params) {
			p.Markets[0].PoolID, p.Markets[0].VenueSymbol = "pool", "SUI"
		},
		"fee network mismatch": func(p *Params) {
			p.FeeAccounts = []FeeAccount{account}
			p.FeeAccounts[0].Network = "testnet"
		},
		"unsupported account venue": func(p *Params) {
			p.FeeAccounts = []FeeAccount{account}
			p.FeeAccounts[0].Venue = market.Cetus
		},
		"malformed account": func(p *Params) {
			p.FeeAccounts = []FeeAccount{account}
			p.FeeAccounts[0].Address = "0x" + strings.Repeat("z", 40)
		},
		"duplicate accounts": func(p *Params) { p.FeeAccounts = []FeeAccount{account, account} },
		"too many accounts":  func(p *Params) { p.FeeAccounts = make([]FeeAccount, MaximumMarkets+1) },
	} {
		t.Run(name, func(t *testing.T) {
			p := validParams()
			change(&p)
			if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("expected inspectable validation failure: %v", err)
			}
		})
	}
	// Symbol validation must not depend on a hard-coded list of known assets.
	p := validParams()
	p.Symbol = "NEW/USDC"
	p.Quantity = market.Quantity{Amount: "1", Decimals: 255}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestParamsStrictDecode rejects missing, null, duplicate and unsupported fields without partial mutation.
//
// Version:
//   - 2026-10-20: Added.
func TestParamsStrictDecode(t *testing.T) {
	base := `{"assetType":"crypto","marketType":"spot","symbol":"SUI/USDC","side":"buy","quantity":{"amount":"100","decimals":0},"markets":[{"venue":"cetus","chain":"sui","network":"mainnet"}]}`
	var accepted Params
	if err := json.Unmarshal([]byte(base), &accepted); err != nil {
		t.Fatal(err)
	}
	invalid := []string{"null", "[]", base + "{}",
		strings.Replace(base, `"assetType":"crypto",`, "", 1),
		strings.Replace(base, `"crypto"`, `null`, 1),
		strings.Replace(base, `"quantity":{"amount":"100","decimals":0}`, `"quantity":null`, 1),
		strings.Replace(base, `,"decimals":0`, "", 1),
		strings.Replace(base, `"decimals":0`, `"decimals":256`, 1),
		strings.Replace(base, `"amount":"100"`, `"amount":"100","Amount":"1"`, 1),
		strings.Replace(base, `"side":"buy"`, `"side":"buy","side":"sell"`, 1),
		strings.Replace(base, `"side":"buy"`, `"side":"buy","kind":"exact-output"`, 1),
		strings.Replace(base, `"network":"mainnet"`, `"network":null`, 1),
		strings.Replace(base, `"markets":[{"venue":"cetus","chain":"sui","network":"mainnet"}]`, `"markets":[]`, 1),
		strings.Replace(base, `"markets":`, `"feeAccounts":[{"venue":"hyperliquid","network":"mainnet"}],"markets":`, 1),
	}
	for i, raw := range invalid {
		p := validParams()
		before := p.Normalize()
		if err := json.Unmarshal([]byte(raw), &p); !errors.Is(err, apperror.InvalidParameter()) {
			// encoding/json itself rejects trailing documents before UnmarshalJSON.
			if i != 2 || err == nil {
				t.Fatalf("case %d accepted or lost validation error: %v", i, err)
			}
		}
		if !reflect.DeepEqual(p, before) {
			t.Fatalf("case %d changed the receiver", i)
		}
	}
}
