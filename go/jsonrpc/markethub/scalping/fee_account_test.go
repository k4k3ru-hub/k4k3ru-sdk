package scalping_test

import (
	"encoding/json"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	dto "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	"strings"
	"testing"
)

// TestFeeAccountSubscriptionIdentity verifies default/account separation and canonical address matching.
//
// Version:
//   - 2026-09-28: Added.
func TestFeeAccountSubscriptionIdentity(t *testing.T) {
	p := dto.Params{MarketType: market.MarketTypeSpot, Symbol: "SUI/USDC", WindowMS: 60000, Markets: []market.MarketTarget{{Venue: market.Hyperliquid, Network: "mainnet"}}}
	standard, err := p.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	p.FeeAccounts = []dto.FeeAccount{{Venue: market.Hyperliquid, Network: "mainnet", Address: "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}}
	account, err := p.SubscriptionKey()
	if err != nil {
		t.Fatal(err)
	}
	if account == standard {
		t.Fatal("account subscription shared standard fees")
	}
	p.FeeAccounts[0].Address = strings.ToLower(p.FeeAccounts[0].Address)
	canonical, err := p.SubscriptionKey()
	if err != nil || canonical != account {
		t.Fatal("equivalent address got a different key", err)
	}
	n := p.Normalize()
	n.FeeAccounts[0].Address = "changed"
	if p.FeeAccounts[0].Address == n.FeeAccounts[0].Address {
		t.Fatal("normalization aliases account input")
	}
	p.FeeAccounts = append(p.FeeAccounts, p.FeeAccounts[0])
	if err := p.Validate(); err == nil {
		t.Fatal("duplicate scope accepted")
	}
	p.FeeAccounts = p.FeeAccounts[:1]
	p.FeeAccounts[0].Network = "testnet"
	if err := p.Validate(); err == nil {
		t.Fatal("account unrelated to requested markets accepted")
	}
}

// TestFeeAccountJSON verifies optional defaults, strict fields and malformed-address rejection.
//
// Version:
//   - 2026-09-28: Added.
func TestFeeAccountJSON(t *testing.T) {
	const valid = `{"marketType":"spot","symbol":"SUI/USDC","markets":[{"venue":"hyperliquid","network":"mainnet"}],"feeAccounts":[{"venue":"hyperliquid","network":"mainnet","address":"0x1111111111111111111111111111111111111111"}]}`
	var p dto.Params
	if err := json.Unmarshal([]byte(valid), &p); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{strings.Replace(valid, `"address":`, `"privateKey":`, 1), strings.Replace(valid, "0x1111111111111111111111111111111111111111", "wrong", 1)} {
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			t.Fatal("invalid fee account accepted")
		}
	}
}

// TestNetResultFieldNames rejects accidental reintroduction of public gross field names.
//
// Version:
//   - 2026-09-28: Added.
func TestNetResultFieldNames(t *testing.T) {
	price := "2.5"
	v := dto.MarketPrice{Market: market.MarketRef{Venue: market.Hyperliquid, Network: "mainnet", VenueSymbol: "@1"}, Status: dto.PriceStatusVWAP, NetPrice: &price, NetReceiveQuantity: &market.Quantity{Amount: "40", Decimals: 0}}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"netPrice":"2.5"`, `"netReceiveQuantity"`} {
		if !strings.Contains(string(b), field) {
			t.Fatal("missing net field", field)
		}
	}
	for _, field := range []string{`"price"`, `"receiveQuantity"`} {
		if strings.Contains(string(b), field) {
			t.Fatal("gross field leaked", field)
		}
	}
}
