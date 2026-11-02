package assetnews

import (
	"encoding/json"
	"strings"
	"testing"
)

func contextFixture() (Params, Result) {
	p := Params{AssetType: "crypto", Asset: "sui"}
	s := Source{ID: "sui-blog", Name: "Sui Blog", URL: "https://www.sui.io/blog", Status: "complete", ItemCount: 1, EligibleCount: 1, AssociationBasis: "registered_official_source", RunID: strings.Repeat("a", 32), ObservedAt: 5, ResponseSHA256: strings.Repeat("b", 64), Input: &Input{AnnotationSetID: strings.Repeat("c", 32), DocumentSHA256: strings.Repeat("d", 64), Profile: "official-asset-context-v1", ParserVersion: "sui-context-v1", ProducerVersion: "official-asset-context-v1", PublisherID: "sui", SourceName: "Sui Blog", SourceURL: "https://www.sui.io/blog/rss.xml", ClassifiedAt: 7}, Classification: &Classification{Profile: "official-asset-context-v1", Status: "complete", LastAttemptAt: 6, LastSuccessAt: 7}}
	i := Item{ArticleID: strings.Repeat("1", 64), RecordSHA256: strings.Repeat("2", 64), AnnotationSetID: s.Input.AnnotationSetID, RunID: s.RunID, SourceID: s.ID, URL: "https://www.sui.io/blog/fixture", Title: "Planned launch", Summary: "Source describes a future plan", Kind: "announcement", PublishedAt: 1, ObservedAt: 5, NetworkScope: "unspecified", ClaimStatus: "planned", NetworkEvidence: "source metadata; network remains unverified"}
	return p, Result{AssetType: "crypto", Asset: "sui", Scope: Scope, Coverage: "latest_source_windows", Snapshot: "fixture", Page: 1, Limit: 20, Total: 1, Items: []Item{i}, Sources: []Source{s}, SentAt: 10}
}

// TestContextSelectors preserves strict selectors and rejects Mainnet or owner overrides.
//
// Version:
//   - 2026-11-02: Added.
func TestContextSelectors(t *testing.T) {
	for _, raw := range []string{`null`, `{}`, `{"assetType":"crypto","asset":"sui","scope":"mainnet"}`, `{"assetType":"crypto","asset":"sui","network":"mainnet"}`, `{"assetType":"crypto","asset":"sui","accountId":"1"}`, `{"assetType":"crypto","asset":"sui","page":2}`, `{"assetType":"crypto","asset":"sui","asset":"eth"}`, `{"assetType":"crypto","asset":"sui","limit":null}`} {
		var p Params
		if err := json.Unmarshal([]byte(raw), &p); err == nil {
			t.Fatalf("accepted invalid selector: %s", raw)
		}
	}
	var p Params
	if err := json.Unmarshal([]byte(`{"assetType":"crypto","asset":"sui"}`), &p); err != nil || p.Page != 1 || p.Limit != 20 {
		t.Fatal(p, err)
	}
}

// TestContextContract rejects mismatched provenance, scope and unsupported dates.
//
// Version:
//   - 2026-11-02: Added.
func TestContextContract(t *testing.T) {
	p, r := contextFixture()
	if err := r.ValidateFor(p); err != nil {
		t.Fatal(err)
	}
	changes := map[string]func(*Result){
		"scope":       func(r *Result) { r.Scope = "mainnet" },
		"undated":     func(r *Result) { r.Items[0].PublishedAt = 0 },
		"annotation":  func(r *Result) { r.Items[0].AnnotationSetID = strings.Repeat("e", 32) },
		"network":     func(r *Result) { r.Items[0].NetworkScope = "unknown-is-mainnet" },
		"claim":       func(r *Result) { r.Items[0].ClaimStatus = "operational" },
		"time":        func(r *Result) { r.Sources[0].Input.ClassifiedAt = 4 },
		"count":       func(r *Result) { r.Sources[0].EligibleCount = 0 },
		"association": func(r *Result) { r.Sources[0].AssociationBasis = "official-is-mainnet" },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			p, r := contextFixture()
			change(&r)
			if r.ValidateFor(p) == nil {
				t.Fatal("accepted invalid context")
			}
		})
	}
	for _, scope := range []string{"non_mainnet", "mixed", "not_applicable", "unspecified"} {
		_, r := contextFixture()
		r.Items[0].NetworkScope = scope
		if err := r.ValidateFor(p); err != nil {
			t.Fatal(scope, err)
		}
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), `"scope":"asset_context"`, `"scope":"asset_context","network":"mainnet"`, 1))
	var decoded Result
	if json.Unmarshal(raw, &decoded) == nil {
		t.Fatal("accepted crossed legacy shape")
	}
}
