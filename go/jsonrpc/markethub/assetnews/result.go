package assetnews

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

const Scope = "asset_context"

type Item struct {
	ArticleID       string `json:"articleId"`
	RecordSHA256    string `json:"recordSha256"`
	AnnotationSetID string `json:"annotationSetId"`
	RunID           string `json:"runId"`
	SourceID        string `json:"sourceId"`
	URL             string `json:"url"`
	Title           string `json:"title"`
	Summary         string `json:"summary"`
	Kind            string `json:"kind"`
	PublishedAt     int64  `json:"publishedAt"`
	UpdatedAt       int64  `json:"updatedAt,omitempty"`
	OccurredAt      int64  `json:"occurredAt,omitempty"`
	ObservedAt      int64  `json:"observedAt"`
	NetworkScope    string `json:"networkScope"`
	ClaimStatus     string `json:"claimStatus"`
	NetworkEvidence string `json:"networkEvidence"`
	Status          string `json:"status,omitempty"`
}

type Input struct {
	AnnotationSetID string `json:"annotationSetId"`
	DocumentSHA256  string `json:"documentSha256"`
	Profile         string `json:"profile"`
	ParserVersion   string `json:"parserVersion"`
	ProducerVersion string `json:"producerVersion"`
	PublisherID     string `json:"publisherId"`
	SourceName      string `json:"sourceName"`
	SourceURL       string `json:"sourceUrl"`
	ClassifiedAt    int64  `json:"classifiedAt"`
}

type Classification struct {
	Profile       string `json:"profile"`
	Status        string `json:"status"`
	LastAttemptAt int64  `json:"lastAttemptAt,omitempty"`
	LastSuccessAt int64  `json:"lastSuccessAt,omitempty"`
	Stale         bool   `json:"stale"`
}

type Source struct {
	Input            *Input          `json:"input,omitempty"`
	Classification   *Classification `json:"classification,omitempty"`
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	URL              string          `json:"url"`
	Status           string          `json:"status"`
	LastAttemptAt    int64           `json:"lastAttemptAt,omitempty"`
	LastSuccessAt    int64           `json:"lastSuccessAt,omitempty"`
	RunID            string          `json:"runId,omitempty"`
	ObservedAt       int64           `json:"observedAt,omitempty"`
	ItemCount        int             `json:"itemCount"`
	EligibleCount    int             `json:"eligibleCount"`
	AssociationBasis string          `json:"associationBasis"`
	ResponseSHA256   string          `json:"responseSha256,omitempty"`
}

type Result struct {
	AssetType market.AssetClass `json:"assetType"`
	Asset     string            `json:"asset"`
	Scope     string            `json:"scope"`
	Coverage  string            `json:"coverage"`
	Snapshot  string            `json:"snapshot"`
	Page      int               `json:"page"`
	Limit     int               `json:"limit"`
	Total     int               `json:"total"`
	HasMore   bool              `json:"hasMore"`
	Items     []Item            `json:"items"`
	Sources   []Source          `json:"sources"`
	SentAt    int64             `json:"sentAt"`
}

var hash = regexp.MustCompile(`^[0-9a-f]{64}$`)
var runID = regexp.MustCompile(`^[0-9a-f]{32}$`)

// UnmarshalJSON rejects crossed legacy fields and unknown context metadata shapes.
//
// Version:
//   - 2026-11-02: Added.
func (r *Result) UnmarshalJSON(data []byte) error {
	if r == nil {
		return invalid("destination")
	}
	type wire Result
	var value wire
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&value); err != nil {
		return fmt.Errorf("failed to decode asset news result: %w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return invalid("result")
	}
	*r = Result(value)
	return nil
}

// ValidateFor checks official context scope, immutable inputs and pagination.
// Network and claim labels describe source context rather than verified operation.
//
// Version:
//   - 2026-11-02: Added.
func (r Result) ValidateFor(p Params) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("failed to validate asset news result: %w", err)
	}
	p = p.Normalize()
	if r.AssetType != p.AssetType || r.Asset != p.Asset || r.Scope != Scope || r.Coverage != "latest_source_windows" || r.Page != p.Page || r.Limit != p.Limit || r.SentAt <= 0 || len(r.Snapshot) == 0 || len(r.Snapshot) > 8192 || (p.Snapshot != "" && p.Snapshot != r.Snapshot) {
		return invalid("result_scope")
	}
	if r.Items == nil || r.Sources == nil || len(r.Sources) > 16 || r.Total < 0 || r.Total > 16000 {
		return invalid("result")
	}
	remaining := max(0, r.Total-(p.Page-1)*p.Limit)
	if len(r.Items) != min(p.Limit, remaining) || r.HasMore != (remaining > p.Limit) {
		return invalid("page")
	}
	sources := make(map[string]Source)
	eligible := 0
	for _, s := range r.Sources {
		if s.ID == "" || s.Name == "" || !safeURL(s.URL) || s.AssociationBasis != "registered_official_source" {
			return invalid("source")
		}
		if _, exists := sources[s.ID]; exists {
			return invalid("source_duplicate")
		}
		switch s.Status {
		case "complete", "stale", "failed", "running", "never_collected":
		default:
			return invalid("source_status")
		}
		if s.ItemCount < 0 || s.ItemCount > 1000 || s.EligibleCount < 0 || s.EligibleCount > s.ItemCount || s.LastAttemptAt < 0 || s.LastSuccessAt < 0 {
			return invalid("source_counts")
		}
		if s.Classification == nil {
			return invalid("classification")
		}
		h := s.Classification
		if !safeToken(h.Profile) || h.LastAttemptAt < 0 || h.LastSuccessAt < 0 {
			return invalid("classification")
		}
		switch h.Status {
		case "never_classified", "pending", "running", "failed", "complete":
		default:
			return invalid("classification_status")
		}
		attempted := h.Status == "running" || h.Status == "failed" || h.Status == "complete"
		if attempted != (h.LastAttemptAt > 0) || h.Status == "never_classified" && h.LastSuccessAt != 0 || h.Status == "complete" && h.LastSuccessAt == 0 {
			return invalid("classification_time")
		}
		if s.RunID == "" {
			if s.Input != nil || s.ObservedAt != 0 || s.ItemCount != 0 || s.EligibleCount != 0 || s.ResponseSHA256 != "" || h.Stale {
				return invalid("source_empty")
			}
		} else {
			v := s.Input
			if !runID.MatchString(s.RunID) || s.ObservedAt <= 0 || !hash.MatchString(s.ResponseSHA256) || v == nil {
				return invalid("source_provenance")
			}
			if !runID.MatchString(v.AnnotationSetID) || !hash.MatchString(v.DocumentSHA256) || !safeToken(v.Profile) || !safeToken(v.ParserVersion) || !safeToken(v.ProducerVersion) || !safeToken(v.PublisherID) || !safeURL(v.SourceURL) || v.Profile != h.Profile || v.ClassifiedAt < s.ObservedAt || v.ClassifiedAt > r.SentAt || v.SourceName == "" || len(v.SourceName) > 256 || !utf8.ValidString(v.SourceName) || strings.IndexFunc(v.SourceName, unicode.IsControl) >= 0 {
				return invalid("source_input")
			}
		}
		eligible += s.EligibleCount
		sources[s.ID] = s
	}
	if r.Total > eligible || p.From == 0 && p.To == 0 && r.Total != eligible {
		return invalid("source_totals")
	}
	seen, pageCounts := make(map[string]bool), make(map[string]int)
	var previous *Item
	for n := range r.Items {
		i := &r.Items[n]
		s, ok := sources[i.SourceID]
		if !ok || !hash.MatchString(i.ArticleID) || !hash.MatchString(i.RecordSHA256) || !runID.MatchString(i.AnnotationSetID) || !runID.MatchString(i.RunID) || s.Input == nil || i.RunID != s.RunID || i.AnnotationSetID != s.Input.AnnotationSetID || i.ObservedAt != s.ObservedAt || !safeURL(i.URL) || i.PublishedAt <= 0 || i.UpdatedAt < 0 || i.OccurredAt < 0 {
			return invalid("item_provenance")
		}
		switch i.NetworkScope {
		case "mainnet", "non_mainnet", "mixed", "not_applicable", "unspecified":
		default:
			return invalid("network_scope")
		}
		switch i.ClaimStatus {
		case "unverified", "planned", "reported_live", "retrospective", "mixed":
		default:
			return invalid("claim_status")
		}
		if i.Kind != "announcement" && i.Kind != "incident" {
			return invalid("kind")
		}
		if i.Title == "" || len([]rune(i.Title)) > 500 || len([]rune(i.Summary)) > 500 || i.NetworkEvidence == "" || len([]rune(i.NetworkEvidence)) > 1000 || strings.IndexFunc(i.Title+i.Summary+i.NetworkEvidence, unicode.IsControl) >= 0 {
			return invalid("text")
		}
		if seen[i.ArticleID] || p.From > 0 && i.PublishedAt < p.From || p.To > 0 && i.PublishedAt >= p.To {
			return invalid("item_scope")
		}
		if previous != nil && (i.PublishedAt > previous.PublishedAt || i.PublishedAt == previous.PublishedAt && i.ArticleID <= previous.ArticleID) {
			return invalid("order")
		}
		pageCounts[i.SourceID]++
		if pageCounts[i.SourceID] > s.EligibleCount {
			return invalid("source_page_count")
		}
		seen[i.ArticleID] = true
		previous = i
	}
	return nil
}

func safeURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && len(raw) <= 2048 && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.Fragment == ""
}

func safeToken(v string) bool {
	if len(v) == 0 || len(v) > 64 {
		return false
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("_.:/-", c)) {
			return false
		}
	}
	return true
}
