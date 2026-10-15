package bluefin

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution/prepare"
	perpetual "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/perpetual"
)

func testIntent() OrderIntent {
	address := "0x" + strings.Repeat("1", 64)
	return OrderIntent{Scope: Scope{"bluefin", "testnet", "SUI/USDC", address}, SignerAddress: address, Order: perpetual.OrderIntent{Side: "buy", Quantity: "1", LimitPrice: "1.05", TimeInForce: "ioc", ClientOrderID: "durable_order_1"}, Margin: Margin{1, "cross"}, Salt: "42", SignedAtMS: 1791907200000, ExecutionTTLMS: 60000}
}
func TestCommonEnvelopeAndStableIntent(t *testing.T) {
	wanted := testIntent()
	params, err := wanted.PrepareParams("session_one")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	var decoded prepare.Params
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	actual, err := Intent(*decoded.Perpetual)
	if err != nil || actual != wanted {
		t.Fatal("stable intent changed", err)
	}
	decoded.Perpetual.Signing.Bluefin.SessionID = "session_two"
	actual, err = Intent(*decoded.Perpetual)
	if err != nil || actual != wanted {
		t.Fatal("reauthentication changed trading identity", err)
	}
	decoded.Perpetual.Venue = "hyperliquid"
	if decoded.Validate() == nil {
		t.Fatal("Bluefin signing input accepted for Hyperliquid")
	}
}
func TestSecretSafeStrictDecode(t *testing.T) {
	for _, raw := range []string{`{"sessionId":"PRIVATE","sessionId":"PRIVATE"}`, `{"sessionId":"PRIVATE","unexpected":"PRIVATE"}`, `{"sessionId":null}`, `{}`} {
		var p SessionDeleteParams
		err := json.Unmarshal([]byte(raw), &p)
		if err == nil || strings.Contains(err.Error(), "PRIVATE") {
			t.Fatal("invalid session accepted or disclosed", err)
		}
		if !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatal("validation error chain lost", err)
		}
	}
	signature := base64.StdEncoding.EncodeToString(make([]byte, 97))
	var s Signature
	if err := json.Unmarshal([]byte(`{"serializedSignature":"`+signature+`","extra":true}`), &s); err == nil {
		t.Fatal("unknown signature field accepted")
	}
	p := testIntent()
	p.Salt = "18446744073709551616"
	if p.Validate() == nil {
		t.Fatal("overflowing salt accepted")
	}
	p = testIntent()
	p.AccountAddress = "0x" + strings.Repeat("1", 40)
	if p.Validate() == nil {
		t.Fatal("EVM address accepted")
	}
	p = testIntent()
	p.SignerAddress = "0x" + strings.Repeat("2", 64)
	if p.Validate() == nil {
		t.Fatal("unapproved authorized signer accepted")
	}
}
