package ammpool

import (
	"encoding/json"
	"testing"
)

func TestListResultEncodesEmptyPoolsAsArray(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(ListResult{Pools: []PoolMetadata{}})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(payload) != `{"pools":[]}` {
		t.Fatalf("json.Marshal() = %s, want empty pools array", payload)
	}
}
