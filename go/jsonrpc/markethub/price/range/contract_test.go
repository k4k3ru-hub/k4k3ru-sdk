package pricerange

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func pointer[T any](value T) *T { return &value }

func resultFixture() Result {
	to := time.Date(2026, 10, 19, 12, 10, 30, 123000, time.UTC)
	return Result{Asset: "sui", MarketType: "spot", Definition: "composite", Currency: "USD", Method: "volume_weighted_latest", SentAt: to.UnixMicro() + 1000,
		Ranges: []Range{
			{Period: Period24h, From: to.Add(-24 * time.Hour).Truncate(time.Minute).UnixMicro(), To: to.UnixMicro(), Low: pointer("0.9"), High: pointer("1.1"), CoveragePercentage: "50", CoverageStatus: CoverageStatusSufficient},
			{Period: Period7d, From: to.Add(-7 * 24 * time.Hour).Truncate(time.Minute).UnixMicro(), To: to.UnixMicro(), Low: pointer("0.9"), High: pointer("1.1"), CoveragePercentage: "7.142857142857142857", CoverageStatus: CoverageStatusInsufficient},
			{Period: Period5m, From: to.Add(-5 * time.Minute).Truncate(time.Minute).UnixMicro(), To: to.UnixMicro(), CoveragePercentage: "0", CoverageStatus: CoverageStatusUnavailable},
		},
	}
}

// TestRangeParameters verifies all period durations, immutable defaults and invalid selector errors.
//
// Version:
//   - 2026-10-19: Added.
func TestRangeParameters(t *testing.T) {
	p := Params{AssetType: "crypto", Asset: "sui", MarketType: "spot"}
	want := []Period{Period5m, Period15m, Period1h, Period4h, Period24h, Period7d}
	for _, raw := range []string{`{"assetType":"crypto","asset":"sui","marketType":"spot"}`, `{"assetType":"crypto","asset":"sui","marketType":"spot","periods":null}`, `{"assetType":"crypto","asset":"sui","marketType":"spot","periods":[]}`} {
		var original Params
		if err := json.Unmarshal([]byte(raw), &original); err != nil {
			t.Fatal(err)
		}
		if err := original.Validate(); err != nil {
			t.Fatal(err)
		}
		n := original.Normalize()
		if !slices.Equal(n.Periods, want) || len(original.Periods) != 0 {
			t.Fatal("invalid defaults or input mutation")
		}
		n.Periods[0] = Period7d
		if !slices.Equal(original.Normalize().Periods, want) {
			t.Fatal("shared defaults")
		}
	}
	p.Periods = []Period{Period24h, Period5m}
	n := p.Normalize()
	if !slices.Equal(n.Periods, p.Periods) {
		t.Fatal("selection reordered")
	}
	n.Periods[0] = Period7d
	if p.Periods[0] != Period24h {
		t.Fatal("shared selection storage")
	}
	for period, duration := range map[Period]time.Duration{Period5m: 5 * time.Minute, Period15m: 15 * time.Minute, Period1h: time.Hour, Period4h: 4 * time.Hour, Period24h: 24 * time.Hour, Period7d: 168 * time.Hour, "1d": 0} {
		if period.Duration() != duration {
			t.Fatalf("wrong duration: %s", period)
		}
	}
	for _, periods := range [][]Period{{Period5m, Period5m}, {"1d"}, {""}, make([]Period, 7), {Period(strings.Repeat("private", 100))}} {
		p.Periods = periods
		if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) || strings.Contains(err.Error(), "private") {
			t.Fatalf("invalid period error: %v", err)
		}
	}
}

// TestRangeResultRoundTrip preserves partial data, null values and distinct microsecond times.
//
// Version:
//   - 2026-10-19: Added.
func TestRangeResultRoundTrip(t *testing.T) {
	r := resultFixture()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Result
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r, decoded) || !strings.Contains(string(raw), `"low":null,"high":null`) {
		t.Fatal("round trip changed range evidence")
	}
	if r.Ranges[0].To == r.SentAt {
		t.Fatal("fixture hides calculation/send time distinction")
	}
	for _, coverage := range []string{"50", "50.000000000000000001", "100"} {
		r.Ranges[0].CoveragePercentage = coverage
		if err := r.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	r.Ranges[0].CoverageStatus = CoverageStatusInsufficient
	for _, coverage := range []string{"0", "49.999999999999999999"} {
		r.Ranges[0].CoveragePercentage = coverage
		if err := r.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for i := range r.Ranges {
		r.Ranges[i].To = time.UnixMicro(r.Ranges[i].To).Truncate(time.Minute).UnixMicro()
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("exact minute boundary rejected: %v", err)
	}
	forming := resultFixture()
	forming.Ranges = forming.Ranges[2:]
	forming.Ranges[0].Low, forming.Ranges[0].High = pointer("1"), pointer("1")
	forming.Ranges[0].CoverageStatus = CoverageStatusInsufficient
	if err := forming.Validate(); err != nil {
		t.Fatalf("developing-only prices with zero closed coverage rejected: %v", err)
	}
}

// TestRangeRejectsContradictions rejects impossible coverage, prices and calculation boundaries.
//
// Version:
//   - 2026-10-19: Added.
func TestRangeRejectsContradictions(t *testing.T) {
	for name, mutate := range map[string]func(*Result){
		"metadata":             func(r *Result) { r.Method = "vwap" },
		"no ranges":            func(r *Result) { r.Ranges = nil },
		"too many":             func(r *Result) { r.Ranges = make([]Range, 7) },
		"duplicate":            func(r *Result) { r.Ranges[1].Period = r.Ranges[0].Period },
		"unsupported":          func(r *Result) { r.Ranges[0].Period = "1d" },
		"missing low":          func(r *Result) { r.Ranges[0].Low = nil },
		"reversed prices":      func(r *Result) { r.Ranges[0].Low = pointer("2") },
		"coverage missing":     func(r *Result) { r.Ranges[0].CoveragePercentage = "" },
		"coverage above 100":   func(r *Result) { r.Ranges[0].CoveragePercentage = "100.01" },
		"below sufficient":     func(r *Result) { r.Ranges[0].CoveragePercentage = "49" },
		"above insufficient":   func(r *Result) { r.Ranges[1].CoveragePercentage = "50" },
		"unavailable values":   func(r *Result) { r.Ranges[0].CoverageStatus = CoverageStatusUnavailable },
		"unavailable coverage": func(r *Result) { r.Ranges[2].CoveragePercentage = "1" },
		"unknown status":       func(r *Result) { r.Ranges[0].CoverageStatus = "full" },
		"wrong boundary":       func(r *Result) { r.Ranges[0].From += time.Minute.Microseconds() },
		"unaligned boundary":   func(r *Result) { r.Ranges[0].From++ },
		"different snapshot":   func(r *Result) { r.Ranges[1].To++ },
		"future calculation":   func(r *Result) { r.SentAt = r.Ranges[0].To - 1 },
		"seconds":              func(r *Result) { r.Ranges[0].From /= 1e6 },
		"zero send time":       func(r *Result) { r.SentAt = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			r := resultFixture()
			mutate(&r)
			if r.Validate() == nil {
				t.Fatal("accepted contradictory result")
			}
		})
	}
	for _, value := range []string{"", "0", "-1", "NaN", "1e3", "1/2", strings.Repeat("9", 513)} {
		r := resultFixture()
		r.Ranges[0].Low = &value
		if r.Validate() == nil {
			t.Fatal("accepted invalid price")
		}
	}
}
