package ammpool

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	rpc "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
)

// TestListParams verifies optional filters, normalization, and safe validation errors.
//
// Version:
//   - 2026-09-17: Added.
func TestListParams(t *testing.T) {
	for _, payload := range []string{`{}`, `{"chain":"base"}`, `{"network":"sepolia"}`, `{"venue":"uniswap-v3"}`, `{"chain":"future-chain"}`} {
		var p ListParams
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			t.Fatalf("%s: %v", payload, err)
		}
	}
	var normalized ListParams
	if err := json.Unmarshal([]byte(`{"chain":" Base ","network":" Sepolia ","venue":" Uniswap-V3 "}`), &normalized); err != nil {
		t.Fatal(err)
	}
	if normalized != (ListParams{Chain: "base", Network: "sepolia", Venue: "uniswap-v3"}) {
		t.Fatalf("not normalized: %+v", normalized)
	}
	for _, p := range []ListParams{{Chain: "base/extra"}, {Network: "sepolia!"}, {Venue: strings.Repeat("x", 65)}} {
		if err := p.Validate(); !errors.Is(err, app.InvalidParameter()) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
}

// TestListParamsStrictDecode verifies malformed input and destination preservation.
//
// Version:
//   - 2026-09-17: Added.
func TestListParamsStrictDecode(t *testing.T) {
	for _, payload := range []string{`null`, `[]`, `"base"`, `{"unknown":true}`, `{"chain":1}`, `{} {}`, `{"venue":"bad!"}`, `{`} {
		p := ListParams{Chain: "original"}
		err := p.UnmarshalJSON([]byte(payload))
		if !errors.Is(err, app.InvalidParameter()) {
			t.Fatalf("%s: expected invalid parameter, got %v", payload, err)
		}
		if p.Chain != "original" {
			t.Fatal("invalid input overwrote destination")
		}
	}
	var p *ListParams
	if err := p.UnmarshalJSON([]byte(`{}`)); !errors.Is(err, app.InvalidParameter()) {
		t.Fatalf("nil destination: %v", err)
	}
}

// TestListWireContract verifies token order, numeric fields, and request method encoding.
//
// Version:
//   - 2026-09-19: Support manual swap pool discovery.
//   - 2026-09-17: Added.
func TestListWireContract(t *testing.T) {
	const payload = `{"pools":[{"chain":"base","network":"mainnet","venue":"uniswap-v3","poolId":"pool","token0":{"assetId":"weth-address","symbol":"WETH","decimals":18},"token1":{"assetId":"usdc-address","symbol":"USDC","decimals":6},"fee":500}]}`
	var result ListResult
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Pools) != 1 || result.Pools[0].Token0.Symbol != "WETH" || result.Pools[0].Token1.Symbol != "USDC" || result.Pools[0].Fee != 500 {
		t.Fatalf("unexpected result: %+v", result)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != payload[:len(payload)-1]+`,"supportedScopes":null}` {
		t.Fatalf("wire mismatch: %s", encoded)
	}
	empty, err := json.Marshal(ListResult{Pools: []PoolMetadata{}})
	if err != nil {
		t.Fatal(err)
	}
	if string(empty) != `{"pools":[],"supportedScopes":null}` {
		t.Fatalf("unexpected empty result: %s", empty)
	}
	params, err := json.Marshal(ListParams{})
	if err != nil {
		t.Fatal(err)
	}
	req := rpc.Request{ID: json.RawMessage(`"pools-1"`), Method: rpc.MethodTradeHubAMMPoolList, Params: params}
	if err := req.Validate(); err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if string(wire) != `{"id":"pools-1","method":"TradeHub.AMMPool.List","params":{}}` {
		t.Fatalf("request mismatch: %s", wire)
	}
}
