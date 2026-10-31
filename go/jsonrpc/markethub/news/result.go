package news

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type Item struct {
	ArticleID       string `json:"articleId,omitempty"`
	RecordSHA256    string `json:"recordSha256,omitempty"`
	AnnotationSetID string `json:"annotationSetId,omitempty"`
	ID              string `json:"id"`
	Revision        string `json:"revision"`
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
	Network         string `json:"network"`
	NetworkEvidence string `json:"networkEvidence"`
	Status          string `json:"status,omitempty"`
}

// Input identifies the selected immutable collection/classification pair.
// SourceURL is the captured public endpoint; Source.URL is the current human homepage.
type Input struct {
	AnnotationSetID string `json:"annotationSetId"`
	DocumentSHA256  string `json:"documentSha256"`
	Profile         string `json:"profile"`
	ParserVersion   string `json:"parserVersion"`
	ProducerVersion string `json:"producerVersion"`
	PublisherID     string `json:"publisherId"`
	SourceName      string `json:"sourceName"`
	SourceURL       string `json:"sourceUrl"`
}

// Classification describes current health for the latest completed collection.
// Stale compares the selected pair with current completed collection/classification attempts.
type Classification struct {
	Profile       string `json:"profile"`
	Status        string `json:"status"` // never_classified, pending, running, failed, complete.
	LastAttemptAt int64  `json:"lastAttemptAt,omitempty"`
	LastSuccessAt int64  `json:"lastSuccessAt,omitempty"`
	Stale         bool   `json:"stale"`
}

type Source struct {
	Input          *Input          `json:"input,omitempty"`
	Classification *Classification `json:"classification,omitempty"`
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	URL            string          `json:"url"`
	Status         string          `json:"status"` // complete, stale, failed, running, never_collected.
	LastAttemptAt  int64           `json:"lastAttemptAt,omitempty"`
	LastSuccessAt  int64           `json:"lastSuccessAt,omitempty"` // Current collector health, independent of pinned data.
	RunID          string          `json:"runId,omitempty"`         // Immutable data run selected in this response.
	ObservedAt     int64           `json:"observedAt,omitempty"`
	ItemCount      int             `json:"itemCount"`
	MainnetCount   int             `json:"mainnetCount"`
	ResponseSHA256 string          `json:"responseSha256,omitempty"`
}

type Result struct {
	AssetType market.AssetClass `json:"assetType"`
	Asset     string            `json:"asset"`
	Network   string            `json:"network"`
	Coverage  string            `json:"coverage"` // latest_source_windows; not an exhaustive news archive.
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

// ValidateFor checks scope, provenance and page consistency for the originating request.
//
// Version:
//   - 2026-10-31: Added; validate optional generic input and independent classification health.
func (r Result) ValidateFor(p Params) error {
	if err := p.Validate(); err != nil {
		return fmt.Errorf("failed to validate news result: %w", err)
	}
	p = p.Normalize()
	if r.AssetType != p.AssetType || r.Asset != p.Asset || r.Network != "mainnet" || r.Coverage != "latest_source_windows" || r.Page != p.Page || r.Limit != p.Limit || r.SentAt <= 0 || len(r.Snapshot) == 0 || len(r.Snapshot) > 8192 || (p.Snapshot != "" && p.Snapshot != r.Snapshot) {
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
	generic := false
	mainnetTotal := 0
	for _, source := range r.Sources {
		if source.ID == "" || source.Name == "" || !safeURL(source.URL) {
			return invalid("source")
		}
		if _, exists := sources[source.ID]; exists {
			return invalid("source_duplicate")
		}
		switch source.Status {
		case "complete", "stale", "failed", "running", "never_collected":
		default:
			return invalid("source_status")
		}
		if source.ItemCount < 0 || source.ItemCount > 1000 || source.MainnetCount < 0 || source.MainnetCount > source.ItemCount || source.LastAttemptAt < 0 || source.LastSuccessAt < 0 {
			return invalid("source_counts")
		}
		if source.RunID != "" {
			if !runID.MatchString(source.RunID) || source.ObservedAt <= 0 || !hash.MatchString(source.ResponseSHA256) {
				return invalid("source_provenance")
			}
		} else if source.ObservedAt != 0 || source.ItemCount != 0 || source.MainnetCount != 0 || source.ResponseSHA256 != "" {
			return invalid("source_empty")
		}
		if source.Input != nil {
			pin := source.Input
			if source.RunID == "" || !runID.MatchString(pin.AnnotationSetID) || !hash.MatchString(pin.DocumentSHA256) || !safeToken(pin.Profile) || !safeToken(pin.ParserVersion) || !safeToken(pin.ProducerVersion) || !safeToken(pin.PublisherID) || pin.SourceName == "" || len(pin.SourceName) > 256 || !utf8.ValidString(pin.SourceName) || strings.IndexFunc(pin.SourceName, unicode.IsControl) >= 0 || !safeURL(pin.SourceURL) || source.Classification == nil || source.Classification.Profile != pin.Profile {
				return invalid("source_input")
			}
		}
		if source.Classification != nil {
			generic = true
			health := source.Classification
			if !safeToken(health.Profile) || health.LastAttemptAt < 0 || health.LastSuccessAt < 0 {
				return invalid("classification")
			}
			switch health.Status {
			case "never_classified", "pending", "running", "failed", "complete":
			default:
				return invalid("classification_status")
			}
			if (health.Status == "never_classified" || health.Status == "pending") && health.LastAttemptAt != 0 {
				return invalid("classification_attempt")
			}
			if (health.Status == "running" || health.Status == "failed" || health.Status == "complete") && health.LastAttemptAt == 0 {
				return invalid("classification_attempt")
			}
			if health.Status == "never_classified" && health.LastSuccessAt != 0 || health.Status == "complete" && health.LastSuccessAt == 0 {
				return invalid("classification_success")
			}
			if source.Input == nil && (source.RunID != "" || health.Stale) {
				return invalid("classification_input")
			}
		}
		mainnetTotal += source.MainnetCount
		sources[source.ID] = source
	}
	if generic && (r.Total > mainnetTotal || p.From == 0 && p.To == 0 && r.Total != mainnetTotal) {
		return invalid("source_totals")
	}
	seen := make(map[string]bool)
	pageBySource := make(map[string]int)
	var previous *Item
	for idx := range r.Items {
		i := &r.Items[idx]
		source, ok := sources[i.SourceID]
		if !ok || !hash.MatchString(i.ID) || !hash.MatchString(i.Revision) || !runID.MatchString(i.RunID) || i.RunID != source.RunID || i.ObservedAt <= 0 || i.ObservedAt != source.ObservedAt || i.Network != "mainnet" || i.NetworkEvidence == "" || !safeURL(i.URL) || i.PublishedAt <= 0 || i.UpdatedAt < 0 || i.OccurredAt < 0 {
			return invalid("item")
		}
		if source.Input != nil {
			if !hash.MatchString(i.ArticleID) || !hash.MatchString(i.RecordSHA256) || i.AnnotationSetID != source.Input.AnnotationSetID {
				return invalid("item_input")
			}
		} else if i.ArticleID != "" || i.RecordSHA256 != "" || i.AnnotationSetID != "" {
			return invalid("item_legacy_input")
		}
		if i.Kind != "announcement" && i.Kind != "incident" {
			return invalid("kind")
		}
		if i.Title == "" || len([]rune(i.Title)) > 500 || len([]rune(i.Summary)) > 500 || len([]rune(i.NetworkEvidence)) > 1000 || strings.IndexFunc(i.Title+i.Summary+i.NetworkEvidence, unicode.IsControl) >= 0 {
			return invalid("text")
		}
		if (p.From > 0 && i.PublishedAt < p.From) || (p.To > 0 && i.PublishedAt >= p.To) || seen[i.ID] {
			return invalid("item_scope")
		}
		if previous != nil && (i.PublishedAt > previous.PublishedAt || (i.PublishedAt == previous.PublishedAt && i.ID <= previous.ID)) {
			return invalid("order")
		}
		seen[i.ID] = true
		pageBySource[i.SourceID]++
		if generic && pageBySource[i.SourceID] > source.MainnetCount {
			return invalid("source_page_count")
		}
		previous = i
	}
	return nil
}

func safeURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && len(raw) <= 2048 && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && u.Fragment == ""
}

func safeToken(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_.:/-", r)) {
			return false
		}
	}
	return true
}
