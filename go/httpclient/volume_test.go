package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/volume"
)

func volumeResponse(p volume.Params) volume.Result {
	to := time.Date(2026, 10, 19, 12, 10, 30, 123000, time.UTC)
	r := volume.Result{Asset: p.Asset, MarketType: p.MarketType, Definition: "composite", Currency: "USD", Method: "sum", SentAt: to.UnixMicro() + 1000}
	for _, period := range p.Normalize().Periods {
		r.Volumes = append(r.Volumes, volume.PeriodVolume{Period: period, From: to.Add(-period.Duration()).Truncate(time.Minute).UnixMicro(), To: to.UnixMicro(), CoveragePercentage: "0", CoverageStatus: volume.CoverageStatusUnavailable})
	}
	return r
}

// TestVolumeComposition verifies constructor wiring and one request for each whole snapshot.
//
// Version:
//   - 2026-10-19: Added.
func TestVolumeComposition(t *testing.T) {
	calls := 0
	client, err := NewClient(Config{EndpointURL: "https://api.example.test/", HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var req jsonrpc.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if err := r.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if req.Method != jsonrpc.MethodMarketHubVolumeGet || r.Method != http.MethodPost {
			t.Fatal("wrong method")
		}
		if string(req.Method) != "MarketHub.Volume.Get" {
			t.Fatal("wrong wire method")
		}
		var p volume.Params
		if err := json.Unmarshal(req.Params, &p); err != nil {
			t.Fatal(err)
		}
		if p.AssetType != market.AssetClassCrypto || p.Asset != "sui" || p.MarketType != "spot" {
			t.Fatal("lost selectors")
		}
		if calls == 1 && len(p.Periods) != 6 {
			t.Fatal("omitted periods were not normalized")
		}
		if calls == 2 && !slices.Equal(p.Periods, []volume.Period{volume.Period24h, volume.Period5m}) {
			t.Fatal("explicit periods reordered")
		}
		return testResponse(t, req, volumeResponse(p), nil, 200), nil
	})})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 || client.MarketHub().Volume() == nil || client.MarketHub().Price().Change() == nil || client.MarketHub().Candles() == nil {
		t.Fatal("incomplete or eager composition")
	}
	p := volume.Params{AssetType: market.AssetClassCrypto, Asset: "sui", MarketType: market.MarketTypeSpot}
	r, err := client.MarketHub().Volume().Get(t.Context(), p)
	if err != nil || r == nil || len(r.Volumes) != 6 || len(p.Periods) != 0 || calls != 1 {
		t.Fatalf("invalid default snapshot: %v", err)
	}
	p.Periods = []volume.Period{volume.Period24h, volume.Period5m}
	if _, err := client.MarketHub().Volume().Get(t.Context(), p); err != nil || calls != 2 {
		t.Fatalf("invalid selected snapshot: %v", err)
	}
}

// TestVolumeErrors preserves wrapped failures and rejects malformed or mismatched snapshots.
//
// Version:
//   - 2026-10-19: Added.
func TestVolumeErrors(t *testing.T) {
	p := volume.Params{AssetType: "crypto", Asset: "sui", MarketType: "spot", Periods: []volume.Period{volume.Period5m}}
	for name, mutate := range map[string]func(*volume.Result){
		"wrong asset":    func(r *volume.Result) { r.Asset = "btc" },
		"wrong market":   func(r *volume.Result) { r.MarketType = "perpetual" },
		"missing period": func(r *volume.Result) { r.Volumes = nil },
		"wrong period": func(r *volume.Result) {
			r.Volumes[0].Period = volume.Period15m
			r.Volumes[0].From -= 10 * time.Minute.Microseconds()
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := volumeResponse(p)
			mutate(&r)
			client, err := newVolumeClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(r)
			}))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Get(t.Context(), p); err == nil {
				t.Fatal("invalid snapshot accepted")
			}
		})
	}
	for _, raw := range []string{"null", "{", `{"volumes":null}`} {
		c, err := newVolumeClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
			return []byte(raw), nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Get(t.Context(), p); err == nil {
			t.Fatal("malformed response accepted")
		}
	}
	calls := 0
	cause := apperror.OutOfTicks()
	c, err := newVolumeClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		calls++
		return nil, cause
	}))
	if err != nil {
		t.Fatal(err)
	}
	invalid := p
	invalid.AssetType = ""
	if _, err := c.Get(t.Context(), invalid); !errors.Is(err, apperror.InvalidParameter()) || calls != 0 {
		t.Fatal("invalid selectors reached transport")
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := c.Get(canceled, p); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("canceled request reached transport")
	}
	if _, err := c.Get(nil, p); err == nil || calls != 0 {
		t.Fatal("nil context accepted")
	}
	if _, err := c.Get(t.Context(), p); !errors.Is(err, cause) || calls != 1 {
		t.Fatal("transport error lost or retried")
	}
	var absent *VolumeClient
	if _, err := absent.Get(t.Context(), p); err == nil {
		t.Fatal("nil client accepted")
	}
	var group *MarketHubClient
	if group.Volume() != nil {
		t.Fatal("nil group returned a client")
	}
	if _, err := newVolumeClient(nil); err == nil {
		t.Fatal("nil caller accepted")
	}
}
