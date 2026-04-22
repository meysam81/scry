package report

import (
	"bytes"
	"context"
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/meysam81/scry/core/model"
)

func TestJUnitReporterName(t *testing.T) {
	r := &JUnitReporter{}
	if r.Name() != "junit" {
		t.Errorf("Name() = %q, want %q", r.Name(), "junit")
	}
}

func TestJUnitReporterNilResult(t *testing.T) {
	r := &JUnitReporter{}
	var buf bytes.Buffer

	if err := r.Write(context.Background(), nil, &buf); err != nil {
		t.Fatalf("Write(nil) error: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for nil result, got %d bytes", buf.Len())
	}
}

func TestJUnitReporterValidXML(t *testing.T) {
	r := &JUnitReporter{}
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
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  5 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// Must be well-formed XML.
	var doc junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("xml.Unmarshal() error: %v", err)
	}
}

func TestJUnitReporterXMLHeader(t *testing.T) {
	r := &JUnitReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL:   "https://example.com",
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	if !strings.HasPrefix(buf.String(), "<?xml") {
		t.Error("output does not start with XML declaration")
	}
}

func TestJUnitReporterGroupsByCategory(t *testing.T) {
	r := &JUnitReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{CheckName: "seo/missing-title", Severity: model.SeverityCritical, Message: "msg1", URL: "https://a.com"},
			{CheckName: "seo/title-length", Severity: model.SeverityWarning, Message: "msg2", URL: "https://b.com"},
			{CheckName: "health/4xx", Severity: model.SeverityCritical, Message: "msg3", URL: "https://c.com"},
			{CheckName: "perf/large-image", Severity: model.SeverityInfo, Message: "msg4", URL: "https://d.com"},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  3 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("xml.Unmarshal() error: %v", err)
	}

	if doc.Name != "scry" {
		t.Errorf("testsuites name = %q, want %q", doc.Name, "scry")
	}
	if doc.Tests != 4 {
		t.Errorf("testsuites tests = %d, want 4", doc.Tests)
	}
	if doc.Failures != 4 {
		t.Errorf("testsuites failures = %d, want 4", doc.Failures)
	}

	// Expect 3 suites: seo, health, perf.
	if len(doc.TestSuites) != 3 {
		t.Fatalf("testsuite count = %d, want 3", len(doc.TestSuites))
	}

	// Find the seo suite (should have 2 test cases).
	var seoSuite *junitTestSuite
	for i := range doc.TestSuites {
		if doc.TestSuites[i].Name == "seo" {
			seoSuite = &doc.TestSuites[i]
			break
		}
	}
	if seoSuite == nil {
		t.Fatal("missing seo testsuite")
	}
	if seoSuite.Tests != 2 {
		t.Errorf("seo testsuite tests = %d, want 2", seoSuite.Tests)
	}
	if seoSuite.Failures != 2 {
		t.Errorf("seo testsuite failures = %d, want 2", seoSuite.Failures)
	}
}

func TestJUnitReporterTestCaseFields(t *testing.T) {
	r := &JUnitReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{
				CheckName: "seo/missing-title",
				Severity:  model.SeverityCritical,
				Message:   "page is missing a title tag",
				URL:       "https://example.com/page",
				Detail:    "no <title> found in <head>",
			},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  2 * time.Second,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("xml.Unmarshal() error: %v", err)
	}

	tc := doc.TestSuites[0].TestCases[0]
	if tc.Name != "seo/missing-title" {
		t.Errorf("testcase name = %q, want %q", tc.Name, "seo/missing-title")
	}
	if tc.ClassName != "https://example.com/page" {
		t.Errorf("testcase classname = %q, want %q", tc.ClassName, "https://example.com/page")
	}
	if tc.Failure == nil {
		t.Fatal("testcase failure is nil, want non-nil")
	}
	if tc.Failure.Message != "page is missing a title tag" {
		t.Errorf("failure message = %q, want %q", tc.Failure.Message, "page is missing a title tag")
	}
	if tc.Failure.Type != "critical" {
		t.Errorf("failure type = %q, want %q", tc.Failure.Type, "critical")
	}
	if tc.Failure.Body != "no <title> found in <head>" {
		t.Errorf("failure body = %q, want %q", tc.Failure.Body, "no <title> found in <head>")
	}
}

func TestJUnitReporterDuration(t *testing.T) {
	r := &JUnitReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL:   "https://example.com",
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  12345 * time.Millisecond,
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("xml.Unmarshal() error: %v", err)
	}

	if doc.Time != "12.345" {
		t.Errorf("time = %q, want %q", doc.Time, "12.345")
	}
}

func TestJUnitReporterEmptyIssues(t *testing.T) {
	r := &JUnitReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL:   "https://example.com",
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("xml.Unmarshal() error: %v", err)
	}

	if doc.Tests != 0 {
		t.Errorf("tests = %d, want 0", doc.Tests)
	}
	if doc.Failures != 0 {
		t.Errorf("failures = %d, want 0", doc.Failures)
	}
	if len(doc.TestSuites) != 0 {
		t.Errorf("testsuites count = %d, want 0", len(doc.TestSuites))
	}
}

func TestJUnitReporterNoCategorySlash(t *testing.T) {
	r := &JUnitReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{CheckName: "orphan-check", Severity: model.SeverityWarning, Message: "msg", URL: "https://a.com"},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var doc junitTestSuites
	if err := xml.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("xml.Unmarshal() error: %v", err)
	}

	if len(doc.TestSuites) != 1 {
		t.Fatalf("testsuites count = %d, want 1", len(doc.TestSuites))
	}
	if doc.TestSuites[0].Name != "orphan-check" {
		t.Errorf("suite name = %q, want %q", doc.TestSuites[0].Name, "orphan-check")
	}
}

func TestIssueCategoryHelper(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"seo/missing-title", "seo"},
		{"health/4xx", "health"},
		{"perf/large-image", "perf"},
		{"orphan-check", "orphan-check"},
		{"a/b/c", "a"},
	}
	for _, tt := range tests {
		got := issueCategory(tt.input)
		if got != tt.want {
			t.Errorf("issueCategory(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
