package onchain

import (
	"encoding/json"
	"reflect"
	"testing"

	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

func TestPaymentOnchainCreateIntentParamsJSON(t *testing.T) {
	t.Parallel()

	want := CreateIntentParams{ProductName: "usage-credits-1000"}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"productName":"usage-credits-1000"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got CreateIntentParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestPaymentOnchainCreateIntentResultJSON(t *testing.T) {
	t.Parallel()

	want := CreateIntentResult{
		IntentID:  2001,
		AccountID: 1786180518874776239,
		Status:    "pending",
		Chain:     k4k3ruOnchainCore.ChainBase,
		Network:   k4k3ruOnchainCore.NetworkMainnet,
		Token:     k4k3ruOnchainCore.TokenUSDC,
		Amount:    "10.00",
		Address:   "0x1234567890abcdef",
		ExpiresAt: "2026-08-29T12:34:56Z",
		Metadata:  json.RawMessage(`{"productName":"usage-credits-1000"}`),
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"intentId":"2001","accountId":"1786180518874776239","status":"pending","chain":"base","network":"mainnet","token":"USDC","amount":"10.00","address":"0x1234567890abcdef","expiresAt":"2026-08-29T12:34:56Z","metadata":{"productName":"usage-credits-1000"}}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got CreateIntentResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}
