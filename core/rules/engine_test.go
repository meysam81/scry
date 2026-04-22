package rules

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/core/model"
)

func TestEvaluate_StatusCode(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/4xx",
			Severity:  "critical",
			Condition: "page.status_code >= 400",
			Message:   "Page returned client error",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	page := &model.Page{
		URL:        "https://example.com/missing",
		StatusCode: 404,
	}

	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	iss := issues[0]
	if iss.CheckName != "test/4xx" {
		t.Errorf("CheckName = %q, want %q", iss.CheckName, "test/4xx")
	}
	if iss.Severity != model.SeverityCritical {
		t.Errorf("Severity = %q, want %q", iss.Severity, model.SeverityCritical)
	}
	if iss.URL != page.URL {
		t.Errorf("URL = %q, want %q", iss.URL, page.URL)
	}
}

func TestEvaluate_StatusCode_NoMatch(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/4xx",
			Severity:  "critical",
			Condition: "page.status_code >= 400",
			Message:   "Page returned client error",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	page := &model.Page{
		URL:        "https://example.com/",
		StatusCode: 200,
	}

	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(issues))
	}
}

func TestEvaluate_MissingHeader(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/missing-csp",
			Severity:  "warning",
			Condition: "!('content-security-policy' in page.headers)",
			Message:   "Missing CSP header",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	// Page without CSP header.
	page := &model.Page{
		URL:        "https://example.com/",
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": []string{"text/html"}},
	}

	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].CheckName != "test/missing-csp" {
		t.Errorf("CheckName = %q, want %q", issues[0].CheckName, "test/missing-csp")
	}
}

func TestEvaluate_HeaderPresent(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/missing-csp",
			Severity:  "warning",
			Condition: "!('content-security-policy' in page.headers)",
			Message:   "Missing CSP header",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	page := &model.Page{
		URL:        "https://example.com/",
		StatusCode: 200,
		Headers: http.Header{
			"Content-Security-Policy": []string{"default-src 'self'"},
		},
	}

	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues, got %d", len(issues))
	}
}

func TestEvaluate_BodyContains(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/missing-title",
			Severity:  "warning",
			Condition: "!page.body.contains('<title>')",
			Message:   "Missing title tag",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	// Page without <title>.
	page := &model.Page{
		URL:        "https://example.com/",
		StatusCode: 200,
		Body:       []byte("<html><head></head><body></body></html>"),
	}

	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	// Page with <title>.
	page.Body = []byte("<html><head><title>Hello</title></head><body></body></html>")
	issues = engine.Evaluate(context.Background(), page)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues for page with title, got %d", len(issues))
	}
}

func TestEvaluate_EmptyLinks(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/no-links",
			Severity:  "info",
			Condition: "size(page.links) == 0",
			Message:   "No outgoing links",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	page := &model.Page{
		URL:        "https://example.com/",
		StatusCode: 200,
		Links:      nil,
	}

	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	page.Links = []string{"https://example.com/about"}
	issues = engine.Evaluate(context.Background(), page)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues when links present, got %d", len(issues))
	}
}

func TestEvaluate_URLMatchPattern(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/https-only",
			Severity:  "warning",
			Match:     "https://*",
			Condition: "page.status_code == 200",
			Message:   "HTTPS page found",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	// HTTPS URL should match.
	page := &model.Page{
		URL:        "https://example.com",
		StatusCode: 200,
	}
	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for HTTPS URL, got %d", len(issues))
	}

	// HTTP URL should not match the glob.
	page.URL = "http://example.com"
	issues = engine.Evaluate(context.Background(), page)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues for HTTP URL, got %d", len(issues))
	}
}

func TestEvaluate_MatchWildcardAndEmpty(t *testing.T) {
	for _, pattern := range []string{"", "*"} {
		engine, err := NewEngine([]Rule{
			{
				Name:      "test/always",
				Severity:  "info",
				Match:     pattern,
				Condition: "true",
				Message:   "always fires",
			},
		}, logger.Nop())
		if err != nil {
			t.Fatalf("NewEngine(match=%q): %v", pattern, err)
		}

		page := &model.Page{URL: "https://anything.example.com/page"}
		issues := engine.Evaluate(context.Background(), page)
		if len(issues) != 1 {
			t.Errorf("match=%q: expected 1 issue, got %d", pattern, len(issues))
		}
	}
}

func TestNewEngine_InvalidCEL(t *testing.T) {
	_, err := NewEngine([]Rule{
		{
			Name:      "test/bad",
			Severity:  "info",
			Condition: "this is not valid CEL !!!",
			Message:   "should not compile",
		},
	}, logger.Nop())
	if err == nil {
		t.Fatal("expected error for invalid CEL expression, got nil")
	}
}

func TestNewEngine_InvalidGlob(t *testing.T) {
	_, err := NewEngine([]Rule{
		{
			Name:      "test/bad-glob",
			Severity:  "info",
			Match:     "[invalid",
			Condition: "true",
			Message:   "bad glob",
		},
	}, logger.Nop())
	if err == nil {
		t.Fatal("expected error for invalid glob pattern, got nil")
	}
}

func TestEvaluate_MultipleRules(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/4xx",
			Severity:  "critical",
			Condition: "page.status_code >= 400",
			Message:   "Client error",
		},
		{
			Name:      "test/no-links",
			Severity:  "info",
			Condition: "size(page.links) == 0",
			Message:   "No links",
		},
		{
			Name:      "test/fast",
			Severity:  "info",
			Condition: "page.fetch_duration_ms < 100",
			Message:   "Fast page",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	page := &model.Page{
		URL:           "https://example.com/missing",
		StatusCode:    404,
		Links:         nil,
		FetchDuration: 50 * time.Millisecond,
	}

	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d: %+v", len(issues), issues)
	}

	// Verify all three rules fired.
	names := map[string]bool{}
	for _, iss := range issues {
		names[iss.CheckName] = true
	}
	for _, want := range []string{"test/4xx", "test/no-links", "test/fast"} {
		if !names[want] {
			t.Errorf("expected issue %q to fire", want)
		}
	}
}

func TestEvaluate_InSitemap(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/not-in-sitemap",
			Severity:  "info",
			Condition: "!page.in_sitemap",
			Message:   "Not in sitemap",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	page := &model.Page{
		URL:        "https://example.com/orphan",
		StatusCode: 200,
		InSitemap:  false,
	}

	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	page.InSitemap = true
	issues = engine.Evaluate(context.Background(), page)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues when in sitemap, got %d", len(issues))
	}
}

func TestEvaluate_FetchDuration(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/slow",
			Severity:  "warning",
			Condition: "page.fetch_duration_ms > 2000",
			Message:   "Slow page",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	page := &model.Page{
		URL:           "https://example.com/slow",
		StatusCode:    200,
		FetchDuration: 3 * time.Second,
	}

	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	page.FetchDuration = 500 * time.Millisecond
	issues = engine.Evaluate(context.Background(), page)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues for fast page, got %d", len(issues))
	}
}

func TestEvaluate_NilHeaders(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/header-check",
			Severity:  "info",
			Condition: "!('x-custom' in page.headers)",
			Message:   "Missing custom header",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	page := &model.Page{
		URL:        "https://example.com/",
		StatusCode: 200,
		Headers:    nil,
	}

	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue for nil headers, got %d", len(issues))
	}
}

func TestEvaluate_BodySize(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/large-page",
			Severity:  "info",
			Condition: "size(page.body) > 100",
			Message:   "Large page",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	page := &model.Page{
		URL:        "https://example.com/",
		StatusCode: 200,
		Body:       make([]byte, 200),
	}

	issues := engine.Evaluate(context.Background(), page)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}

	page.Body = make([]byte, 50)
	issues = engine.Evaluate(context.Background(), page)
	if len(issues) != 0 {
		t.Fatalf("expected 0 issues for small page, got %d", len(issues))
	}
}

func TestRuleCount(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{Name: "a", Severity: "info", Condition: "true", Message: "a"},
		{Name: "b", Severity: "info", Condition: "true", Message: "b"},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	if got := engine.RuleCount(); got != 2 {
		t.Errorf("RuleCount() = %d, want 2", got)
	}
}

func TestRuleChecker_Interface(t *testing.T) {
	engine, err := NewEngine([]Rule{
		{
			Name:      "test/always",
			Severity:  "info",
			Condition: "true",
			Message:   "always",
		},
	}, logger.Nop())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	checker := NewRuleChecker(engine)
	if checker.Name() != "rules" {
		t.Errorf("Name() = %q, want %q", checker.Name(), "rules")
	}

	page := &model.Page{URL: "https://example.com/", StatusCode: 200}
	issues := checker.Check(context.Background(), page)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue via RuleChecker, got %d", len(issues))
	}
}

func TestLoadRuleFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/rules.yml"

	content := `rules:
  - name: "test/example"
    severity: warning
    condition: "page.status_code >= 400"
    message: "Error page"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	rf, err := LoadRuleFile(path)
	if err != nil {
		t.Fatalf("LoadRuleFile: %v", err)
	}

	if len(rf.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rf.Rules))
	}
	if rf.Rules[0].Name != "test/example" {
		t.Errorf("rule name = %q, want %q", rf.Rules[0].Name, "test/example")
	}
}

func TestLoadRuleFile_NotFound(t *testing.T) {
	_, err := LoadRuleFile("/nonexistent/path.yml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadRuleFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/bad.yml"

	if err := os.WriteFile(path, []byte("{{not yaml"), 0o644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	_, err := LoadRuleFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
