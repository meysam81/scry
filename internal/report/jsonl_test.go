package report

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/meysam81/scry/core/model"
)

func TestJSONLReporterName(t *testing.T) {
	r := &JSONLReporter{}
	if r.Name() != "jsonl" {
		t.Errorf("Name() = %q, want %q", r.Name(), "jsonl")
	}
}

func TestJSONLReporterNilResult(t *testing.T) {
	r := &JSONLReporter{}
	var buf bytes.Buffer

	if err := r.Write(context.Background(), nil, &buf); err != nil {
		t.Fatalf("Write(nil) error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for nil result, got %d bytes", buf.Len())
	}
}

func TestJSONLReporterOneLinePerIssue(t *testing.T) {
	r := &JSONLReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Pages: []*model.Page{
			{URL: "https://example.com", StatusCode: 200},
			{URL: "https://example.com/about", StatusCode: 200},
		},
		Issues: []model.Issue{
			{CheckName: "seo/missing-title", Severity: model.SeverityCritical, Message: "missing title", URL: "https://example.com/page"},
			{CheckName: "seo/title-length", Severity: model.SeverityWarning, Message: "title too short", URL: "https://example.com/about"},
			{CheckName: "perf/large-image", Severity: model.SeverityInfo, Message: "image too big", URL: "https://example.com/gallery", Detail: "hero.png: 1.2MB"},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  5 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	// 3 issue lines + 1 summary line = 4.
	if len(lines) != 4 {
		t.Fatalf("line count = %d, want 4", len(lines))
	}

	// Each line must be valid JSON.
	for i, line := range lines {
		if !json.Valid([]byte(line)) {
			t.Errorf("line %d is not valid JSON: %q", i, line)
		}
	}
}

func TestJSONLReporterIssueLineFields(t *testing.T) {
	r := &JSONLReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{
				CheckName: "seo/missing-title",
				Severity:  model.SeverityCritical,
				Message:   "page is missing a <title> tag",
				URL:       "https://example.com/page",
				Detail:    "no title element",
			},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  5 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) < 1 {
		t.Fatal("expected at least 1 line")
	}

	var issue jsonlIssueLine
	if err := json.Unmarshal([]byte(lines[0]), &issue); err != nil {
		t.Fatalf("Unmarshal issue line: %v", err)
	}

	if issue.CheckName != "seo/missing-title" {
		t.Errorf("check_name = %q, want %q", issue.CheckName, "seo/missing-title")
	}
	if issue.Severity != model.SeverityCritical {
		t.Errorf("severity = %q, want %q", issue.Severity, model.SeverityCritical)
	}
	if issue.Message != "page is missing a <title> tag" {
		t.Errorf("message = %q, want %q", issue.Message, "page is missing a <title> tag")
	}
	if issue.URL != "https://example.com/page" {
		t.Errorf("url = %q, want %q", issue.URL, "https://example.com/page")
	}
	if issue.Detail != "no title element" {
		t.Errorf("detail = %q, want %q", issue.Detail, "no title element")
	}
}

func TestJSONLReporterSummaryLine(t *testing.T) {
	r := &JSONLReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Pages: []*model.Page{
			{URL: "https://example.com", StatusCode: 200},
			{URL: "https://example.com/about", StatusCode: 200},
			{URL: "https://example.com/contact", StatusCode: 200},
		},
		Issues: []model.Issue{
			{CheckName: "a", Severity: model.SeverityCritical, Message: "m", URL: "u"},
			{CheckName: "b", Severity: model.SeverityCritical, Message: "m", URL: "u"},
			{CheckName: "c", Severity: model.SeverityWarning, Message: "m", URL: "u"},
			{CheckName: "d", Severity: model.SeverityInfo, Message: "m", URL: "u"},
			{CheckName: "e", Severity: model.SeverityInfo, Message: "m", URL: "u"},
			{CheckName: "f", Severity: model.SeverityInfo, Message: "m", URL: "u"},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  7500 * time.Millisecond,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	// 6 issue lines + 1 summary = 7.
	if len(lines) != 7 {
		t.Fatalf("line count = %d, want 7", len(lines))
	}

	// The last line is the summary.
	var summary jsonlSummaryLine
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &summary); err != nil {
		t.Fatalf("Unmarshal summary line: %v", err)
	}

	if summary.Type != "summary" {
		t.Errorf("type = %q, want %q", summary.Type, "summary")
	}
	if summary.SeedURL != "https://example.com" {
		t.Errorf("seed_url = %q, want %q", summary.SeedURL, "https://example.com")
	}
	if summary.Pages != 3 {
		t.Errorf("pages = %d, want 3", summary.Pages)
	}
	if summary.Issues != 6 {
		t.Errorf("issues = %d, want 6", summary.Issues)
	}
	if summary.Critical != 2 {
		t.Errorf("critical = %d, want 2", summary.Critical)
	}
	if summary.Warning != 1 {
		t.Errorf("warning = %d, want 1", summary.Warning)
	}
	if summary.Info != 3 {
		t.Errorf("info = %d, want 3", summary.Info)
	}
	if summary.Duration != "7.5s" {
		t.Errorf("duration = %q, want %q", summary.Duration, "7.5s")
	}
}

func TestJSONLReporterEmptyIssues(t *testing.T) {
	r := &JSONLReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Pages: []*model.Page{
			{URL: "https://example.com", StatusCode: 200},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  1 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	// Only 1 summary line (no issues).
	if len(lines) != 1 {
		t.Fatalf("line count = %d, want 1 (summary only)", len(lines))
	}

	var summary jsonlSummaryLine
	if err := json.Unmarshal([]byte(lines[0]), &summary); err != nil {
		t.Fatalf("Unmarshal summary: %v", err)
	}

	if summary.Type != "summary" {
		t.Errorf("type = %q, want %q", summary.Type, "summary")
	}
	if summary.Issues != 0 {
		t.Errorf("issues = %d, want 0", summary.Issues)
	}
	if summary.Pages != 1 {
		t.Errorf("pages = %d, want 1", summary.Pages)
	}
}

func TestJSONLReporterDetailOmittedWhenEmpty(t *testing.T) {
	r := &JSONLReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{CheckName: "seo/missing-title", Severity: model.SeverityCritical, Message: "msg", URL: "https://a.com"},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) < 1 {
		t.Fatal("expected at least 1 line")
	}

	// The first line (issue) should not contain a "detail" key.
	if strings.Contains(lines[0], `"detail"`) {
		t.Errorf("issue line should omit detail when empty, got: %s", lines[0])
	}
}

func TestJSONLReporterNoTypeFieldInIssueLines(t *testing.T) {
	r := &JSONLReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{CheckName: "a", Severity: model.SeverityInfo, Message: "m", URL: "u"},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	// First line is issue, should NOT have "type" field.
	if strings.Contains(lines[0], `"type"`) {
		t.Errorf("issue line should not contain type field, got: %s", lines[0])
	}
	// Last line is summary, should have "type" field.
	if !strings.Contains(lines[len(lines)-1], `"type":"summary"`) {
		t.Errorf("summary line should contain type field, got: %s", lines[len(lines)-1])
	}
}
