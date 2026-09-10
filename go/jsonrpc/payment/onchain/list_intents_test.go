package onchain

import (
	"encoding/json"
	"reflect"
	"testing"

	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
)

func TestListIntentsRequestContract(t *testing.T) {
	if rpc.MethodPaymentOnchainListIntents != "PaymentOnchain.ListIntents" {
		t.Fatal("wrong method name")
	}
	b, err := json.Marshal(ListIntentsParams{Page: 2})
	if err != nil {
		t.Fatal(err)
	}
	// The public request must not accept an account override.
	if string(b) != `{"page":2}` {
		t.Fatalf("unexpected params: %s", b)
	}
}

func TestListIntentsResultContract(t *testing.T) {
	raw := `{"intents":[{"intentId":"1787652917680715472","accountId":"1787450982076479542","status":"pending","chain":"base","network":"sepolia","symbol":"USDC","recipientAddress":"0x1111111111111111111111111111111111111111","amount":"1000000","expiresAt":"2026-09-10T13:00:00Z","createdAt":"2026-09-10T12:00:00Z","metadata":{"name":"usdc-base-sepolia-1","creditTicks":"1000000"}}],"page":1,"limit":20,"total":21,"totalPages":2}`
	var result ListIntentsResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Intents) != 1 || result.Intents[0].IntentID != 1787652917680715472 || result.Intents[0].AccountID != 1787450982076479542 || result.Intents[0].Amount != "1000000" {
		t.Fatal("ID or amount precision lost")
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != raw {
		t.Fatalf("wire contract changed: %s", encoded)
	}
	var roundtrip ListIntentsResult
	if err := json.Unmarshal(encoded, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, roundtrip) {
		t.Fatal("result changed after roundtrip")
	}
}

func TestListIntentsEmptyPage(t *testing.T) {
	b, err := json.Marshal(ListIntentsResult{Intents: []*ListIntentsIntent{}, Page: 1, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"intents":[],"page":1,"limit":20,"total":0,"totalPages":0}` {
		t.Fatalf("unexpected empty page: %s", b)
	}
}
