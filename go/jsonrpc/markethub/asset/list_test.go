package asset

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

// TestListParamsStrictDecoding checks optional defaults and atomic strict decoding.
//
// Version:
//   - 2026-10-21: Added.
func TestListParamsStrictDecoding(t *testing.T) {
	var p ListParams
	if err := json.Unmarshal([]byte(`{"assetType":"crypto"}`), &p); err != nil || p != (ListParams{AssetType: "crypto", Page: 1, Limit: 100}) {
		t.Fatal(p, err)
	}
	for _, raw := range []string{
		`null`, `[]`, `{}`, `{"assetType":null}`, `{"assetType":"fx"}`, `{"assetType":""}`,
		`{"assetType":"crypto","asset":"sui"}`, `{"assetType":"crypto","marketType":"perp"}`,
		`{"assetType":"crypto","marketType":""}`, `{"assetType":"crypto","search":"a\nb"}`,
		`{"assetType":"crypto","Search":"a","search":"b"}`, `{"assetType":"crypto"}{}`,
		`{"assetType":"crypto","search":"` + strings.Repeat("あ", 65) + `"}`,
	} {
		before := p
		if err := json.Unmarshal([]byte(raw), &p); err == nil || p != before {
			t.Fatalf("accepted or mutated for %s: %v", raw, err)
		}
	}
	for _, field := range []string{"marketType", "search", "page", "limit", "Page", "Limit", "MarketType"} {
		raw := `{"assetType":"crypto","` + field + `":null}`
		before := p
		if err := json.Unmarshal([]byte(raw), &p); err == nil || p != before {
			t.Fatalf("accepted null: %s", field)
		}
	}
	for _, field := range []string{"page", "limit"} {
		for _, value := range []string{"0", "-1", "1.5", `"1"`, "9223372036854775808"} {
			if err := json.Unmarshal([]byte(`{"assetType":"crypto","`+field+`":`+value+`}`), &p); err == nil {
				t.Fatal(field, value)
			}
		}
	}
	for _, limit := range []int{1, 500} {
		p := ListParams{AssetType: "crypto", MarketType: "perpetual", Search: strings.Repeat("あ", 64), Limit: limit}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		var got ListParams
		if err := json.Unmarshal(data, &got); err != nil || got != p.Normalize() {
			t.Fatal(got, err)
		}
	}
	for _, bad := range []ListParams{{}, {AssetType: "crypto", Limit: 501}, {AssetType: "crypto", Page: -1}, {AssetType: "crypto", Search: string([]byte{0xff})}} {
		if !errors.Is(bad.Validate(), apperror.InvalidParameter()) {
			t.Fatal("validation cause lost")
		}
	}
	var absent *ListParams
	if err := absent.UnmarshalJSON([]byte(`{"assetType":"crypto"}`)); err == nil {
		t.Fatal("nil receiver accepted")
	}
}

// TestListFilters preserves complete coverage and searches each identity field independently.
//
// Version:
//   - 2026-10-21: Added.
func TestListFilters(t *testing.T) {
	entry := ListedAsset{Asset: "sui-token-ref", Symbol: "SUI", Name: "Sui Network", MarketTypes: []market.MarketType{"spot", "perpetual"}}
	for _, search := range []string{"", "TOKEN-REF", "sUi", "NETWORK"} {
		for _, kind := range []market.MarketType{"", "spot", "perpetual"} {
			if !(ListParams{Search: search, MarketType: kind}).Matches(entry) {
				t.Fatal(search, kind)
			}
		}
	}
	for _, p := range []ListParams{{Search: "bitcoin"}, {Search: "refSUI"}, {MarketType: "fx"}} {
		if p.Matches(entry) {
			t.Fatal("incorrect match", p)
		}
	}
	if !reflect.DeepEqual(entry.MarketTypes, []market.MarketType{"spot", "perpetual"}) {
		t.Fatal("mutated coverage")
	}
}

func listResultFixture() ListResult {
	return ListResult{AssetType: "crypto", Assets: []ListedAsset{
		{Asset: "sui", Symbol: "SUI", Name: "Sui", MarketTypes: []market.MarketType{"spot", "perpetual"}},
		{Asset: "sui-other", Symbol: "SUI", Name: "Other", MarketTypes: []market.MarketType{"spot"}},
	}, Page: 1, Limit: 100, Total: 2, SentAt: 1792548000123456}
}

// TestListResultContract checks complete pages, duplicate identity, ordering and request binding.
//
// Version:
//   - 2026-10-21: Added.
func TestListResultContract(t *testing.T) {
	valid := listResultFixture()
	data, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	var got ListResult
	if err := json.Unmarshal(data, &got); err != nil || !reflect.DeepEqual(got, valid) {
		t.Fatal(got, err)
	}
	if err := got.ValidateFor(ListParams{AssetType: "crypto", MarketType: "spot", Search: "sui"}); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*ListResult){
		"nil assets":     func(r *ListResult) { r.Assets = nil },
		"short page":     func(r *ListResult) { r.Assets = r.Assets[:1] },
		"zero page":      func(r *ListResult) { r.Page = 0 },
		"zero limit":     func(r *ListResult) { r.Limit = 0 },
		"limit overflow": func(r *ListResult) { r.Limit = 501 },
		"negative total": func(r *ListResult) { r.Total = -1 },
		"unsent":         func(r *ListResult) { r.SentAt = 0 },
		"wrong class":    func(r *ListResult) { r.AssetType = "fx" },
		"metadata":       func(r *ListResult) { r.Assets[0].Name = "" },
		"duplicate":      func(r *ListResult) { r.Assets[1].Asset = "sui"; r.Assets[1].Symbol = "ZZZ" },
		"reversed":       func(r *ListResult) { r.Assets[0], r.Assets[1] = r.Assets[1], r.Assets[0] },
		"type order":     func(r *ListResult) { r.Assets[0].MarketTypes = []market.MarketType{"perpetual", "spot"} },
	} {
		t.Run(name, func(t *testing.T) {
			r := listResultFixture()
			mutate(&r)
			if r.Validate() == nil {
				t.Fatal("accepted invalid response")
			}
		})
	}
	for _, p := range []ListParams{{AssetType: "crypto", Page: 2}, {AssetType: "crypto", Limit: 25}, {AssetType: "crypto", Search: "bitcoin"}, {AssetType: "crypto", MarketType: "perpetual"}} {
		if valid.ValidateFor(p) == nil {
			t.Fatal("response out of scope", p)
		}
	}
	for _, tc := range []struct{ total, page int }{{0, 1}, {2, 2}, {2, int(^uint(0) >> 1)}} {
		r := ListResult{AssetType: "crypto", Assets: []ListedAsset{}, Total: tc.total, Page: tc.page, Limit: 100, SentAt: valid.SentAt}
		if err := r.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}
