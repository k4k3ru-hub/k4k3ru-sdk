package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/authentication"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/change"
)

func unavailableChanges(params change.Params) change.Result {
	result := change.Result{
		Asset: params.Asset, MarketType: params.MarketType, Definition: "composite",
		Currency: "USD", Method: "volume_weighted_latest", SentAt: 1792326930000000,
	}
	for _, period := range params.Normalize().Periods {
		result.Changes = append(result.Changes, change.Change{Period: period, Status: change.StatusUnavailable})
	}
	return result
}

// TestPriceChangeComposition verifies constructor wiring, shared authentication and one request for all selected periods.
//
// Version:
//   - 2026-10-19: Update required crypto asset selectors.
//   - 2026-10-18: Added.
func TestPriceChangeComposition(t *testing.T) {
	for _, signed := range []bool{false, true} {
		name := "anonymous"
		if signed {
			name = "signed"
		}
		t.Run(name, func(t *testing.T) {
			calls := 0
			now := time.UnixMicro(1792326930000000)
			credential := authentication.Credential{APIKey: "price-change-test-key", SecretKey: "price-change-test-secret", SignatureAlgorithm: authentication.SignatureAlgorithmHMACSHA256}
			config := Config{
				EndpointURL: "https://api.example.test/", Now: func() time.Time { return now }, Nonce: func() (string, error) { return "test-nonce", nil },
				HTTPClient: doerFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					var request jsonrpc.Request
					decodeErr := json.NewDecoder(r.Body).Decode(&request)
					closeErr := r.Body.Close()
					if decodeErr != nil || closeErr != nil {
						t.Fatalf("request unreadable: %v %v", decodeErr, closeErr)
					}
					if request.Method != jsonrpc.MethodMarketHubPriceChangeGet || string(request.Method) != "MarketHub.Price.Change.Get" || r.Method != http.MethodPost {
						t.Fatal("incorrect operation")
					}
					if signed {
						want, err := authentication.SignRequest(request.Method, request.Params, credential, now.Unix(), "test-nonce")
						if err != nil || !reflect.DeepEqual(request.Auth, want) {
							t.Fatalf("authentication changed: %v", err)
						}
					} else if request.Auth != nil {
						t.Fatal("anonymous request unexpectedly signed")
					}
					var params change.Params
					if err := json.Unmarshal(request.Params, &params); err != nil {
						t.Fatal(err)
					}
					want := (change.Params{}).Normalize().Periods
					if calls == 2 {
						want = []change.Period{change.Period24h, change.Period5m}
					}
					if params.Asset != "sui" || params.MarketType != "perpetual" || !slices.Equal(params.Periods, want) {
						t.Fatal("selectors or period defaults changed")
					}
					result := unavailableChanges(params)
					slices.Reverse(result.Changes) // Period identity must not depend on response order.
					return testResponse(t, request, result, nil, http.StatusOK), nil
				}),
			}
			if signed {
				config.CredentialProvider = credentialFunc(func(context.Context) (authentication.Credential, error) { return credential, nil })
			}
			client, err := NewClient(config)
			if err != nil {
				t.Fatal(err)
			}
			if calls != 0 || client.MarketHub().Candles() == nil || client.MarketHub().Price() == nil || client.MarketHub().Price().Change() == nil {
				t.Fatal("composition made a request or omitted an operation")
			}
			for i, periods := range [][]change.Period{nil, {change.Period24h, change.Period5m}} {
				params := change.Params{AssetType: "crypto", Asset: "sui", MarketType: "perpetual", Periods: periods}
				original := slices.Clone(periods)
				result, err := client.MarketHub().Price().Change().Get(t.Context(), params)
				if err != nil || result == nil || calls != i+1 || !slices.Equal(original, params.Periods) || len(result.Changes) != len(params.Normalize().Periods) {
					t.Fatalf("incorrect response or request mutation: %v", err)
				}
			}
		})
	}
}

// TestPriceChangePreservesPartialSnapshot returns available periods even when another period is unavailable.
//
// Version:
//   - 2026-10-19: Update required crypto asset selectors.
//   - 2026-10-18: Added.
func TestPriceChangePreservesPartialSnapshot(t *testing.T) {
	raw := json.RawMessage(`{
		"asset":"sui","marketType":"spot","definition":"composite","currency":"USD",
		"method":"volume_weighted_latest","price":"1.02","sentAt":1792326930000000,
		"changes":[
			{"period":"5m","changePercentage":"2","referencePrice":"1","referenceAt":1792326600000000,"status":"live"},
			{"period":"15m","changePercentage":"6.25","referencePrice":"0.96","referenceAt":1792325880000000,"status":"stale"},
			{"period":"24h","changePercentage":null,"referencePrice":null,"referenceAt":null,"status":"unavailable"}
		]
	}`)
	client, err := newPriceChangeClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) { return raw, nil }))
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Get(t.Context(), change.Params{AssetType: "crypto", Asset: "sui", MarketType: "spot", Periods: []change.Period{change.Period5m, change.Period15m, change.Period24h}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Price == nil || *result.Price != "1.02" || *result.Changes[0].ChangePercentage != "2" || *result.Changes[1].ReferenceAt != 1792325880000000 || result.Changes[2].ChangePercentage != nil {
		t.Fatal("partial snapshot lost a price, reference or availability")
	}
}

// TestPriceChangeRejectsMalformedResults rejects wrong selectors, missing periods and invalid payloads.
//
// Version:
//   - 2026-10-19: Update required crypto asset selectors.
//   - 2026-10-18: Added.
func TestPriceChangeRejectsMalformedResults(t *testing.T) {
	params := change.Params{AssetType: "crypto", Asset: "sui", MarketType: "spot", Periods: []change.Period{change.Period5m, change.Period24h}}
	for name, mutate := range map[string]func(*change.Result){
		"wrong asset":      func(r *change.Result) { r.Asset = "eth" },
		"wrong market":     func(r *change.Result) { r.MarketType = "perpetual" },
		"missing period":   func(r *change.Result) { r.Changes = r.Changes[:1] },
		"different period": func(r *change.Result) { r.Changes[1].Period = change.Period7d },
		"duplicate period": func(r *change.Result) { r.Changes[1].Period = r.Changes[0].Period },
		"extra period": func(r *change.Result) {
			r.Changes = append(r.Changes, change.Change{Period: change.Period7d, Status: change.StatusUnavailable})
		},
	} {
		t.Run(name, func(t *testing.T) {
			value := unavailableChanges(params)
			mutate(&value)
			client, err := newPriceChangeClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(value)
			}))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.Get(t.Context(), params); err == nil {
				t.Fatal("invalid response accepted")
			}
		})
	}
	for _, raw := range []string{"null", "{}", "{", "[]"} {
		client, err := newPriceChangeClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(raw), nil
		}))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.Get(t.Context(), params); err == nil {
			t.Fatal("malformed response accepted")
		}
	}
}

// TestPriceChangeErrors preserves errors and avoids transport calls for invalid or canceled requests.
//
// Version:
//   - 2026-10-19: Update required crypto asset selectors.
//   - 2026-10-18: Added.
func TestPriceChangeErrors(t *testing.T) {
	params := change.Params{AssetType: "crypto", Asset: "sui", MarketType: "spot"}
	calls := 0
	want := apperror.OutOfTicks()
	client, err := newPriceChangeClient(callerFunc(func(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error) {
		calls++
		return nil, want
	}))
	if err != nil {
		t.Fatal(err)
	}
	invalid := params
	invalid.Periods = []change.Period{change.Period5m, change.Period5m}
	if _, err := client.Get(t.Context(), invalid); !errors.Is(err, apperror.InvalidParameter()) || calls != 0 {
		t.Fatalf("invalid request reached transport or error chain lost: %v", err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := client.Get(canceled, params); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("cancellation reached transport or error chain lost: %v", err)
	}
	if _, err := client.Get(nil, params); err == nil || calls != 0 {
		t.Fatal("nil context reached transport")
	}
	if _, err := client.Get(t.Context(), params); !errors.Is(err, want) || calls != 1 {
		t.Fatalf("transport error lost or request retried: %v", err)
	}
	var absent *PriceChangeClient
	if _, err := absent.Get(t.Context(), params); err == nil {
		t.Fatal("nil client accepted")
	}
	var priceClient *PriceClient
	if priceClient.Change() != nil {
		t.Fatal("nil price group has a change client")
	}
	if _, err := newPriceChangeClient(nil); err == nil {
		t.Fatal("nil caller accepted")
	}
	if _, err := newPriceClient(client.caller, nil, nil); err == nil {
		t.Fatal("missing nested operation accepted")
	}
}
