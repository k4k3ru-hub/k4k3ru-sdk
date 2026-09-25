package execution

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/onchain/go/sui"
)

// TestSuiOrderReference verifies explicit linkage normalization and legacy compatibility.
//
// Version:
//   - 2026-09-25: Added.
func TestSuiOrderReference(t *testing.T) {
	var p SubmitParams
	if err := json.Unmarshal([]byte(`{"executionId":"close_1","openExecutionId":" open_1 ","payloadDigest":"digest","signedPayload":{"chainFamily":"sui","encoding":"base64","transactionBytes":"AQ==","signatures":[]}}`), &p); err != nil {
		t.Fatal(err)
	}
	p.SignedPayload.Signatures = []string{base64.StdEncoding.EncodeToString(make([]byte, 97))}
	if p.Normalize().OpenExecutionID != "open_1" || p.Validate() != nil {
		t.Fatal("valid explicit close rejected")
	}
	for _, invalid := range []string{"close_1", "bad/id", strings.Repeat("a", 65)} {
		copy := p
		copy.OpenExecutionID = invalid
		if copy.Validate() == nil {
			t.Fatalf("accepted reference %q", invalid)
		}
	}
	p.SignedPayload.ChainFamily = ChainFamilyEVM
	p.SignedPayload.Encoding = PayloadEncodingHex
	p.SignedPayload.TransactionBytes = "0x01"
	p.SignedPayload.Signatures = nil
	if p.Validate() == nil {
		t.Fatal("accepted unsupported EVM close linkage")
	}
	p.OpenExecutionID = ""
	if err := p.Validate(); err != nil {
		t.Fatal("broke unlinked EVM submit", err)
	}
}

// TestSuiOMSSnapshot verifies signed gas, exact realized PnL, and terminal checkpoint requirements.
//
// Version:
//   - 2026-09-25: Added.
func TestSuiOMSSnapshot(t *testing.T) {
	checkpoint := uint64(9007199254740993)
	asset := "0x0000000000000000000000000000000000000000000000000000000000000002::sui::SUI"
	snapshot := &ExecutionSnapshot{Status: ObservationStatusSuccess, ObservedAt: 1, Onchain: &OnchainExecution{ChainFamily: ChainFamilySui, Chain: "sui", Network: "testnet", TransactionID: sui.TransactionDigest{1}.String(), Checkpoint: &checkpoint}, OMS: &ExecutionOMS{
		OrderID: "18446744073709551615", OpenExecutionID: "open_1",
		Fill: &SwapFill{TokenInAssetID: "0x3::usdc::USDC", TokenOutAssetID: asset, TokenInDecimals: 6, TokenOutDecimals: 9, AmountIn: "4000", AmountOut: "1100000"},
		Fee:  &ExecutionFee{AssetID: asset, Decimals: 9, Amount: "-125"},
		PnL:  ExecutionPnL{Status: "realized", AssetID: asset, Decimals: 9, Amount: "100000"},
	}}
	event := SubscriptionEvent{ExecutionID: "close_1", SubscriptionKey: "sub_1", Sequence: 1, Kind: ExecutionEventSnapshot, Snapshot: snapshot}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	var decoded SubscriptionEvent
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if *decoded.Snapshot.Onchain.Checkpoint != checkpoint || decoded.Snapshot.OMS.PnL.Amount != "100000" {
		t.Fatal("lost exact quantities")
	}
	snapshot.Onchain.Checkpoint = nil
	if event.Validate() == nil {
		t.Fatal("accepted uncheckpointed fill")
	}
	snapshot.Onchain.Checkpoint = &checkpoint
	snapshot.OMS.PnL.Amount = "0.1"
	if event.Validate() == nil {
		t.Fatal("accepted fractional atomic PnL")
	}
	snapshot.Status, snapshot.Failure = ObservationStatusFailed, &ExecutionFailure{Code: "transaction_failed"}
	snapshot.OMS.Fill, snapshot.OMS.PnL = nil, ExecutionPnL{Status: "unavailable"}
	if err := event.Validate(); err != nil {
		t.Fatal("failed gas record rejected", err)
	}
	snapshot.OMS.PnL.Amount = "0"
	if event.Validate() == nil {
		t.Fatal("represented unavailable PnL as zero")
	}
	snapshot.OMS.PnL.Amount = ""
	snapshot.Status, snapshot.Failure, snapshot.OMS.Fee, snapshot.Onchain.Checkpoint = ObservationStatusPending, nil, nil, nil
	if err := event.Validate(); err != nil {
		t.Fatal("pending OMS rejected", err)
	}
}
