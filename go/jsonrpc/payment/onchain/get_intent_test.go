package onchain

import (
	"encoding/json"
	"reflect"
	"testing"

	k4k3ruOnchainCore "github.com/k4k3ru-hub/onchain/go/core"
)

func TestPaymentOnchainGetIntentParamsJSON(t *testing.T) {
	t.Parallel()

	want := GetIntentParams{IntentID: 2001}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"intentId":"2001"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got GetIntentParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestPaymentOnchainGetIntentResultJSON(t *testing.T) {
	t.Parallel()

	want := GetIntentResult{
		IntentID:         2001,
		AccountID:        1786180518874776239,
		Status:           "pending",
		Chain:            k4k3ruOnchainCore.ChainBase,
		Network:          k4k3ruOnchainCore.NetworkMainnet,
		Token:            k4k3ruOnchainCore.TokenUSDC,
		RecipientAddress: "0x1234567890abcdef",
		Amount:           "10.00",
		ExpiresAt:        "2026-08-29T12:34:56Z",
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"intentId":"2001","accountId":"1786180518874776239","status":"pending","chain":"base","network":"mainnet","symbol":"USDC","recipientAddress":"0x1234567890abcdef","amount":"10.00","expiresAt":"2026-08-29T12:34:56Z"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got GetIntentResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}
