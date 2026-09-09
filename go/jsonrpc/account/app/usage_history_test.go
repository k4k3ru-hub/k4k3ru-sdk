package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUsageHistoryExactJSON(t *testing.T) {
	original := ListUsageCreditEventsResult{AccountID: ^uint64(0), Items: []UsageCreditEvent{{ID: ^uint64(0), OperationID: 1, CreditID: 2, Type: 2, DeltaTicks: -9223372036854775808}}, NextBeforeID: ^uint64(0)}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"deltaTicks":"-9223372036854775808"`) || !strings.Contains(string(data), `"accountId":"18446744073709551615"`) {
		t.Fatalf("not exact string encoding: %s", data)
	}
	var result ListUsageCreditEventsResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if result.AccountID != original.AccountID || result.Items[0].DeltaTicks != original.Items[0].DeltaTicks {
		t.Fatal("round trip changed integers")
	}
}
