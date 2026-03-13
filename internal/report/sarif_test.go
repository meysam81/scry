package report

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/meysam81/scry/internal/model"
)

func TestSARIFReporterName(t *testing.T) {
	r := &SARIFReporter{}
	if r.Name() != "sarif" {
		t.Errorf("Name() = %q, want %q", r.Name(), "sarif")
	}
}

func TestSARIFReporterNilResult(t *testing.T) {
	r := &SARIFReporter{}
	var buf bytes.Buffer

	if err := r.Write(context.Background(), nil, &buf); err != nil {
		t.Fatalf("Write(nil) error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for nil result, got %d bytes", buf.Len())
	}
}

func TestSARIFReporterValidJSON(t *testing.T) {
	r := &SARIFReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{
				CheckName: "seo/missing-title",
				Severity:  model.SeverityCritical,
				Message:   "page is missing a <title> tag",
				URL:       "https://example.com/page",
			},
			{
				CheckName: "seo/title-length",
				Severity:  model.SeverityWarning,
				Message:   "Title too short",
				URL:       "https://example.com/about",
			},
			{
				CheckName: "perf/large-image",
				Severity:  model.SeverityInfo,
				Message:   "Image exceeds 500KB",
				URL:       "https://example.com/gallery",
				Detail:    "hero.png: 1.2MB",
			},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  5 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	if !json.Valid(buf.Bytes()) {
		t.Fatal("output is not valid JSON")
	}
}

func TestSARIFReporterSchemaAndVersion(t *testing.T) {
	r := &SARIFReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL:   "https://example.com",
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc sarifDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	wantSchema := "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/main/sarif-2.1/schema/sarif-schema-2.1.0.json"
	if doc.Schema != wantSchema {
		t.Errorf("$schema = %q, want %q", doc.Schema, wantSchema)
	}
	if doc.Version != "2.1.0" {
		t.Errorf("version = %q, want %q", doc.Version, "2.1.0")
	}
}

func TestSARIFReporterToolDriver(t *testing.T) {
	r := &SARIFReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL:   "https://example.com",
		Issues:    []model.Issue{{CheckName: "a", Severity: model.SeverityInfo, Message: "m", URL: "u"}},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc sarifDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if len(doc.Runs) != 1 {
		t.Fatalf("runs count = %d, want 1", len(doc.Runs))
	}

	driver := doc.Runs[0].Tool.Driver
	if driver.Name != "scry" {
		t.Errorf("driver.name = %q, want %q", driver.Name, "scry")
	}
	if driver.Version != "1.0.0" {
		t.Errorf("driver.version = %q, want %q", driver.Version, "1.0.0")
	}
	if driver.InformationURI != "https://github.com/meysam81/scry" {
		t.Errorf("driver.informationUri = %q, want %q", driver.InformationURI, "https://github.com/meysam81/scry")
	}
}

func TestSARIFReporterSeverityMapping(t *testing.T) {
	r := &SARIFReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{CheckName: "check-crit", Severity: model.SeverityCritical, Message: "crit msg", URL: "https://a.com"},
			{CheckName: "check-warn", Severity: model.SeverityWarning, Message: "warn msg", URL: "https://b.com"},
			{CheckName: "check-info", Severity: model.SeverityInfo, Message: "info msg", URL: "https://c.com"},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc sarifDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	run := doc.Runs[0]

	// Verify rules.
	wantRuleLevels := map[string]string{
		"check-crit": "error",
		"check-warn": "warning",
		"check-info": "note",
	}
	if len(run.Tool.Driver.Rules) != 3 {
		t.Fatalf("rules count = %d, want 3", len(run.Tool.Driver.Rules))
	}
	for _, rule := range run.Tool.Driver.Rules {
		want, ok := wantRuleLevels[rule.ID]
		if !ok {
			t.Errorf("unexpected rule id %q", rule.ID)
			continue
		}
		if rule.DefaultConfiguration.Level != want {
			t.Errorf("rule %q level = %q, want %q", rule.ID, rule.DefaultConfiguration.Level, want)
		}
	}

	// Verify results.
	wantResultLevels := map[string]string{
		"check-crit": "error",
		"check-warn": "warning",
		"check-info": "note",
	}
	if len(run.Results) != 3 {
		t.Fatalf("results count = %d, want 3", len(run.Results))
	}
	for _, res := range run.Results {
		want, ok := wantResultLevels[res.RuleID]
		if !ok {
			t.Errorf("unexpected result ruleId %q", res.RuleID)
			continue
		}
		if res.Level != want {
			t.Errorf("result %q level = %q, want %q", res.RuleID, res.Level, want)
		}
	}
}

func TestSARIFReporterRuleDeduplication(t *testing.T) {
	r := &SARIFReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{CheckName: "seo/missing-title", Severity: model.SeverityCritical, Message: "msg1", URL: "https://a.com"},
			{CheckName: "seo/missing-title", Severity: model.SeverityCritical, Message: "msg2", URL: "https://b.com"},
			{CheckName: "seo/title-length", Severity: model.SeverityWarning, Message: "msg3", URL: "https://c.com"},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc sarifDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	rules := doc.Runs[0].Tool.Driver.Rules
	if len(rules) != 2 {
		t.Fatalf("rules count = %d, want 2 (deduplicated)", len(rules))
	}

	// But results should still be 3 (no dedup on results).
	results := doc.Runs[0].Results
	if len(results) != 3 {
		t.Fatalf("results count = %d, want 3", len(results))
	}
}

func TestSARIFReporterLocations(t *testing.T) {
	r := &SARIFReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{
				CheckName: "seo/missing-title",
				Severity:  model.SeverityCritical,
				Message:   "page is missing a title tag",
				URL:       "https://example.com/page",
			},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc sarifDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	res := doc.Runs[0].Results[0]
	if len(res.Locations) != 1 {
		t.Fatalf("locations count = %d, want 1", len(res.Locations))
	}

	uri := res.Locations[0].PhysicalLocation.ArtifactLocation.URI
	if uri != "https://example.com/page" {
		t.Errorf("location URI = %q, want %q", uri, "https://example.com/page")
	}
}

func TestSARIFReporterDetailAppendedToMessage(t *testing.T) {
	r := &SARIFReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{
				CheckName: "perf/large-image",
				Severity:  model.SeverityInfo,
				Message:   "Image exceeds 500KB",
				URL:       "https://example.com/gallery",
				Detail:    "hero.png: 1.2MB",
			},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc sarifDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	msg := doc.Runs[0].Results[0].Message.Text
	want := "Image exceeds 500KB: hero.png: 1.2MB"
	if msg != want {
		t.Errorf("message = %q, want %q", msg, want)
	}
}

func TestSARIFReporterEmptyIssues(t *testing.T) {
	r := &SARIFReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL:   "https://example.com",
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc sarifDocument
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if len(doc.Runs) != 1 {
		t.Fatalf("runs count = %d, want 1", len(doc.Runs))
	}
	if len(doc.Runs[0].Tool.Driver.Rules) != 0 {
		t.Errorf("rules count = %d, want 0", len(doc.Runs[0].Tool.Driver.Rules))
	}
	if len(doc.Runs[0].Results) != 0 {
		t.Errorf("results count = %d, want 0", len(doc.Runs[0].Results))
	}
}
