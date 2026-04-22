package report

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/meysam81/scry/core/model"
)

func TestMarkdownReporterContainsHeader(t *testing.T) {
	r := &MarkdownReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Pages: []*model.Page{
			{URL: "https://example.com", StatusCode: 200},
			{URL: "https://example.com/about", StatusCode: 200},
		},
		Issues: []model.Issue{
			{
				CheckName: "health/4xx",
				Severity:  model.SeverityCritical,
				Message:   "Page returned status 404",
				URL:       "https://example.com/broken",
			},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  12400 * time.Millisecond,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "# Site Audit Report") {
		t.Error("output missing '# Site Audit Report' header")
	}
}

func TestMarkdownReporterContainsSeedURLAndPageCount(t *testing.T) {
	r := &MarkdownReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://mysite.example.org",
		Pages: []*model.Page{
			{URL: "https://mysite.example.org"},
			{URL: "https://mysite.example.org/a"},
			{URL: "https://mysite.example.org/b"},
		},
		CrawledAt: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		Duration:  3 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "https://mysite.example.org") {
		t.Error("output missing seed URL")
	}
	if !strings.Contains(out, "**Pages:** 3") {
		t.Error("output missing page count")
	}
}

func TestMarkdownReporterSeveritySections(t *testing.T) {
	r := &MarkdownReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{CheckName: "a", Severity: model.SeverityCritical, Message: "crit msg", URL: "https://example.com/a"},
			{CheckName: "b", Severity: model.SeverityWarning, Message: "warn msg", URL: "https://example.com/b"},
			{CheckName: "c", Severity: model.SeverityInfo, Message: "info msg", URL: "https://example.com/c"},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  1 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	out := buf.String()

	for _, heading := range []string{"## Critical Issues", "## Warning Issues", "## Info Issues"} {
		if !strings.Contains(out, heading) {
			t.Errorf("output missing heading %q", heading)
		}
	}
}

func TestMarkdownReporterWithLighthouse(t *testing.T) {
	r := &MarkdownReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Lighthouse: []model.LighthouseResult{
			{
				URL:                "https://example.com",
				PerformanceScore:   0.85,
				AccessibilityScore: 0.92,
				BestPracticesScore: 0.88,
				SEOScore:           0.90,
			},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  1 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "## Lighthouse Scores") {
		t.Error("output missing Lighthouse Scores section")
	}

	// Scores should be converted to integers: 85, 92, 88, 90.
	for _, score := range []string{"85", "92", "88", "90"} {
		if !strings.Contains(out, score) {
			t.Errorf("output missing lighthouse score %q", score)
		}
	}
}

func TestMarkdownReporterNoIssues(t *testing.T) {
	r := &MarkdownReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL:   "https://example.com",
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  1 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	out := buf.String()

	// Severity section headings should be absent when there are no issues.
	for _, heading := range []string{"## Critical Issues", "## Warning Issues", "## Info Issues"} {
		if strings.Contains(out, heading) {
			t.Errorf("output should not contain %q when there are no issues", heading)
		}
	}

	// Lighthouse section should be absent.
	if strings.Contains(out, "## Lighthouse Scores") {
		t.Error("output should not contain Lighthouse section when there are no lighthouse results")
	}
}
