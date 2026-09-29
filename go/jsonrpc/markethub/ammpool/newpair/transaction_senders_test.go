package newpair

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestTransactionSendersJSON preserves absent observations, exact counts and clone isolation.
//
// Version:
//   - 2026-09-29: Added.
func TestTransactionSendersJSON(t *testing.T) {
	for _, raw := range []string{`{"swapCount":"40"}`, `{"transactionSenders":null}`} {
		var w ActivityShortWindow
		if err := json.Unmarshal([]byte(raw), &w); err != nil {
			t.Fatal(err)
		}
		if w.TransactionSenders != nil {
			t.Fatal("missing observation became a count")
		}
	}
	for _, count := range []string{"0", "12", "9007199254740993"} {
		w := &ActivityShortWindow{TransactionSenders: &TransactionSenders{
			TransactionSenderCount: TransactionSenderCount{UniqueCount: &count, ResolvedSwapCount: "9007199254740994"},
			Token0ToToken1:         TransactionSenderCount{UniqueCount: &count, ResolvedSwapCount: "20"},
			Token1ToToken0:         TransactionSenderCount{ResolvedSwapCount: "0"},
		}}
		c := CloneActivityWindows(&ActivityWindows{FiveMinutes: w, FifteenMinutes: w})
		*c.FiveMinutes.TransactionSenders.UniqueCount = "changed"
		*c.FiveMinutes.TransactionSenders.Token0ToToken1.UniqueCount = "changed"
		if *w.TransactionSenders.UniqueCount != count || *c.FifteenMinutes.TransactionSenders.UniqueCount != count {
			t.Fatal("shared pointers")
		}
		data, err := json.Marshal(w)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), `"uniqueCount":"`+count+`"`) || !strings.Contains(string(data), `"uniqueCount":null`) {
			t.Fatal("nullable exact value lost", string(data))
		}
	}
}
