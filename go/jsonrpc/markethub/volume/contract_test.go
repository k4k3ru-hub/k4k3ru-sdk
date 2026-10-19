package volume

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

func volumePointer(value string) *string { return &value }

func volumeFixture() Result {
	to := time.Date(2026, 10, 19, 12, 10, 30, 123456000, time.UTC)
	return Result{Asset: "sui", MarketType: "spot", Definition: "composite", Currency: "USD", Method: "sum", SentAt: to.UnixMicro() + 1000,
		Volumes: []PeriodVolume{
			{Period: Period1h, Volume: volumePointer("0"), From: to.Add(-time.Hour).Truncate(time.Minute).UnixMicro(), To: to.UnixMicro(), CoveragePercentage: "50", CoverageStatus: CoverageStatusSufficient},
			{Period: Period5m, Volume: volumePointer("12.345"), From: to.Add(-5 * time.Minute).Truncate(time.Minute).UnixMicro(), To: to.UnixMicro(), CoveragePercentage: "0", CoverageStatus: CoverageStatusInsufficient},
			{Period: Period7d, From: to.Add(-7 * 24 * time.Hour).Truncate(time.Minute).UnixMicro(), To: to.UnixMicro(), CoveragePercentage: "0", CoverageStatus: CoverageStatusUnavailable},
		}}
}

// TestParams validates required selectors and a copied, bounded period selection.
//
// Version:
//   - 2026-10-19: Added.
func TestParams(t *testing.T) {
	p := Params{AssetType: "crypto", Asset: "sui", MarketType: "spot"}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	q := p.Normalize()
	if !slices.Equal(q.Periods, []Period{Period5m, Period15m, Period1h, Period4h, Period24h, Period7d}) || len(p.Periods) != 0 {
		t.Fatal("invalid default periods")
	}
	for _, period := range q.Periods {
		if period.Duration() <= 0 {
			t.Fatal("missing duration", period)
		}
	}
	p.Periods = []Period{Period7d, Period5m}
	q = p.Normalize()
	q.Periods[0] = Period15m
	if p.Periods[0] != Period7d {
		t.Fatal("mutated caller selection")
	}
	for name, mutate := range map[string]func(*Params){
		"missing asset type":     func(p *Params) { p.AssetType = "" },
		"unsupported asset type": func(p *Params) { p.AssetType = "fx" },
		"missing asset":          func(p *Params) { p.Asset = "" },
		"noncanonical asset":     func(p *Params) { p.Asset = "SUI" },
		"missing market":         func(p *Params) { p.MarketType = "" },
		"legacy perp":            func(p *Params) { p.MarketType = "perp" },
		"unsupported period":     func(p *Params) { p.Periods = []Period{"1m"} },
		"duplicate period":       func(p *Params) { p.Periods = []Period{Period5m, Period5m} },
		"too many periods":       func(p *Params) { p.Periods = make([]Period, 7) },
	} {
		t.Run(name, func(t *testing.T) {
			bad := p
			mutate(&bad)
			if !errors.Is(bad.Validate(), apperror.InvalidParameter()) {
				t.Fatal("invalid parameter cause lost")
			}
		})
	}
}

// TestResult distinguishes observed zero, positive partial turnover and missing data on the wire.
//
// Version:
//   - 2026-10-19: Added.
func TestResult(t *testing.T) {
	r := volumeFixture()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"volume":"0"`) || !strings.Contains(string(raw), `"volume":null`) || strings.Contains(string(raw), `observedAt`) {
		t.Fatal(string(raw))
	}
	var decoded Result
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := decoded.Validate(); err != nil || decoded.SentAt != r.SentAt {
		t.Fatal("round trip lost microseconds", err)
	}
	r.MarketType = "perpetual"
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestInvalidResults rejects malformed amounts, contradictory coverage and incompatible snapshots.
//
// Version:
//   - 2026-10-19: Added.
func TestInvalidResults(t *testing.T) {
	for name, mutate := range map[string]func(*Result){
		"wrong method":         func(r *Result) { r.Method = "volume_weighted_latest" },
		"wrong currency":       func(r *Result) { r.Currency = "USDC" },
		"wrong definition":     func(r *Result) { r.Definition = "venue" },
		"missing send time":    func(r *Result) { r.SentAt = 0 },
		"empty volumes":        func(r *Result) { r.Volumes = nil },
		"too many volumes":     func(r *Result) { r.Volumes = make([]PeriodVolume, 7) },
		"duplicate period":     func(r *Result) { r.Volumes[1] = r.Volumes[0] },
		"unknown period":       func(r *Result) { r.Volumes[0].Period = "1m" },
		"unequal times":        func(r *Result) { r.Volumes[1].To-- },
		"future evaluation":    func(r *Result) { r.SentAt = r.Volumes[0].To - 1 },
		"unaligned start":      func(r *Result) { r.Volumes[0].From++ },
		"wrong start":          func(r *Result) { r.Volumes[0].From -= time.Minute.Microseconds() },
		"missing volume":       func(r *Result) { r.Volumes[0].Volume = nil },
		"unavailable amount":   func(r *Result) { r.Volumes[2].Volume = volumePointer("0") },
		"unavailable coverage": func(r *Result) { r.Volumes[2].CoveragePercentage = "1" },
		"below sufficient":     func(r *Result) { r.Volumes[0].CoveragePercentage = "49.999" },
		"above insufficient":   func(r *Result) { r.Volumes[1].CoveragePercentage = "50" },
		"coverage too large":   func(r *Result) { r.Volumes[0].CoveragePercentage = "101" },
		"unknown status":       func(r *Result) { r.Volumes[0].CoverageStatus = "live" },
	} {
		t.Run(name, func(t *testing.T) {
			r := volumeFixture()
			mutate(&r)
			if err := r.Validate(); err == nil {
				t.Fatal("invalid response accepted")
			}
		})
	}
	for _, text := range []string{"", "-1", "NaN", "Infinity", "1e6", " 0", "+1", strings.Repeat("1", 513)} {
		r := volumeFixture()
		r.Volumes[0].Volume = &text
		if err := r.Validate(); err == nil {
			t.Fatal("invalid amount accepted")
		}
		r = volumeFixture()
		r.Volumes[0].CoveragePercentage = text
		if err := r.Validate(); err == nil {
			t.Fatal("invalid percentage accepted")
		}
	}
}
