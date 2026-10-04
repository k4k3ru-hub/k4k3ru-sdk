package hyperliquid

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/execution/prepare"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/perpetual"
)

// TestExecutionLifetimeContract preserves explicit lifetimes and rejects ambiguous JSON at both boundaries.
//
// Version:
//   - 2026-10-04: Added.
func TestExecutionLifetimeContract(t *testing.T) {
	var absentIntent *perpetual.PrepareParams
	var absentCommon *prepare.PerpetualParams
	if absentIntent.UnmarshalJSON([]byte(`{}`)) == nil || absentCommon.UnmarshalJSON([]byte(`{}`)) == nil {
		t.Fatal("null decode destination accepted")
	}
	for _, ttl := range []uint64{1, 30000, 90000, perpetual.MaximumExecutionTTLMS} {
		p := intentFixture()
		p.ExecutionTTLMS = &ttl
		if err := p.Validate(); err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(PrepareParams(p))
		if err != nil {
			t.Fatal(err)
		}
		var common prepare.Params
		if err := json.Unmarshal(raw, &common); err != nil {
			t.Fatal(err)
		}
		got, err := Intent(*common.Perpetual)
		if err != nil || got.ExecutionTTLMS == nil || got.EffectiveExecutionTTLMS() != ttl {
			t.Fatal("lifetime lost", err)
		}
	}
	p := intentFixture()
	if p.EffectiveExecutionTTLMS() != 60000 {
		t.Fatal("manual default changed")
	}
	legacy, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	common, err := json.Marshal(PrepareParams(p))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(common), "executionTtlMs") {
		t.Fatal("default should remain omitted")
	}
	for _, invalid := range []string{"0", "null", `"30000"`, "-1", fmt.Sprint(perpetual.MaximumExecutionTTLMS + 1), "30000,\"executionTtlMs\":60000"} {
		field := `"executionTtlMs":` + invalid + `,"signerAddress":`
		for _, input := range []struct {
			raw    []byte
			common bool
		}{{legacy, false}, {common, true}} {
			modified := strings.Replace(string(input.raw), `"signerAddress":`, field, 1)
			var err error
			if input.common {
				var v prepare.Params
				err = json.Unmarshal([]byte(modified), &v)
				if err == nil {
					err = v.Validate()
				}
			} else {
				var v perpetual.PrepareParams
				err = json.Unmarshal([]byte(modified), &v)
				if err == nil {
					err = v.Validate()
				}
			}
			if err == nil {
				t.Fatalf("invalid lifetime accepted: common=%t value=%s", input.common, invalid)
			}
		}
	}
}
