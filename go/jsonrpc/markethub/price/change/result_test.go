package change

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func pointer[T any](value T) *T { return &value }

func mixedResult() Result {
	return Result{
		Asset: "sui", MarketType: "spot", Definition: "composite", Currency: "USD", Method: "volume_weighted_latest",
		Price: pointer("1.02"), SentAt: 1792326930000000,
		Changes: []Change{
			{Period: Period5m, ChangePercentage: pointer("2"), ReferencePrice: pointer("1"), ReferenceAt: pointer(int64(1792326600000000)), Status: StatusLive},
			{Period: Period15m, ChangePercentage: pointer("6.25"), ReferencePrice: pointer("0.96"), ReferenceAt: pointer(int64(1792325880000000)), Status: StatusStale},
			{Period: Period24h, Status: StatusUnavailable},
		},
	}
}

// TestPartialResponseRoundTrip preserves per-period availability, exact decimals and microseconds.
//
// Version:
//   - 2026-10-18: Added.
func TestPartialResponseRoundTrip(t *testing.T) {
	value := mixedResult()
	if err := value.Validate(); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Result
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(value, decoded) {
		t.Fatal("response values changed on serialization")
	}
	if !strings.Contains(string(encoded), `"referenceAt":1792326600000000`) || !strings.Contains(string(encoded), `"changePercentage":null,"referencePrice":null,"referenceAt":null`) {
		t.Fatal("microseconds or explicit unavailable fields were lost")
	}
	value.Price = nil
	for i := range value.Changes {
		value.Changes[i].Status = StatusUnavailable
		value.Changes[i].ChangePercentage = nil
	}
	if err := value.Validate(); err != nil {
		t.Fatalf("known reference with unavailable current price rejected: %v", err)
	}
}

// TestReferenceTimeBounds rejects future, unaligned and seconds-based references without imposing an age cutoff.
//
// Version:
//   - 2026-10-18: Added.
func TestReferenceTimeBounds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		at     int64
		period Period
		valid  bool
	}{
		{"normal minute end", 1792326600000000, Period5m, true},
		{"older fallback", 1792325880000000, Period5m, true},
		{"seven day boundary anchor", 1791722100000000, Period7d, true},
		{"minute after target", 1792326660000000, Period5m, false},
		{"after response", 1792326960000000, Period5m, false},
		{"seconds instead of microseconds", 1792326600, Period5m, false},
		{"not a minute end", 1792326600000001, Period5m, false},
		{"zero", 0, Period5m, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := mixedResult()
			value.Changes = value.Changes[:1]
			value.Changes[0].Period = tc.period
			value.Changes[0].ReferenceAt = &tc.at
			value.Changes[0].Status = StatusStale
			if err := value.Validate(); (err == nil) != tc.valid {
				t.Fatalf("unexpected validity: %v", err)
			}
		})
	}
	value := mixedResult()
	value.SentAt = *value.Changes[0].ReferenceAt + (5 * time.Minute).Microseconds()
	if err := value.Validate(); err != nil {
		t.Fatalf("exact period boundary rejected: %v", err)
	}
}

// TestInvalidResponses rejects contradictory availability and malformed numerical evidence.
//
// Version:
//   - 2026-10-18: Added.
func TestInvalidResponses(t *testing.T) {
	mutations := map[string]func(*Result){
		"legacy method":              func(r *Result) { r.Method = "vwap" },
		"wrong currency":             func(r *Result) { r.Currency = "USDC" },
		"missing send time":          func(r *Result) { r.SentAt = 0 },
		"missing current":            func(r *Result) { r.Price = nil },
		"no periods":                 func(r *Result) { r.Changes = nil },
		"too many periods":           func(r *Result) { r.Changes = make([]Change, 7) },
		"repeated period":            func(r *Result) { r.Changes[1].Period = r.Changes[0].Period },
		"unsupported period":         func(r *Result) { r.Changes[0].Period = "1d" },
		"unknown status":             func(r *Result) { r.Changes[0].Status = "unknown" },
		"missing reference":          func(r *Result) { r.Changes[0].ReferencePrice = nil },
		"missing reference time":     func(r *Result) { r.Changes[0].ReferenceAt = nil },
		"missing percentage":         func(r *Result) { r.Changes[0].ChangePercentage = nil },
		"unavailable value":          func(r *Result) { r.Changes[0].Status = StatusUnavailable },
		"nonpositive reference":      func(r *Result) { r.Changes[0].ReferencePrice = pointer("0") },
		"percentage below minus 100": func(r *Result) { r.Changes[0].ChangePercentage = pointer("-100.0001") },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			value := mixedResult()
			mutate(&value)
			if value.Validate() == nil {
				t.Fatal("invalid response accepted")
			}
		})
	}
	for _, invalid := range []string{"", "0", "-1", "NaN", "1e3", "1/2", "+1", strings.Repeat("9", 513)} {
		value := mixedResult()
		value.Price = &invalid
		if value.Validate() == nil {
			t.Fatal("invalid current price accepted")
		}
	}
	for _, invalid := range []string{"", "NaN", "1e3", "1/2", "+1", strings.Repeat("9", 513)} {
		value := mixedResult()
		value.Changes[0].ChangePercentage = &invalid
		if value.Validate() == nil {
			t.Fatal("invalid change percentage accepted")
		}
	}
	for _, valid := range []string{"0", "-2.5", "200", "0.000000000000000001"} {
		value := mixedResult()
		value.Changes[0].ChangePercentage = &valid
		if err := value.Validate(); err != nil {
			t.Fatalf("signed decimal rejected: %v", err)
		}
	}
}
