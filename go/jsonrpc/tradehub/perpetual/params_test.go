package perpetual

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

const offlineScopeJSON = `"venue":"hyperliquid","network":"testnet","symbol":"SUI/USDC","accountAddress":"0x1234567890123456789012345678901234567890"`

// TestPerpetualStrictParameters rejects ambiguous scopes and missing explicit trading constraints.
//
// Version:
//   - 2026-09-28: Added.
func TestPerpetualStrictParameters(t *testing.T) {
	valid := `{` + offlineScopeJSON + `,"signerAddress":"0x1234567890123456789012345678901234567890","kind":"order","nonce":1600000000000,"order":{"side":"buy","quantity":"10","limitPrice":"1.2","timeInForce":"ioc","reduceOnly":false,"clientOrderId":"0x12345678901234567890123456789012"}}`
	var p PrepareParams
	if err := json.Unmarshal([]byte(valid), &p); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		strings.Replace(valid, `"testnet"`, `"mainnet"`, 1), strings.Replace(valid, `"SUI/USDC"`, `"ETH/USDC"`, 1),
		strings.Replace(valid, `"reduceOnly":false,`, "", 1), strings.Replace(valid, `"reduceOnly":false`, `"reduceOnly":null`, 1),
		strings.Replace(valid, `"quantity":"10"`, `"quantity":"0"`, 1), strings.Replace(valid, `"timeInForce":"ioc"`, `"timeInForce":"gtc"`, 1),
		strings.Replace(valid, `"nonce":1600000000000`, `"nonce":0`, 1), strings.Replace(valid, `"kind":"order"`, `"kind":"order","accountId":8`, 1),
		strings.Replace(valid, `"network":"testnet"`, `"network":"testnet","network":"mainnet"`, 1), valid + `{}`,
	} {
		var p PrepareParams
		err := json.Unmarshal([]byte(bad), &p)
		if err == nil {
			err = p.Validate()
		}
		if err == nil {
			t.Fatal("invalid intent accepted")
		}
	}
	for _, value := range []string{"", "01", "-1", "+1", "18446744073709551616"} {
		q := OrderParams{Scope: p.Scope, OrderID: value}
		if q.Validate() == nil {
			t.Fatal("invalid order ID accepted")
		}
	}
	q := OrderParams{Scope: p.Scope, OrderID: "1", ClientOrderID: p.Order.ClientOrderID}
	if q.Validate() == nil {
		t.Fatal("two identifiers accepted")
	}
	p.Network = "mainnet"
	if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
		t.Fatal("validation code lost")
	}
}
