package app

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCreditPaymentJSONRoundTrip(t *testing.T) {
	want := CreditPaymentParams{Source: "payment", IntentKind: "onchain", IntentID: 18446744073709551615, AccountID: 3001, CreditTicks: 12500000, ExpiresInDays: 30}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"source":"payment","intentKind":"onchain","intentId":"18446744073709551615","accountId":"3001","creditTicks":"12500000","expiresInDays":30}`
	if string(b) != expected {
		t.Fatalf("JSON=%s", b)
	}
	var got CreditPaymentParams
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roundtrip=%#v", got)
	}
	result := CreditPaymentResult{AccountID: 3001, CreditID: 4001, OperationID: 5001, CreditTicks: 12500000}
	b, err = json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var returned CreditPaymentResult
	if err := json.Unmarshal(b, &returned); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, returned) {
		t.Fatalf("result=%#v", returned)
	}
}
