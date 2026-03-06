package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSeverityLevel(t *testing.T) {
	tests := []struct {
		sev  Severity
		want int
	}{
		{SeverityCritical, 3},
		{SeverityWarning, 2},
		{SeverityInfo, 1},
		{Severity("bogus"), 0},
		{Severity(""), 0},
	}
	for _, tt := range tests {
		if got := tt.sev.Level(); got != tt.want {
			t.Errorf("Severity(%q).Level() = %d, want %d", tt.sev, got, tt.want)
		}
	}
}

func TestSeverityOrdering(t *testing.T) {
	if SeverityCritical.Level() <= SeverityWarning.Level() {
		t.Error("critical should be greater than warning")
	}
	if SeverityWarning.Level() <= SeverityInfo.Level() {
		t.Error("warning should be greater than info")
	}
	if SeverityInfo.Level() <= 0 {
		t.Error("info should be greater than unknown (0)")
	}
}

func TestSeverityFromString(t *testing.T) {
	tests := []struct {
		input string
		want  Severity
	}{
		{"critical", SeverityCritical},
		{"CRITICAL", SeverityCritical},
		{"Critical", SeverityCritical},
		{"warning", SeverityWarning},
		{"WARNING", SeverityWarning},
		{"info", SeverityInfo},
		{"INFO", SeverityInfo},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := SeverityFromString(tt.input); got != tt.want {
			t.Errorf("SeverityFromString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSeverityFromStringRoundTrip(t *testing.T) {
	for _, sev := range []Severity{SeverityCritical, SeverityWarning, SeverityInfo} {
		if got := SeverityFromString(string(sev)); got != sev {
			t.Errorf("round-trip failed for %q: got %q", sev, got)
		}
	}
}

func TestAtLeast(t *testing.T) {
	severities := []Severity{SeverityInfo, SeverityWarning, SeverityCritical}

	for _, recv := range severities {
		for _, thresh := range severities {
			got := recv.AtLeast(thresh)
			want := recv.Level() >= thresh.Level()
			if got != want {
				t.Errorf("%q.AtLeast(%q) = %v, want %v", recv, thresh, got, want)
			}
		}
	}
}

func TestAtLeastUnknown(t *testing.T) {
	unknown := Severity("nope")
	if unknown.AtLeast(SeverityInfo) {
		t.Error("unknown severity should not be at least info")
	}
	if !SeverityInfo.AtLeast(unknown) {
		t.Error("info should be at least unknown")
	}
}

func TestIssueJSON(t *testing.T) {
	issue := Issue{
		CheckName: "broken-link",
		Severity:  SeverityWarning,
		Message:   "link returns 404",
		URL:       "https://example.com/missing",
		Detail:    "checked at 2024-01-01",
	}

	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Issue
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.CheckName != issue.CheckName {
		t.Errorf("CheckName = %q, want %q", got.CheckName, issue.CheckName)
	}
	if got.Severity != issue.Severity {
		t.Errorf("Severity = %q, want %q", got.Severity, issue.Severity)
	}
	if got.Message != issue.Message {
		t.Errorf("Message = %q, want %q", got.Message, issue.Message)
	}
	if got.URL != issue.URL {
		t.Errorf("URL = %q, want %q", got.URL, issue.URL)
	}
	if got.Detail != issue.Detail {
		t.Errorf("Detail = %q, want %q", got.Detail, issue.Detail)
	}
}

func TestIssueJSONSnakeCase(t *testing.T) {
	data := `{"check_name":"test","severity":"info","message":"hello","url":"https://x.com"}`
	var issue Issue
	if err := json.Unmarshal([]byte(data), &issue); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if issue.CheckName != "test" {
		t.Errorf("CheckName = %q, want %q", issue.CheckName, "test")
	}
}

func TestCrawlResultJSON(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	cr := CrawlResult{
		SeedURL:   "https://example.com",
		Pages:     []*Page{{URL: "https://example.com", StatusCode: 200}},
		Issues:    []Issue{{CheckName: "test", Severity: SeverityInfo, Message: "ok", URL: "https://example.com"}},
		CrawledAt: now,
		Duration:  5 * time.Second,
	}

	data, err := json.Marshal(cr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got CrawlResult
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.SeedURL != cr.SeedURL {
		t.Errorf("SeedURL = %q, want %q", got.SeedURL, cr.SeedURL)
	}
	if len(got.Pages) != 1 {
		t.Fatalf("len(Pages) = %d, want 1", len(got.Pages))
	}
	if got.Pages[0].URL != "https://example.com" {
		t.Errorf("Pages[0].URL = %q, want %q", got.Pages[0].URL, "https://example.com")
	}
	if len(got.Issues) != 1 {
		t.Fatalf("len(Issues) = %d, want 1", len(got.Issues))
	}
	if got.Duration != cr.Duration {
		t.Errorf("Duration = %v, want %v", got.Duration, cr.Duration)
	}
}

func TestPageBodyExcludedFromJSON(t *testing.T) {
	page := Page{
		URL:        "https://example.com",
		StatusCode: 200,
		Body:       []byte("<html>hello</html>"),
	}

	data, err := json.Marshal(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Body should not appear in the JSON output.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, exists := raw["body"]; exists {
		t.Error("Body field should be excluded from JSON (json:\"-\") but was present")
	}

	// Unmarshalling back should leave Body nil.
	var got Page
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal page: %v", err)
	}
	if got.Body != nil {
		t.Errorf("Body = %v, want nil after JSON round-trip", got.Body)
	}
}

func TestOmitemptyOptionalSlices(t *testing.T) {
	// Issue with empty Detail should omit detail.
	issue := Issue{
		CheckName: "x",
		Severity:  SeverityInfo,
		Message:   "msg",
		URL:       "https://example.com",
	}
	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("marshal issue: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, exists := raw["detail"]; exists {
		t.Error("empty Detail should be omitted from JSON")
	}

	// Page with nil slices should omit them.
	page := Page{URL: "https://example.com", StatusCode: 200}
	data, err = json.Marshal(page)
	if err != nil {
		t.Fatalf("marshal page: %v", err)
	}
	raw = nil
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	for _, field := range []string{"redirect_chain", "links", "assets"} {
		if _, exists := raw[field]; exists {
			t.Errorf("nil %s should be omitted from JSON", field)
		}
	}

	// CrawlResult with nil Lighthouse should omit it.
	cr := CrawlResult{SeedURL: "https://example.com"}
	data, err = json.Marshal(cr)
	if err != nil {
		t.Fatalf("marshal crawl result: %v", err)
	}
	raw = nil
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, exists := raw["lighthouse"]; exists {
		t.Error("nil Lighthouse should be omitted from JSON")
	}
}
