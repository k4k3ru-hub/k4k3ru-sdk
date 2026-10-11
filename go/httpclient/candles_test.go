package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/candles"
)

type callerFunc func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error)

func (f callerFunc) call(ctx context.Context, method jsonrpc.Method, params json.RawMessage) (json.RawMessage, error) {
	return f(ctx, method, params)
}

// TestCandlesGet continues across gaps using immutable per-call parameters and response.To.
//
// Version:
//   - 2026-10-11: Added.
func TestCandlesGet(t *testing.T) {
	params := testParams()
	original := params
	calls := 0
	client, err := newCandlesClient(callerFunc(func(_ context.Context, method jsonrpc.Method, raw json.RawMessage) (json.RawMessage, error) {
		calls++
		if method != jsonrpc.MethodMarketHubCandlesGet || strings.Contains(string(raw), "cursor") {
			t.Fatal("incorrect operation contract")
		}
		var p candles.Params
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("failed to decode fixture candle parameters: %w", err)
		}
		result := candles.Result{Definition: "composite", Asset: p.Asset, MarketType: p.MarketType, Interval: p.Interval, From: p.From, To: p.To, Candles: []candles.Candle{}}
		if calls == 1 {
			result.To = p.From + 180
			result.Candles = []candles.Candle{{Time: p.From + 120, Revision: "stored", Closed: true}}
			result.MissingMinutes = 2
		} else {
			result.MissingMinutes = int((p.To - p.From) / 60)
		}
		return json.Marshal(result)
	}))
	if err != nil {
		t.Fatal(err)
	}
	first, err := client.Get(t.Context(), params)
	if err != nil || calls != 1 || params != original || first.To != params.From+180 || first.Candles[0].Open != nil {
		t.Fatalf("first call changed parameters or fetched extra pages: %+v %v", first, err)
	}
	params.From = first.To
	second, err := client.Get(t.Context(), params)
	if err != nil || calls != 2 || len(second.Candles) != 0 || second.To != original.To {
		t.Fatalf("empty final range did not finish: %+v %v", second, err)
	}
	params.Limit = 301
	if _, err := client.Get(t.Context(), params); !errors.Is(err, apperror.InvalidParameter()) || calls != 2 {
		t.Fatalf("invalid parameters reached transport: %v", err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := client.Get(canceled, original); !errors.Is(err, context.Canceled) || calls != 2 {
		t.Fatalf("cancellation reached transport or lost its cause: %v", err)
	}
}

// TestCandlesRejectMalformedPages prevents continuation loops, duplicate bars and incorrect selectors.
//
// Version:
//   - 2026-10-11: Added.
func TestCandlesRejectMalformedPages(t *testing.T) {
	params := testParams()
	for name, mutate := range map[string]func(*candles.Result){
		"wrong_asset":             func(r *candles.Result) { r.Asset = "eth" },
		"wrong_start":             func(r *candles.Result) { r.From-- },
		"no_progress":             func(r *candles.Result) { r.To = r.From },
		"end_outside_request":     func(r *candles.Result) { r.To = params.To + 60 },
		"unaligned_end":           func(r *candles.Result) { r.To++ },
		"nil_candles":             func(r *candles.Result) { r.Candles = nil },
		"empty_intermediate_page": func(r *candles.Result) { r.Candles = []candles.Candle{} },
		"over_limit":              func(r *candles.Result) { r.Candles = append(r.Candles, r.Candles[0]) },
		"bar_outside_range":       func(r *candles.Result) { r.Candles[0].Time = r.To },
		"bar_unaligned":           func(r *candles.Result) { r.Candles[0].Time++ },
		"skipped_next_bar":        func(r *candles.Result) { r.To += 60 },
	} {
		t.Run(name, func(t *testing.T) {
			value := candles.Result{Definition: "composite", Asset: params.Asset, MarketType: params.MarketType, Interval: params.Interval, From: params.From, To: params.From + 60, Candles: []candles.Candle{{Time: params.From}}}
			mutate(&value)
			client, err := newCandlesClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(value)
			}))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Get(t.Context(), params); err == nil {
				t.Fatal("invalid page accepted")
			}
		})
	}
	for i, raw := range []string{"null", "{", `{"candles":null}`} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			client, err := newCandlesClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(raw), nil
			}))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Get(t.Context(), params); err == nil {
				t.Fatal("malformed result accepted")
			}
		})
	}
}
