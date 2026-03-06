package report

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/meysam81/scry/internal/model"
)

func sampleResult() *model.CrawlResult {
	return &model.CrawlResult{
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
			{
				CheckName: "seo/title-length",
				Severity:  model.SeverityWarning,
				Message:   "Title length is 25 characters (recommended: 30-60)",
				URL:       "https://example.com/about",
			},
			{
				CheckName: "perf/large-image",
				Severity:  model.SeverityInfo,
				Message:   "Image exceeds 500KB",
				URL:       "https://example.com/gallery",
			},
		},
		CrawledAt: time.Now(),
		Duration:  12400 * time.Millisecond,
	}
}

func TestTerminalReporterBasicOutput(t *testing.T) {
	r := &TerminalReporter{}
	var buf bytes.Buffer

	err := r.Write(context.Background(), sampleResult(), &buf)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	out := buf.String()

	// Check key text appears in output.
	mustContain := []string{
		"Scry Audit Report",
		"https://example.com",
		"Critical",
		"Warning",
		"Info",
		"health/4xx",
		"Page returned status 404",
		"seo/title-length",
		"perf/large-image",
		"12.4s",
	}

	for _, want := range mustContain {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestTerminalReporterEmptyIssues(t *testing.T) {
	r := &TerminalReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL:   "https://empty.example.com",
		Pages:     nil,
		Issues:    nil,
		CrawledAt: time.Now(),
		Duration:  500 * time.Millisecond,
	}

	err := r.Write(context.Background(), result, &buf)
	if err != nil {
		t.Fatalf("Write() error on empty result: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Scry Audit Report") {
		t.Error("output missing header on empty result")
	}
	// Should not contain issue sections when there are no issues.
	if strings.Contains(out, "-- Critical Issues") {
		t.Error("output should not show Critical Issues section when there are none")
	}
}

func TestTerminalReporterLighthouseScores(t *testing.T) {
	r := &TerminalReporter{}
	var buf bytes.Buffer

	result := sampleResult()
	result.Lighthouse = []model.LighthouseResult{
		{
			URL:                "https://example.com",
			PerformanceScore:   85,
			AccessibilityScore: 92,
			BestPracticesScore: 88,
			SEOScore:           90,
		},
	}

	err := r.Write(context.Background(), result, &buf)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	out := buf.String()
	mustContain := []string{
		"Lighthouse Scores",
		"85",
		"92",
		"88",
		"90",
	}
	for _, want := range mustContain {
		if !strings.Contains(out, want) {
			t.Errorf("lighthouse output missing %q", want)
		}
	}

	// go-pretty uppercases header text, so check case-insensitively.
	upper := strings.ToUpper(out)
	headerKeywords := []string{"PERFORMANCE", "ACCESSIBILITY", "BEST PRACTICES", "SEO"}
	for _, kw := range headerKeywords {
		if !strings.Contains(upper, kw) {
			t.Errorf("lighthouse output missing header %q (case-insensitive)", kw)
		}
	}
}

func TestTerminalReporterNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	r := &TerminalReporter{}
	var buf bytes.Buffer

	err := r.Write(context.Background(), sampleResult(), &buf)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	out := buf.String()

	// ANSI escape sequences start with ESC (0x1b).
	if strings.Contains(out, "\x1b[") {
		t.Error("output contains ANSI escape sequences when NO_COLOR is set")
	}

	// Should still contain the text content.
	if !strings.Contains(out, "Scry Audit Report") {
		t.Error("output missing header when NO_COLOR is set")
	}
}
