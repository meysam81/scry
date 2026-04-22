package report

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/meysam81/scry/core/model"
)

func TestJSONReporterRoundTrip(t *testing.T) {
	r := &JSONReporter{}
	var buf bytes.Buffer

	original := &model.CrawlResult{
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
			{
				CheckName: "seo/title-length",
				Severity:  model.SeverityWarning,
				Message:   "Title too short",
				URL:       "https://example.com/about",
			},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Duration:  5 * time.Second,
	}

	err := r.Write(context.Background(), original, &buf)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var decoded model.CrawlResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if decoded.SeedURL != original.SeedURL {
		t.Errorf("SeedURL = %q, want %q", decoded.SeedURL, original.SeedURL)
	}
	if len(decoded.Pages) != len(original.Pages) {
		t.Errorf("Pages count = %d, want %d", len(decoded.Pages), len(original.Pages))
	}
	if len(decoded.Issues) != len(original.Issues) {
		t.Fatalf("Issues count = %d, want %d", len(decoded.Issues), len(original.Issues))
	}
	for i, iss := range decoded.Issues {
		want := original.Issues[i]
		if iss.CheckName != want.CheckName {
			t.Errorf("Issue[%d].CheckName = %q, want %q", i, iss.CheckName, want.CheckName)
		}
		if iss.Severity != want.Severity {
			t.Errorf("Issue[%d].Severity = %q, want %q", i, iss.Severity, want.Severity)
		}
		if iss.Message != want.Message {
			t.Errorf("Issue[%d].Message = %q, want %q", i, iss.Message, want.Message)
		}
		if iss.URL != want.URL {
			t.Errorf("Issue[%d].URL = %q, want %q", i, iss.URL, want.URL)
		}
	}
}

func TestJSONReporterEmptyResult(t *testing.T) {
	r := &JSONReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	err := r.Write(context.Background(), result, &buf)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	// Must be valid JSON.
	if !json.Valid(buf.Bytes()) {
		t.Fatal("output is not valid JSON")
	}
}

func TestJSONReporterIssuesPreserved(t *testing.T) {
	r := &JSONReporter{}
	var buf bytes.Buffer

	issues := []model.Issue{
		{CheckName: "a", Severity: model.SeverityCritical, Message: "msg-a", URL: "https://a.com"},
		{CheckName: "b", Severity: model.SeverityWarning, Message: "msg-b", URL: "https://b.com"},
		{CheckName: "c", Severity: model.SeverityInfo, Message: "msg-c", URL: "https://c.com"},
	}
	result := &model.CrawlResult{
		SeedURL:   "https://test.com",
		Issues:    issues,
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	err := r.Write(context.Background(), result, &buf)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	var decoded model.CrawlResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if len(decoded.Issues) != len(issues) {
		t.Fatalf("Issues count = %d, want %d", len(decoded.Issues), len(issues))
	}

	for i, got := range decoded.Issues {
		want := issues[i]
		if got.CheckName != want.CheckName || got.Severity != want.Severity ||
			got.Message != want.Message || got.URL != want.URL {
			t.Errorf("Issue[%d] mismatch: got %+v, want %+v", i, got, want)
		}
	}
}
