package news

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestNewsSelectors rejects ambiguous or out-of-scope selectors before network work.
//
// Version:
//   - 2026-10-31: Added.
func TestNewsSelectors(t *testing.T) {
	for _, raw := range []string{`null`, `{}`, `{"assetType":"crypto","asset":"sui","network":"testnet"}`, `{"assetType":"crypto","asset":"sui","asset":"btc"}`, `{"assetType":"crypto","asset":"sui","page":2}`, `{"assetType":"crypto","asset":"sui","limit":0}`, `{"assetType":"crypto","asset":"sui","limit":null}`, `{"assetType":"crypto","asset":"sui","from":2,"to":1}`} {
		var p Params
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	var p Params
	if err := json.Unmarshal([]byte(`{"assetType":"crypto","asset":"sui"}`), &p); err != nil || p.Limit != 20 || p.Page != 1 {
		t.Fatal(p, err)
	}
}

// TestNewsItemRequiresCollectedRun rejects an item attributed to an uncollected source.
//
// Version:
//   - 2026-10-31: Added.
func TestNewsItemRequiresCollectedRun(t *testing.T) {
	p := Params{AssetType: "crypto", Asset: "sui"}
	r := Result{AssetType: "crypto", Asset: "sui", Network: "mainnet", Coverage: "latest_source_windows", Snapshot: "fixture", Page: 1, Limit: 20, Total: 1, SentAt: 10,
		Sources: []Source{{ID: "sui-status", Name: "Sui Status", URL: "https://status.sui.io", Status: "never_collected"}},
		Items:   []Item{{ID: strings.Repeat("a", 64), Revision: strings.Repeat("b", 64), SourceID: "sui-status", URL: "https://status.sui.io/incidents/fixture", Title: "Fixture", Kind: "incident", PublishedAt: 1, Network: "mainnet", NetworkEvidence: "fixture evidence"}},
	}
	if err := r.ValidateFor(p); err == nil {
		t.Fatal("accepted an item with no collected run")
	}
	r.Sources[0].RunID = strings.Repeat("c", 32)
	r.Sources[0].ObservedAt = 5
	r.Sources[0].ResponseSHA256 = strings.Repeat("d", 64)
	r.Sources[0].Status = "complete"
	r.Sources[0].ItemCount, r.Sources[0].MainnetCount = 1, 1
	r.Items[0].RunID, r.Items[0].ObservedAt = r.Sources[0].RunID, 5
	if err := r.ValidateFor(p); err != nil {
		t.Fatal(err)
	}
}

// TestNewsResultRejectsFalseCoverage checks missing source evidence and pagination boundaries.
//
// Version:
//   - 2026-10-31: Added.
func TestNewsResultRejectsFalseCoverage(t *testing.T) {
	p := Params{AssetType: "crypto", Asset: "sui"}
	r := Result{AssetType: "crypto", Asset: "sui", Network: "mainnet", Coverage: "latest_source_windows", Snapshot: "fixture", Page: 1, Limit: 20, Items: []Item{}, Sources: []Source{}, SentAt: 1}
	if err := r.ValidateFor(p); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Result){func(r *Result) { r.Network = "testnet" }, func(r *Result) { r.Asset = "btc" }, func(r *Result) { r.Total = 1 }, func(r *Result) { r.Items = nil }, func(r *Result) { r.Snapshot = "" }, func(r *Result) {
		r.Sources = []Source{{ID: "sui", Name: "Sui", URL: "javascript:alert(1)", Status: "complete"}}
	}} {
		bad := r
		change(&bad)
		if err := bad.ValidateFor(p); err == nil {
			t.Fatal("accepted malformed news result")
		}
	}
}

// TestNewsGenericProvenance validates immutable input fields and independent current health.
//
// Version:
//   - 2026-10-31: Added.
func TestNewsGenericProvenance(t *testing.T) {
	p := Params{AssetType: "crypto", Asset: "sui"}
	source := Source{ID: "sui-status", Name: "Sui Status", URL: "https://status.sui.io/", Status: "failed", RunID: strings.Repeat("a", 32), ObservedAt: 5, ItemCount: 1, MainnetCount: 1, ResponseSHA256: strings.Repeat("b", 64), Input: &Input{AnnotationSetID: strings.Repeat("c", 32), DocumentSHA256: strings.Repeat("d", 64), Profile: "sui-mainnet-v1", ParserVersion: "sui-context-v1", ProducerVersion: "sui-mainnet-v1", PublisherID: "sui", SourceName: "Sui Status", SourceURL: "https://status.sui.io/api/v2/incidents.json"}, Classification: &Classification{Profile: "sui-mainnet-v1", Status: "complete", LastAttemptAt: 6, LastSuccessAt: 7}}
	item := Item{ID: strings.Repeat("1", 64), Revision: strings.Repeat("2", 64), RunID: source.RunID, SourceID: source.ID, URL: "https://status.sui.io/incidents/fixture", Title: "Fixture", Kind: "incident", PublishedAt: 1, ObservedAt: 5, Network: "mainnet", NetworkEvidence: "fixture evidence", ArticleID: strings.Repeat("3", 64), RecordSHA256: strings.Repeat("4", 64), AnnotationSetID: source.Input.AnnotationSetID}
	result := Result{AssetType: "crypto", Asset: "sui", Network: "mainnet", Coverage: "latest_source_windows", Snapshot: "opaque", Page: 1, Limit: 20, Total: 1, SentAt: 10, Items: []Item{item}, Sources: []Source{source}}
	if err := result.ValidateFor(p); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*Result){
		"partial item":             func(r *Result) { r.Items[0].RecordSHA256 = "" },
		"wrong annotation":         func(r *Result) { r.Items[0].AnnotationSetID = strings.Repeat("e", 32) },
		"partial input":            func(r *Result) { r.Sources[0].Input.DocumentSHA256 = "" },
		"missing health":           func(r *Result) { r.Sources[0].Classification = nil },
		"profile mismatch":         func(r *Result) { r.Sources[0].Classification.Profile = "other" },
		"long token":               func(r *Result) { r.Sources[0].Input.ParserVersion = strings.Repeat("a", 65) },
		"control token":            func(r *Result) { r.Sources[0].Input.PublisherID = "sui\n" },
		"invalid token":            func(r *Result) { r.Sources[0].Input.Profile = "with space" },
		"source name bytes":        func(r *Result) { r.Sources[0].Input.SourceName = strings.Repeat("語", 86) },
		"source name control":      func(r *Result) { r.Sources[0].Input.SourceName = "Sui\nStatus" },
		"endpoint credentials":     func(r *Result) { r.Sources[0].Input.SourceURL = "https://secret@example.org/" },
		"complete without success": func(r *Result) { r.Sources[0].Classification.LastSuccessAt = 0 },
		"pending with attempt":     func(r *Result) { r.Sources[0].Classification.Status = "pending" },
		"never with success": func(r *Result) {
			r.Sources[0].Classification.Status = "never_classified"
			r.Sources[0].Classification.LastAttemptAt = 0
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			var changed Result
			if err := json.Unmarshal(data, &changed); err != nil {
				t.Fatal(err)
			}
			mutate(&changed)
			if err := changed.ValidateFor(p); err == nil {
				t.Fatal("malformed additive fields accepted")
			}
		})
	}
	// An older pinned empty window can report new current classification health.
	empty := result
	empty.Total = 0
	empty.Items = []Item{}
	empty.Sources = []Source{{ID: source.ID, Name: source.Name, URL: source.URL, Status: "complete", Classification: source.Classification}}
	if err := empty.ValidateFor(p); err != nil {
		t.Fatal("old empty pin rejected by new health", err)
	}
}

// TestNewsGenericCounts rejects unsupported result counts without tightening legacy validation.
//
// Version:
//   - 2026-10-31: Added.
func TestNewsGenericCounts(t *testing.T) {
	p := Params{AssetType: "crypto", Asset: "sui", From: 1}
	r := Result{AssetType: "crypto", Asset: "sui", Network: "mainnet", Coverage: "latest_source_windows", Snapshot: "opaque", Page: 2, Limit: 1, Total: 1, SentAt: 10, Items: []Item{}, Sources: []Source{{ID: "sui-status", Name: "Sui Status", URL: "https://status.sui.io/", Status: "never_collected", Classification: &Classification{Profile: "sui-mainnet-v1", Status: "never_classified"}}}}
	p.Page = 2
	p.Limit = 1
	p.Snapshot = "opaque"
	if err := r.ValidateFor(p); err == nil {
		t.Fatal("total exceeded pinned Mainnet counts")
	}
	r.Sources[0].Classification = nil
	if err := r.ValidateFor(p); err != nil {
		t.Fatal("legacy compatibility changed", err)
	}
}
