package app

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAccountAppGetUsageBalanceParamsJSON(t *testing.T) {
	t.Parallel()

	want := GetUsageBalanceParams{AccountID: 1786180518874776239}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"accountId":"1786180518874776239"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got GetUsageBalanceParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestAccountAppGetUsageBalanceResultJSON(t *testing.T) {
	t.Parallel()

	want := GetUsageBalanceResult{
		AccountID:    1786180518874776239,
		BalanceTicks: 1000,
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"accountId":"1786180518874776239","balanceTicks":"1000"}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got GetUsageBalanceResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}
