package candles

import (
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
)

// TestParamsValidate checks public limits, UTC alignment, ranges and asset selectors.
//
// Version:
//   - 2026-10-19: Update required crypto asset selectors.
//   - 2026-10-11: Added.
func TestParamsValidate(t *testing.T) {
	from := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC).Unix()
	base := Params{AssetType: "crypto", Asset: "btc", MarketType: "spot", From: from, To: from + 86400, Interval: "1m"}
	for _, limit := range []int{0, 1, 300} {
		for _, interval := range []string{"1m", "5m", "15m", "1h", "4h", "1d"} {
			p := base
			p.Interval, p.Limit, p.To = interval, limit, from+7*86400
			if err := p.Validate(); err != nil {
				t.Fatalf("valid request rejected: %+v %v", p, err)
			}
		}
	}
	for name, mutate := range map[string]func(*Params){
		"limit_negative":   func(p *Params) { p.Limit = -1 },
		"limit_301":        func(p *Params) { p.Limit = 301 },
		"range_eight_days": func(p *Params) { p.To = p.From + 8*86400 },
		"empty_range":      func(p *Params) { p.To = p.From },
		"reversed_range":   func(p *Params) { p.To = p.From - 60 },
		"zero_start":       func(p *Params) { p.From = 0 },
		"overflow_range":   func(p *Params) { p.From, p.To = math.MinInt64, math.MaxInt64 },
		"microseconds":     func(p *Params) { p.From *= 1e6; p.To *= 1e6 },
		"start_alignment":  func(p *Params) { p.From++ },
		"end_alignment":    func(p *Params) { p.To++ },
		"invalid_interval": func(p *Params) { p.Interval = "2m" },
		"empty_asset":      func(p *Params) { p.Asset = "" },
		"path_asset":       func(p *Params) { p.Asset = "../btc" },
		"long_asset":       func(p *Params) { p.Asset = strings.Repeat("b", 65) },
		"invalid_market":   func(p *Params) { p.MarketType = "future" },
	} {
		t.Run(name, func(t *testing.T) {
			p := base
			mutate(&p)
			if err := p.Validate(); !errors.Is(err, apperror.InvalidParameter()) {
				t.Fatalf("invalid parameters accepted: %v", err)
			}
		})
	}
	base.MarketType = "perp"
	if err := base.Validate(); err != nil {
		t.Fatal(err)
	}
}

// TestCandleJSONContract preserves nullable prices and publication units without cursor fields.
//
// Version:
//   - 2026-10-19: Include USD currency and the public send timestamp.
//   - 2026-10-19: Update required crypto asset selectors.
//   - 2026-10-11: Added.
//   - 2026-10-18: Preserve the explicit calculation method in historical results.
func TestCandleJSONContract(t *testing.T) {
	const wire = `{"definition":"composite","method":"volume_weighted_reference","asset":"btc","marketType":"spot","currency":"USD","interval":"1m","from":1791417600,"to":1791417660,"asOf":1791418000,"sentAt":1791418000123456,"candles":[{"time":1791417600,"open":null,"high":null,"low":null,"close":null,"volume":0,"quoteVolume":0,"tradeCount":0,"quality":0,"usable":false,"closed":true,"revision":"stored","publishedAt":1791417662000000,"coverage":{"known":true,"expectedSourceMinutes":1,"coveredSourceMinutes":1,"contributingSourceMinutes":0,"missingMinutes":0}}],"missingMinutes":0}`
	var result Result
	if err := json.Unmarshal([]byte(wire), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Candles) != 1 || result.Candles[0].Open != nil || result.Candles[0].PublishedAt != 1791417662000000 {
		t.Fatal("nullable prices or publication units changed")
	}
	encoded, err := json.Marshal(result)
	if err != nil || string(encoded) != wire {
		t.Fatalf("result wire contract changed: %s %v", encoded, err)
	}
	params, err := json.Marshal(Params{AssetType: "crypto", Asset: "btc", MarketType: "spot", From: result.From, To: result.To, Interval: "1m"})
	if err != nil || strings.Contains(string(params), "cursor") || strings.Contains(string(params), "limit") {
		t.Fatalf("unexpected default request fields: %s %v", params, err)
	}
}
