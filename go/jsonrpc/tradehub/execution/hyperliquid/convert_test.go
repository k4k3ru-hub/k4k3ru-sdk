package hyperliquid

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution/prepare"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/perpetual"
)

func intentFixture() perpetual.PrepareParams {
	return perpetual.PrepareParams{Scope: perpetual.Scope{Venue: "hyperliquid", Network: "testnet", Symbol: "SUI/USDC", AccountAddress: "0x" + strings.Repeat("12", 20)},
		SignerAddress: "0x" + strings.Repeat("34", 20), Kind: "order", Nonce: 1700000000000,
		Order: &perpetual.OrderIntent{Side: "buy", Quantity: "10", LimitPrice: "1.2", TimeInForce: "ioc", ReduceOnly: false, ClientOrderID: "0x" + strings.Repeat("56", 16)}}
}

// TestExecutionConversions preserves intent, token, signature and receipt across the public envelopes.
//
// Version:
//   - 2026-09-29: Added.
func TestExecutionConversions(t *testing.T) {
	p := intentFixture()
	raw, err := json.Marshal(PrepareParams(p))
	if err != nil {
		t.Fatal(err)
	}
	var common prepare.Params
	if err := json.Unmarshal(raw, &common); err != nil {
		t.Fatal(err)
	}
	got, err := Intent(*common.Perpetual)
	if err != nil || !reflect.DeepEqual(got, p) {
		t.Fatalf("intent conversion failed: %v", err)
	}
	original := &perpetual.PrepareResult{Prepared: perpetual.Prepared{PreparationID: "0x" + strings.Repeat("78", 16), Intent: p, Action: json.RawMessage(`{"type":"order"}`), Digest: "0x" + strings.Repeat("90", 32), PreparedAt: 1700000000000, ExpiresAfter: 1700000060000}, PreparedToken: "offline-token"}
	result, err := PrepareResult(original)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Prepared(result)
	if err != nil || !reflect.DeepEqual(original, decoded) {
		t.Fatalf("prepared conversion failed: %v", err)
	}
	signed := perpetual.SubmitParams{PreparationID: original.Prepared.PreparationID, Digest: original.Prepared.Digest, PreparedToken: original.PreparedToken, Signature: perpetual.Signature{R: "0x0001", S: "0x02", V: 27}}
	submission, err := SubmitParams(signed)
	if err != nil {
		t.Fatal(err)
	}
	back, err := Submission(submission)
	if err != nil || back != signed {
		t.Fatalf("signature or token changed: %v", err)
	}
	receipt := &perpetual.SubmitResult{PreparationID: signed.PreparationID, Kind: "order", Status: "partially_filled", OrderID: "123", ClientOrderID: p.Order.ClientOrderID, RequestedSize: "10", FilledSize: "4", AveragePrice: "1.1", ReconciliationRequired: true}
	ack, err := SubmitResult(receipt, 1700000001000000)
	if err != nil {
		t.Fatal(err)
	}
	backReceipt, err := Receipt(ack)
	if err != nil || !reflect.DeepEqual(receipt, backReceipt) || ack.TransactionID != "" || ack.ChainFamily != "" {
		t.Fatalf("receipt conversion failed: %v", err)
	}
	ack.TransactionID = "fake-tx"
	if _, err := Receipt(ack); err == nil {
		t.Fatal("mixed action and transaction accepted")
	}
}

// TestExecutionBoundaries rejects ambiguity, owner injection, unsupported signing and token rebinding.
//
// Version:
//   - 2026-09-29: Added.
func TestExecutionBoundaries(t *testing.T) {
	p := PrepareParams(intentFixture())
	for _, mutate := range []func(*prepare.PerpetualParams){
		func(p *prepare.PerpetualParams) { p.Network = "mainnet" },
		func(p *prepare.PerpetualParams) { p.Venue = "dydx" },
		func(p *prepare.PerpetualParams) { p.Symbol = "BTC/USDC" },
		func(p *prepare.PerpetualParams) { p.Signing.Hyperliquid = nil },
		func(p *prepare.PerpetualParams) { p.Order.TimeInForce = "gtc" },
	} {
		v := PrepareParams(intentFixture())
		mutate(v.Perpetual)
		if _, err := Intent(*v.Perpetual); err == nil {
			t.Fatal("unsupported intent accepted")
		}
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{`null`, `{"kind":"swap","swap":null}`, `{"kind":"swap","kind":"perpetual"}`, strings.Replace(string(raw), `"kind":"perpetual"`, `"accountId":999,"kind":"perpetual"`, 1), strings.Replace(string(raw), `"reduceOnly":false`, `"reduceOnly":null`, 1)} {
		var v prepare.Params
		if err := json.Unmarshal([]byte(data), &v); err == nil {
			t.Fatal("invalid preparation accepted")
		}
	}
	signed := perpetual.SubmitParams{PreparationID: "0x" + strings.Repeat("11", 16), Digest: "0x" + strings.Repeat("22", 32), PreparedToken: "offline-token", Signature: perpetual.Signature{R: "0x1", S: "0x2", V: 27}}
	for _, mutate := range []func(*execution.SubmitParams){
		func(p *execution.SubmitParams) { p.SignedPayload.ChainFamily = execution.ChainFamilyEVM },
		func(p *execution.SubmitParams) { p.SignedPayload.Action.Network = "mainnet" },
		func(p *execution.SubmitParams) {
			p.SignedPayload.Action.Signature = json.RawMessage(`{"r":"0x1","s":"0x2","v":27,"R":"0x3"}`)
		},
		func(p *execution.SubmitParams) { p.OpenExecutionID = "exec_open" },
		func(p *execution.SubmitParams) { p.PreparedToken = "" },
	} {
		v, err := SubmitParams(signed)
		if err != nil {
			t.Fatal(err)
		}
		mutate(&v)
		if _, err := Submission(v); err == nil {
			t.Fatal("invalid submission accepted")
		}
	}
	var decoded execution.SubmitParams
	err = json.Unmarshal([]byte(`{"executionId":"a","private-token-value":true}`), &decoded)
	if err == nil || strings.Contains(err.Error(), "private-token-value") {
		t.Fatal("decode error leaked input")
	}
	original := &perpetual.PrepareResult{Prepared: perpetual.Prepared{PreparationID: signed.PreparationID, Intent: intentFixture(), Digest: signed.Digest, Action: json.RawMessage(`{}`), PreparedAt: 1, ExpiresAfter: 2}, PreparedToken: signed.PreparedToken}
	for _, mutate := range []func(*prepare.ActionPreparation){func(p *prepare.ActionPreparation) { p.SubmitParams.ExecutionID = "other" }, func(p *prepare.ActionPreparation) { p.SubmitParams.PayloadDigest = "other" }, func(p *prepare.ActionPreparation) { p.ExpiresAt++ }, func(p *prepare.ActionPreparation) { p.Network = "mainnet" }} {
		r, err := PrepareResult(original)
		if err != nil {
			t.Fatal(err)
		}
		mutate(r.Action)
		if _, err := Prepared(r); err == nil {
			t.Fatal("rebound preparation accepted")
		}
	}
}
