package report

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/meysam81/scry/core/model"
)

func TestHTMLReporterValidHTML(t *testing.T) {
	r := &HTMLReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Pages: []*model.Page{
			{URL: "https://example.com", StatusCode: 200},
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
		Duration:  5 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	out := buf.String()

	for _, marker := range []string{"<!DOCTYPE html>", "<html", "</html>"} {
		if !strings.Contains(out, marker) {
			t.Errorf("output missing HTML marker %q", marker)
		}
	}
}

func TestHTMLReporterContainsIssueData(t *testing.T) {
	r := &HTMLReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{
				CheckName: "seo/title-length",
				Severity:  model.SeverityWarning,
				Message:   "Title too short",
				URL:       "https://example.com/about",
			},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  1 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	out := buf.String()

	// Check that issue data appears in the output.
	for _, want := range []string{"seo/title-length", "https://example.com/about", "Title too short"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing issue data %q", want)
		}
	}
}

func TestHTMLReporterContainsFilterElements(t *testing.T) {
	r := &HTMLReporter{}
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

	// Severity dropdown.
	if !strings.Contains(out, "severity-filter") {
		t.Error("output missing severity filter dropdown")
	}

	// Search input.
	if !strings.Contains(out, "search-input") {
		t.Error("output missing search input")
	}
}

func TestHTMLReporterWithLighthouse(t *testing.T) {
	r := &HTMLReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Lighthouse: []model.LighthouseResult{
			{
				URL:                "https://example.com",
				PerformanceScore:   85,
				AccessibilityScore: 92,
				BestPracticesScore: 88,
				SEOScore:           90,
			},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  1 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "Lighthouse Scores") {
		t.Error("output missing Lighthouse Scores section")
	}

	// Scores > 1 are treated as already 0-100.
	for _, score := range []string{"85", "92", "88", "90"} {
		if !strings.Contains(out, score) {
			t.Errorf("output missing lighthouse score %q", score)
		}
	}
}

func TestHTMLReporterEmptyIssues(t *testing.T) {
	r := &HTMLReporter{}
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

	// Should still be valid HTML.
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Error("empty issues output missing DOCTYPE")
	}

	// Should not contain Lighthouse section when there is no data.
	if strings.Contains(out, "Lighthouse Scores") {
		t.Error("output should not contain Lighthouse section when there are no lighthouse results")
	}
}
