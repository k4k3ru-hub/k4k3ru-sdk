package scalping

import (
	"encoding/json"
	observation "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	"strings"
	"testing"
)

// TestRunGrossPriceContract preserves optional gross prices and rejects invalid values.
//
// Version:
//   - 2026-10-04: Added.
func TestRunGrossPriceContract(t *testing.T) {
	s := runSnapshotFixture()
	p := observation.MarketPrice{Market: s.Orders[0].Market, Status: observation.PriceStatusReference, NetPrice: pointer("2.1"), GrossPrice: pointer("2"), ObservedAt: pointer(int64(9000))}
	s.Entry = RunEvaluation{Status: EvaluationStatusMatched, Markets: []RunMarketEvaluation{{Price: p, Status: EvaluationStatusMatched, Candidate: &RunCandidate{CandidateID: "c", Revision: 1}}}}
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var decoded RunSnapshot
	if err = json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Entry.Markets[0].Price.GrossPrice == nil || *decoded.Entry.Markets[0].Price.GrossPrice != "2" {
		t.Fatal("gross price lost")
	}
	for _, invalid := range []string{"0", "-1", "NaN", "1/2"} {
		if err = json.Unmarshal([]byte(strings.Replace(string(raw), `"grossPrice":"2"`, `"grossPrice":"`+invalid+`"`, 1)), &decoded); err == nil {
			t.Fatal("invalid gross price accepted", invalid)
		}
	}
	s.Entry.Markets[0].Price.GrossPrice = nil
	if err = s.Validate(); err != nil {
		t.Fatal("optional gross required", err)
	}
}
