package newpair

import (
	"encoding/json"
	"errors"
	app "github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"strings"
	"testing"
)

// TestQueryIdentity verifies legacy keys, canonical vectors and value equality.
//
// Version:
//   - 2026-09-29: Added.
func TestQueryIdentity(t *testing.T) {
	legacy := Params{Chain: " BASE ", Network: "mainnet", Venue: "uniswap-v4"}
	key, err := legacy.SubscriptionKey()
	if err != nil || key != "MarketHub.AMMPool.NewPair:c=base:n=mainnet:v=uniswap-v4" {
		t.Fatal(key, err)
	}
	digest, err := (Query{}).Digest()
	if err != nil || digest != "cfc9b6b7bad1a7ea8037884aca870c32d39c1ed4149ff70199e163e2f9d16355" {
		t.Fatal(digest, err)
	}
	var a, b Params
	for i, raw := range []string{`{"query":{"conditions":{"minSwapCount":"00020","minLiquidityUsd":"0010.5000","requireNoMinting":false}}}`, `{"query":{"sort":{"direction":"desc","field":"poolCreatedAt"},"conditions":{"minLiquidityUsd":"10.5","minSwapCount":"20"},"activityPeriod":"5m"}}`} {
		target := &a
		if i == 1 {
			target = &b
		}
		if err := json.Unmarshal([]byte(raw), target); err != nil {
			t.Fatal(err)
		}
	}
	if !a.Equal(b) || a.Equal(Params{}) {
		t.Fatal("query identity mismatch")
	}
	original := Params{Query: &Query{Conditions: QueryConditions{MinLiquidityUSD: "0001.0"}}}
	normalized := original.Normalize()
	if original.Query.Conditions.MinLiquidityUSD != "0001.0" || normalized.Query.Conditions.MinLiquidityUSD != "1" {
		t.Fatal("normalization mutated caller")
	}
	zero := Params{Query: &Query{Conditions: QueryConditions{MinSwapCount: "0"}}}
	if zero.Equal(Params{Query: &Query{}}) {
		t.Fatal("explicit zero lost")
	}
}

// TestQueryInvalidInputs rejects ambiguous JSON and unsupported period conditions.
//
// Version:
//   - 2026-09-29: Added.
func TestQueryInvalidInputs(t *testing.T) {
	inputs := []string{`null`, `[]`, `{"conditions":null}`, `{"sort":null}`, `{"conditions":{"minSwapCount":null}}`, `{"conditions":{"minSwapCount":""}}`, `{"sort":{"field":""}}`, `{"conditions":{"minSwapCount":"1","minSwapCount":"2"}}`, `{"conditions":{},"conditions":{}}`, `{"extra":true}`, `{"activityPeriod":"1h","conditions":{"minUniqueSenderCount":"0"}}`, `{"activityPeriod":"24h","sort":{"field":"volumeUsdChangePercentage"}}`, `{"conditions":{"maxPoolFeeRate":"1.1"}}`, `{"conditions":{"minSwapCount":"1.0"}}`, `{"conditions":{"minVolumeUsd":"1e3"}}`, `{"conditions":{"minVolumeUsd":"-1"}}`, `{"conditions":{"minSwapCountChangePercentage":"-100.1"}}`, `{"conditions":{"minVolumeUsd":"+1"}}`}
	inputs = append(inputs, `{"conditions":{"minSwapCount":"`+strings.Repeat("9", 79)+`"}}`, `{`+strings.Repeat(" ", MaxQueryBytes)+`}`)
	inputs = append(inputs, `{"Conditions":{}}`, `{"activityPeriod":"5m","ActivityPeriod":"15m"}`, `{"conditions":{"minSwapCount":"1","MinSwapCount":"2"}}`, `{"sort":{"Field":"swapCount"}}`)
	for _, raw := range inputs {
		var q Query
		err := json.Unmarshal([]byte(raw), &q)
		if err == nil || !errors.Is(err, app.InvalidParameter()) {
			t.Fatalf("accepted invalid query %q: %v", raw, err)
		}
	}
	for _, raw := range []string{`{"query":null}`, `{"Query":null}`, `{"query":{},"query":{}}`, `{"query":{},"QUERY":{}}`} {
		var p Params
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			t.Fatal("accepted ambiguous query", raw)
		}
	}
}

// TestQueryNumericBoundaries retains large exact values and normalizes signed zero.
//
// Version:
//   - 2026-09-29: Added.
func TestQueryNumericBoundaries(t *testing.T) {
	raw := `{"conditions":{"minSwapCount":"` + strings.Repeat("9", 78) + `","maxPoolFeeRate":"1.000000000000000000","minSwapCountChangePercentage":"-100","minVolumeUsdChangePercentage":"-0.00"}}`
	var q Query
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		t.Fatal(err)
	}
	if q.Conditions.MaxPoolFeeRate != "1" || q.Conditions.MinVolumeUSDChangePercentage != "0" {
		t.Fatal(q)
	}
}
