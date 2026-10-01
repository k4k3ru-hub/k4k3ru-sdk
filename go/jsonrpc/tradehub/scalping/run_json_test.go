package scalping

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

const runStartJSON = `{"idempotencyKey":"key","marketType":"spot","symbol":"SUI/USDC","observation":{"markets":[{"venue":"cetus","chain":"sui"}]},"executionRule":{"markets":[{"venue":"cetus","chain":"sui","accountAddress":"wallet"}],"entry":{"side":"buy","maximumQuantity":{"amount":"100","decimals":1},"condition":{"priceChangeBps":{"minimum":"20"}}},"exit":{"maximumHoldingMs":60000}}}`

// TestRunStrictJSON rejects malformed public requests without replacing saved state.
//
// Version:
//   - 2026-10-01: Added.
func TestRunStrictJSON(t *testing.T) {
	for _, tt := range []struct{ name, old, replacement string }{
		{"missing key", `"idempotencyKey":"key",`, ``},
		{"missing type", `"marketType":"spot",`, ``},
		{"missing symbol", `"symbol":"SUI/USDC",`, ``},
		{"unknown root", `"marketType":"spot"`, `"marketType":"spot","unknown":1`},
		{"duplicate root", `"marketType":"spot"`, `"marketType":"spot","MARKETTYPE":"spot"`},
		{"null mode", `"marketType":"spot"`, `"marketType":"spot","testMode":null`},
		{"string mode", `"marketType":"spot"`, `"marketType":"spot","testMode":"true"`},
		{"network", `"symbol":"SUI/USDC"`, `"symbol":"SUI/USDC","network":"testnet"`},
		{"nested network", `"chain":"sui"`, `"chain":"sui","network":"testnet"`},
		{"unknown observation", `"observation":{`, `"observation":{"unknown":1,`},
		{"null window", `"observation":{`, `"observation":{"windowMs":null,`},
		{"fraction window", `"observation":{`, `"observation":{"windowMs":1.5,`},
		{"zero window", `"observation":{`, `"observation":{"windowMs":0,`},
		{"null buy", `"observation":{`, `"observation":{"buy":null,`},
		{"null quantity", `"observation":{`, `"observation":{"buy":{"quantity":null},`},
		{"unknown side setting", `"observation":{`, `"observation":{"buy":{"unknown":1},`},
		{"null rule", `"executionRule":{`, `"executionRule":null,"unused":{`},
		{"unknown rule", `"executionRule":{`, `"executionRule":{"simulate":true,`},
		{"null ttl", `"executionRule":{`, `"executionRule":{"executionTtlMs":null,`},
		{"null unsettled", `"executionRule":{`, `"executionRule":{"maximumUnsettledOrders":null,`},
		{"negative buffer", `"executionRule":{`, `"executionRule":{"reserveBufferBps":-1,`},
		{"uint overflow", `"executionRule":{`, `"executionRule":{"reserveBufferBps":18446744073709551616,`},
		{"string ttl", `"executionRule":{`, `"executionRule":{"executionTtlMs":"30000",`},
		{"missing side", `"side":"buy",`, ``},
		{"null limit", `"side":"buy"`, `"side":"buy","limitPrice":null`},
		{"missing decimals", `,"decimals":1`, ``},
		{"null decimals", `"decimals":1`, `"decimals":null`},
		{"overflow decimals", `"decimals":1`, `"decimals":256`},
		{"unknown quantity field", `"amount":"100"`, `"amount":"100","unit":"base"`},
		{"duplicate nested field", `"minimum":"20"`, `"minimum":"20","minimum":"30"`},
		{"null bound", `"minimum":"20"`, `"minimum":"20","maximum":null`},
		{"unknown metric", `"priceChangeBps"`, `"priceChange"`},
		{"unknown range", `"minimum":"20"`, `"minimum":"20","unknown":"30"`},
		{"number decimal", `"minimum":"20"`, `"minimum":20`},
		{"empty bound", `"minimum":"20"`, `"minimum":""`},
		{"null holding", `"maximumHoldingMs":60000`, `"maximumHoldingMs":null`},
		{"zero-only exit", `"maximumHoldingMs":60000`, `"takeProfit":{"type":"return_bps","value":"0"}`},
		{"unknown trigger", `"maximumHoldingMs":60000`, `"takeProfit":{"type":"return_bps","value":"100","unknown":1}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(runStartJSON, tt.old) {
				t.Fatal("test replacement did not match")
			}
			data := strings.Replace(runStartJSON, tt.old, tt.replacement, 1)
			original := RunParams{ExecutionID: "unchanged"}
			p := original
			if err := json.Unmarshal([]byte(data), &p); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("expected invalid parameter, got %v", err)
			}
			if !reflect.DeepEqual(p, original) {
				t.Fatal("failed decode replaced request")
			}
		})
	}
	for _, data := range []string{`null`, `[]`, `{}`, runStartJSON + `{}`} {
		var p RunParams
		if err := p.UnmarshalJSON([]byte(data)); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("expected invalid object: %v", err)
		}
	}
	config := runConfig()
	before := config.Normalize()
	config = before.Normalize()
	if err := json.Unmarshal([]byte(`{"marketType":"spot"}`), &config); err == nil {
		t.Fatal("incomplete configuration accepted")
	}
	if !reflect.DeepEqual(config, before) {
		t.Fatal("failed configuration decode was not atomic")
	}
}

// TestRunResume verifies resume isolation and new-start defaults.
//
// Version:
//   - 2026-10-01: Added.
func TestRunResume(t *testing.T) {
	var p RunParams
	if err := json.Unmarshal([]byte(`{"executionId":" execution-1 "}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.ExecutionID != "execution-1" || p.Params != nil || p.IdempotencyKey != "" {
		t.Fatal("resume acquired configuration")
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"executionId":"execution-1"}` {
		t.Fatalf("unexpected resume: %s", data)
	}
	for _, data := range []string{
		`{"executionId":""}`, `{"executionId":null}`,
		`{"executionId":"e","idempotencyKey":"k"}`,
		`{"executionId":"e","testMode":false}`,
		`{"executionId":"e","testMode":null}`,
		`{"executionId":"e","executionRule":null}`,
		`{"executionId":"e","unknown":1}`,
	} {
		if err := json.Unmarshal([]byte(data), &p); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("resume override accepted: %v", err)
		}
	}
	c := runConfig()
	for _, request := range []RunParams{{}, {Params: &c}, {IdempotencyKey: "key"}, {ExecutionID: "e", Params: &c}, {ExecutionID: "e", IdempotencyKey: "k"}} {
		if _, err := json.Marshal(request); !errors.Is(err, apperror.InvalidParameter()) {
			t.Fatalf("invalid request serialized: %v", err)
		}
	}
	if err := json.Unmarshal([]byte(runStartJSON), &p); err != nil {
		t.Fatal(err)
	}
	if p.Params.TestMode || *p.Params.Observation.WindowMS != 60000 || *p.Params.ExecutionRule.ExecutionTTLMS != 30000 {
		t.Fatal("start defaults differ")
	}
	// Explicit false is a valid new start, not a missing setting.
	if err := json.Unmarshal([]byte(strings.Replace(runStartJSON, `"marketType":"spot"`, `"marketType":"spot","testMode":false`, 1)), &p); err != nil {
		t.Fatal(err)
	}
	if p.Params.TestMode {
		t.Fatal("false mode changed")
	}
}

// TestRunDurationBoundaries verifies every duration before conversion to time.Duration.
//
// Version:
//   - 2026-10-01: Added.
func TestRunDurationBoundaries(t *testing.T) {
	for name, set := range map[string]func(*RunConfiguration, uint64){
		"trade age":    func(p *RunConfiguration, n uint64) { p.Observation.MaximumTradeAgeMS = &n },
		"snapshot age": func(p *RunConfiguration, n uint64) { p.Observation.MaximumSnapshotAgeMS = &n },
		"interval":     func(p *RunConfiguration, n uint64) { p.ExecutionRule.MinimumOrderIntervalMS = &n },
		"ttl":          func(p *RunConfiguration, n uint64) { p.ExecutionRule.ExecutionTTLMS = &n },
		"holding":      func(p *RunConfiguration, n uint64) { p.ExecutionRule.Exit.MaximumHoldingMS = &n },
	} {
		t.Run(name, func(t *testing.T) {
			p := runConfig()
			set(&p, MaximumRunDurationMS)
			if err := p.Validate(); err != nil {
				t.Fatal(err)
			}
			set(&p, MaximumRunDurationMS+1)
			if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("overflow accepted: %v", err)
			}
		})
	}
}
