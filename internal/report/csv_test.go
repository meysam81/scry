package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/meysam81/scry/core/model"
)

func TestCSVReporterWriteAndParse(t *testing.T) {
	r := &CSVReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			{
				CheckName: "health/4xx",
				Severity:  model.SeverityCritical,
				Message:   "Page returned status 404",
				URL:       "https://example.com/broken",
				Detail:    "HTTP 404",
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

	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("csv.ReadAll() error: %v", err)
	}

	// Header + 3 issue rows.
	wantRows := 4
	if len(records) != wantRows {
		t.Fatalf("row count = %d, want %d", len(records), wantRows)
	}

	// Verify header.
	expectedHeader := []string{"url", "severity", "check", "message", "detail"}
	for i, col := range expectedHeader {
		if records[0][i] != col {
			t.Errorf("header[%d] = %q, want %q", i, records[0][i], col)
		}
	}

	// Verify first data row fields.
	firstRow := records[1]
	if firstRow[0] != "https://example.com/broken" {
		t.Errorf("row 1 url = %q, want %q", firstRow[0], "https://example.com/broken")
	}
	if firstRow[1] != "critical" {
		t.Errorf("row 1 severity = %q, want %q", firstRow[1], "critical")
	}
	if firstRow[2] != "health/4xx" {
		t.Errorf("row 1 check = %q, want %q", firstRow[2], "health/4xx")
	}
	if firstRow[3] != "Page returned status 404" {
		t.Errorf("row 1 message = %q, want %q", firstRow[3], "Page returned status 404")
	}
	if firstRow[4] != "HTTP 404" {
		t.Errorf("row 1 detail = %q, want %q", firstRow[4], "HTTP 404")
	}
}

func TestCSVReporterEmptyIssues(t *testing.T) {
	r := &CSVReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL:   "https://example.com",
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("csv.ReadAll() error: %v", err)
	}

	// Only the header row should be present.
	if len(records) != 1 {
		t.Fatalf("row count = %d, want 1 (header only)", len(records))
	}
}

func TestCSVReporterAllFieldsPresent(t *testing.T) {
	r := &CSVReporter{}
	var buf bytes.Buffer

	result := &model.CrawlResult{
		SeedURL: "https://test.com",
		Issues: []model.Issue{
			{
				CheckName: "check-a",
				Severity:  model.SeverityWarning,
				Message:   "something wrong",
				URL:       "https://test.com/page",
				Detail:    "extra detail",
			},
		},
		CrawledAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("csv.ReadAll() error: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("row count = %d, want 2", len(records))
	}

	row := records[1]
	expectedCols := 5
	if len(row) != expectedCols {
		t.Fatalf("column count = %d, want %d", len(row), expectedCols)
	}

	// Verify column order: url, severity, check, message, detail.
	checks := []struct {
		col  int
		want string
		name string
	}{
		{0, "https://test.com/page", "url"},
		{1, "warning", "severity"},
		{2, "check-a", "check"},
		{3, "something wrong", "message"},
		{4, "extra detail", "detail"},
	}
	for _, c := range checks {
		if row[c.col] != c.want {
			t.Errorf("%s (col %d) = %q, want %q", c.name, c.col, row[c.col], c.want)
		}
	}
}

func TestSanitizeCSVCell(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"", ""},
		{"=cmd|'/C calc'!A0", "\t=cmd|'/C calc'!A0"},
		{"+cmd|'/C calc'!A0", "\t+cmd|'/C calc'!A0"},
		{"-cmd|'/C calc'!A0", "\t-cmd|'/C calc'!A0"},
		{"@SUM(A1:A2)", "\t@SUM(A1:A2)"},
		{"\tcmd", "\t\tcmd"},
		{"\rcmd", "\t\rcmd"},
		{"normal text", "normal text"},
		{"https://example.com", "https://example.com"},
	}
	for _, tt := range tests {
		got := sanitizeCSVCell(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeCSVCell(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCSVReporter_FormulaInjection(t *testing.T) {
	result := &model.CrawlResult{
		Issues: []model.Issue{
			{
				URL:       "https://example.com",
				Severity:  model.SeverityWarning,
				CheckName: "test/check",
				Message:   "=cmd|'/C calc'!A0",
				Detail:    "+malicious",
			},
		},
	}

	var buf bytes.Buffer
	r := &CSVReporter{}
	if err := r.Write(context.Background(), result, &buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	lines := strings.Split(buf.String(), "\n")
	if len(lines) < 2 {
		t.Fatal("expected at least 2 lines (header + data)")
	}

	// The message and detail fields should be sanitized.
	dataLine := lines[1]
	if strings.Contains(dataLine, "=cmd") && !strings.Contains(dataLine, "\t=cmd") {
		t.Errorf("formula not sanitized in CSV output: %s", dataLine)
	}
}
