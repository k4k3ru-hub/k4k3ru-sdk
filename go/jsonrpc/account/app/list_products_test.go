package app

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAccountAppListProductsParamsJSON(t *testing.T) {
	t.Parallel()

	want := ListProductsParams{Page: 3}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != `{"page":3}` {
		t.Fatalf("Marshal() = %s, want %s", data, `{"page":3}`)
	}

	var got ListProductsParams
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestAccountAppListProductsResultJSON(t *testing.T) {
	t.Parallel()

	description := "Starter product"
	want := ListProductsResult{
		Products: []*ListProductsProduct{
			{
				ID:            1001,
				Name:          "starter",
				Type:          "one_time",
				CreditTicks:   1000000,
				BonusTicks:    100000,
				PriceAmount:   500,
				PriceCurrency: "usd",
				ExpiresInDays: 30,
				PurchaseLimit: 1,
				Description:   &description,
				MetaData:      json.RawMessage(`{"featured":true}`),
			},
		},
		Page:       2,
		Limit:      20,
		Total:      21,
		TotalPages: 2,
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	wantJSON := `{"products":[{"id":"1001","name":"starter","type":"one_time","creditTicks":"1000000","bonusTicks":"100000","priceAmount":"500","priceCurrency":"usd","expiresInDays":30,"purchaseLimit":1,"description":"Starter product","metaData":{"featured":true}}],"page":2,"limit":20,"total":21,"totalPages":2}`
	if string(data) != wantJSON {
		t.Fatalf("Marshal() = %s, want %s", data, wantJSON)
	}

	var got ListProductsResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestAccountAppListProductsResultNullableFieldsJSON(t *testing.T) {
	t.Parallel()

	result := ListProductsResult{
		Products: []*ListProductsProduct{
			{
				Description: nil,
				MetaData:    json.RawMessage(`null`),
			},
		},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"products":[{"id":"0","name":"","type":"","creditTicks":"0","bonusTicks":"0","priceAmount":"0","priceCurrency":"","expiresInDays":0,"purchaseLimit":0,"description":null,"metaData":null}],"page":0,"limit":0,"total":0,"totalPages":0}`
	if string(data) != want {
		t.Fatalf("Marshal() = %s, want %s", data, want)
	}
}

func TestAccountAppListProductsResultEmptyProductsJSON(t *testing.T) {
	t.Parallel()

	data, err := json.Marshal(ListProductsResult{Products: []*ListProductsProduct{}})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := `{"products":[],"page":0,"limit":0,"total":0,"totalPages":0}`
	if string(data) != want {
		t.Fatalf("Marshal() = %s, want %s", data, want)
	}
}
