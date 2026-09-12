package ammpool

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	k4k3ruSDKAppError "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func TestListParamsNormalize(t *testing.T) {
	t.Parallel()

	got := (ListParams{Chain: " Base ", Network: " Sepolia ", Venue: " Uniswap-V3 ", Symbol: " weth/usdc "}).Normalize()
	if got.Chain != "base" || got.Network != "sepolia" || got.Venue != "uniswap-v3" || got.Symbol != "WETH/USDC" {
		t.Fatalf("Normalize() = %+v", got)
	}
}

func TestListParamsValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  ListParams
		wantErr string
	}{
		{name: "chain and network", params: ListParams{Chain: "base", Network: "sepolia"}},
		{name: "symbol", params: ListParams{Symbol: "WETH/USDC"}},
		{name: "all filters", params: ListParams{Chain: "base", Network: "mainnet", Venue: "uniswap-v3", Symbol: "WETH/USDC"}},
		{name: "empty selector", params: ListParams{}, wantErr: "selector=empty"},
		{name: "venue only", params: ListParams{Venue: "uniswap-v3"}, wantErr: "selector=empty"},
		{name: "network without chain", params: ListParams{Network: "sepolia", Symbol: "WETH/USDC"}, wantErr: "chain=empty"},
		{name: "chain without network", params: ListParams{Chain: "base"}, wantErr: "network=empty"},
		{name: "invalid chain", params: ListParams{Chain: "base!", Network: "sepolia"}, wantErr: "chain=invalid"},
		{name: "long venue", params: ListParams{Symbol: "WETH/USDC", Venue: strings.Repeat("a", maxListFilterLength+1)}, wantErr: "venue=too_long"},
		{name: "invalid symbol", params: ListParams{Symbol: "WETH"}, wantErr: "symbol=invalid"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.params.Validate()
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v", err)
				}
				return
			}
			if err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Validate() error = %v, want invalid parameter containing %q", err, test.wantErr)
			}
		})
	}
}

func TestListParamsUnmarshalJSON(t *testing.T) {
	t.Parallel()

	var params ListParams
	if err := json.Unmarshal([]byte(`{"chain":"base","network":"sepolia","venue":"uniswap-v3"}`), &params); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if params.Chain != "base" || params.Network != "sepolia" || params.Venue != "uniswap-v3" {
		t.Fatalf("json.Unmarshal() = %+v", params)
	}
	for _, payload := range []string{
		`{"chain":"base","network":"sepolia","unknown":true}`,
		`{"chain":"base"}`,
	} {
		if err := json.Unmarshal([]byte(payload), &params); err == nil || !errors.Is(err, k4k3ruSDKAppError.InvalidParameter()) {
			t.Fatalf("json.Unmarshal(%q) error = %v, want invalid parameter", payload, err)
		}
	}
	if err := json.Unmarshal([]byte(`{"symbol":"WETH/USDC"} {}`), &params); err == nil {
		t.Fatal("json.Unmarshal() error = nil, want malformed JSON error")
	}
}
