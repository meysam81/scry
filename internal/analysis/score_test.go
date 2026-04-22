package analysis

import (
	"testing"
	"time"

	"github.com/meysam81/scry/core/model"
)

func makeCrawlResult(issues []model.Issue, pageCount int) *model.CrawlResult {
	pages := make([]*model.Page, pageCount)
	for i := range pages {
		pages[i] = &model.Page{URL: "https://example.com/page"}
	}
	return &model.CrawlResult{
		SeedURL:   "https://example.com",
		Pages:     pages,
		Issues:    issues,
		CrawledAt: time.Now(),
		Duration:  time.Second,
	}
}

func TestComputeScore_NoIssues(t *testing.T) {
	cr := makeCrawlResult(nil, 5)
	sr := ComputeScore(cr)

	if sr.Overall != 100 {
		t.Errorf("expected overall 100 with no issues, got %d", sr.Overall)
	}
	if sr.TotalPages != 5 {
		t.Errorf("expected 5 total pages, got %d", sr.TotalPages)
	}
	if sr.TotalIssues != 0 {
		t.Errorf("expected 0 total issues, got %d", sr.TotalIssues)
	}
	for cat, score := range sr.Categories {
		if score != 100 {
			t.Errorf("category %q should be 100, got %d", cat, score)
		}
	}
}

func TestComputeScore_OnlyInfoIssues(t *testing.T) {
	issues := []model.Issue{
		{CheckName: "seo/missing-title", Severity: model.SeverityInfo},
		{CheckName: "seo/missing-h1", Severity: model.SeverityInfo},
		{CheckName: "health/slow-ttfb", Severity: model.SeverityInfo},
	}
	cr := makeCrawlResult(issues, 3)
	sr := ComputeScore(cr)

	// 3 info issues = 3 * 1 = 3 deduction → 100 - 3 = 97
	expected := 97
	if sr.Overall != expected {
		t.Errorf("expected overall %d with 3 info issues, got %d", expected, sr.Overall)
	}

	// SEO: 2 info = 2 deduction → 98
	if sr.Categories["seo"] != 98 {
		t.Errorf("expected seo category 98, got %d", sr.Categories["seo"])
	}
	// Health: 1 info = 1 deduction → 99
	if sr.Categories["health"] != 99 {
		t.Errorf("expected health category 99, got %d", sr.Categories["health"])
	}
}

func TestComputeScore_ManyCritical_NearZero(t *testing.T) {
	issues := make([]model.Issue, 25)
	for i := range issues {
		issues[i] = model.Issue{
			CheckName: "seo/missing-title",
			Severity:  model.SeverityCritical,
		}
	}
	cr := makeCrawlResult(issues, 10)
	sr := ComputeScore(cr)

	// 25 critical = 25 * 10 = 250 deduction → 100 - 250 = -150, floored at 0
	if sr.Overall != 0 {
		t.Errorf("expected overall 0 with 25 critical issues, got %d", sr.Overall)
	}
	if sr.Categories["seo"] != 0 {
		t.Errorf("expected seo category 0, got %d", sr.Categories["seo"])
	}
}

func TestComputeScore_FloorAtZero(t *testing.T) {
	issues := make([]model.Issue, 50)
	for i := range issues {
		issues[i] = model.Issue{
			CheckName: "performance/large-html",
			Severity:  model.SeverityCritical,
		}
	}
	cr := makeCrawlResult(issues, 1)
	sr := ComputeScore(cr)

	if sr.Overall != 0 {
		t.Errorf("expected floor at 0, got %d", sr.Overall)
	}
	if sr.Categories["performance"] != 0 {
		t.Errorf("expected performance floor at 0, got %d", sr.Categories["performance"])
	}
}

func TestComputeScore_CategoryBreakdown(t *testing.T) {
	issues := []model.Issue{
		{CheckName: "seo/missing-title", Severity: model.SeverityCritical},
		{CheckName: "seo/missing-h1", Severity: model.SeverityWarning},
		{CheckName: "health/5xx", Severity: model.SeverityCritical},
		{CheckName: "images/missing-alt", Severity: model.SeverityWarning},
		{CheckName: "links/broken-internal", Severity: model.SeverityCritical},
		{CheckName: "security/missing-hsts", Severity: model.SeverityInfo},
		{CheckName: "performance/large-html", Severity: model.SeverityWarning},
		{CheckName: "structured-data/missing-json-ld", Severity: model.SeverityInfo},
		{CheckName: "content/thin-content", Severity: model.SeverityInfo},
		{CheckName: "accessibility/missing-form-label", Severity: model.SeverityWarning},
	}
	cr := makeCrawlResult(issues, 10)
	sr := ComputeScore(cr)

	// SEO: 1 critical (10) + 1 warning (3) = 13 → 87
	if sr.Categories["seo"] != 87 {
		t.Errorf("expected seo 87, got %d", sr.Categories["seo"])
	}
	// Health: 1 critical = 10 → 90
	if sr.Categories["health"] != 90 {
		t.Errorf("expected health 90, got %d", sr.Categories["health"])
	}
	// Images: 1 warning = 3 → 97
	if sr.Categories["images"] != 97 {
		t.Errorf("expected images 97, got %d", sr.Categories["images"])
	}
	// Links: 1 critical = 10 → 90
	if sr.Categories["links"] != 90 {
		t.Errorf("expected links 90, got %d", sr.Categories["links"])
	}
	// Security: 1 info = 1 → 99
	if sr.Categories["security"] != 99 {
		t.Errorf("expected security 99, got %d", sr.Categories["security"])
	}
	// Performance: 1 warning = 3 → 97
	if sr.Categories["performance"] != 97 {
		t.Errorf("expected performance 97, got %d", sr.Categories["performance"])
	}
	// Structured-data: 1 info = 1 → 99
	if sr.Categories["structured-data"] != 99 {
		t.Errorf("expected structured-data 99, got %d", sr.Categories["structured-data"])
	}
	// Content: 1 info = 1 → 99
	if sr.Categories["content"] != 99 {
		t.Errorf("expected content 99, got %d", sr.Categories["content"])
	}
	// Accessibility: 1 warning = 3 → 97
	if sr.Categories["accessibility"] != 97 {
		t.Errorf("expected accessibility 97, got %d", sr.Categories["accessibility"])
	}
}

func TestComputeScore_SeverityBreakdown(t *testing.T) {
	issues := []model.Issue{
		{CheckName: "seo/missing-title", Severity: model.SeverityCritical},
		{CheckName: "seo/missing-h1", Severity: model.SeverityCritical},
		{CheckName: "health/5xx", Severity: model.SeverityWarning},
		{CheckName: "images/missing-alt", Severity: model.SeverityInfo},
	}
	cr := makeCrawlResult(issues, 5)
	sr := ComputeScore(cr)

	if sr.Breakdown["critical"] != 2 {
		t.Errorf("expected 2 critical in breakdown, got %d", sr.Breakdown["critical"])
	}
	if sr.Breakdown["warning"] != 1 {
		t.Errorf("expected 1 warning in breakdown, got %d", sr.Breakdown["warning"])
	}
	if sr.Breakdown["info"] != 1 {
		t.Errorf("expected 1 info in breakdown, got %d", sr.Breakdown["info"])
	}
	if sr.TotalIssues != 4 {
		t.Errorf("expected 4 total issues, got %d", sr.TotalIssues)
	}
}

func TestComputeScore_ExternalLinksCategory(t *testing.T) {
	issues := []model.Issue{
		{CheckName: "external-links/broken", Severity: model.SeverityCritical},
		{CheckName: "external-links/timeout", Severity: model.SeverityWarning},
	}
	cr := makeCrawlResult(issues, 1)
	sr := ComputeScore(cr)

	// external-links maps to "links" category: 1 critical (10) + 1 warning (3) = 13 → 87
	if sr.Categories["links"] != 87 {
		t.Errorf("expected links 87 (external-links mapped), got %d", sr.Categories["links"])
	}
}

func TestComputeScore_TLSMapsToSecurity(t *testing.T) {
	issues := []model.Issue{
		{CheckName: "tls/self-signed", Severity: model.SeverityCritical},
	}
	cr := makeCrawlResult(issues, 1)
	sr := ComputeScore(cr)

	// tls maps to security: 1 critical = 10 → 90
	if sr.Categories["security"] != 90 {
		t.Errorf("expected security 90 (tls mapped), got %d", sr.Categories["security"])
	}
}

func TestComputeScore_HreflangMapsToSEO(t *testing.T) {
	issues := []model.Issue{
		{CheckName: "hreflang/missing-x-default", Severity: model.SeverityWarning},
	}
	cr := makeCrawlResult(issues, 1)
	sr := ComputeScore(cr)

	// hreflang maps to seo: 1 warning = 3 → 97
	if sr.Categories["seo"] != 97 {
		t.Errorf("expected seo 97 (hreflang mapped), got %d", sr.Categories["seo"])
	}
}

func TestComputeScore_MixedSeverities(t *testing.T) {
	issues := []model.Issue{
		{CheckName: "seo/a", Severity: model.SeverityCritical},   // -5
		{CheckName: "seo/b", Severity: model.SeverityWarning},    // -2
		{CheckName: "seo/c", Severity: model.SeverityInfo},       // -0.5
		{CheckName: "health/a", Severity: model.SeverityWarning}, // -2
	}
	cr := makeCrawlResult(issues, 3)
	sr := ComputeScore(cr)

	// Expected overall: 100 minus (10+3+1+3) deductions equals 83.
	if sr.Overall != 83 {
		t.Errorf("expected overall 83, got %d", sr.Overall)
	}
}

func TestComputeScore_UnknownPrefixIgnored(t *testing.T) {
	issues := []model.Issue{
		{CheckName: "foobar/something", Severity: model.SeverityCritical},
	}
	cr := makeCrawlResult(issues, 1)
	sr := ComputeScore(cr)

	// Overall still deducted.
	if sr.Overall != 90 {
		t.Errorf("expected overall 90, got %d", sr.Overall)
	}
	// All named categories should remain 100 since foobar is unknown.
	for cat, score := range sr.Categories {
		if score != 100 {
			t.Errorf("category %q should be 100 for unknown prefix, got %d", cat, score)
		}
	}
}

func TestComputeScore_EmptyResult(t *testing.T) {
	cr := makeCrawlResult(nil, 0)
	sr := ComputeScore(cr)

	if sr.Overall != 100 {
		t.Errorf("expected 100 for empty result, got %d", sr.Overall)
	}
	if sr.TotalPages != 0 {
		t.Errorf("expected 0 pages, got %d", sr.TotalPages)
	}
	if sr.TotalIssues != 0 {
		t.Errorf("expected 0 issues, got %d", sr.TotalIssues)
	}
}

func TestSortedCategories(t *testing.T) {
	cr := makeCrawlResult(nil, 1)
	sr := ComputeScore(cr)
	cats := SortedCategories(sr)

	for i := 1; i < len(cats); i++ {
		if cats[i-1] >= cats[i] {
			t.Errorf("categories not sorted: %v", cats)
			break
		}
	}
}

func TestCategoryFromCheckName(t *testing.T) {
	tests := []struct {
		checkName string
		want      string
	}{
		{"seo/missing-title", "seo"},
		{"health/5xx", "health"},
		{"performance/large-html", "performance"},
		{"security/missing-hsts", "security"},
		{"accessibility/missing-form-label", "accessibility"},
		{"links/broken-internal", "links"},
		{"external-links/broken", "links"},
		{"images/missing-alt", "images"},
		{"structured-data/missing-json-ld", "structured-data"},
		{"content/thin-content", "content"},
		{"hreflang/missing-x-default", "seo"},
		{"tls/self-signed", "security"},
		{"unknown/thing", ""},
		{"noprefix", ""},
	}
	for _, tt := range tests {
		got := categoryFromCheckName(tt.checkName)
		if got != tt.want {
			t.Errorf("categoryFromCheckName(%q) = %q, want %q", tt.checkName, got, tt.want)
		}
	}
}

func TestDeduction(t *testing.T) {
	tests := []struct {
		severity model.Severity
		want     float64
	}{
		{model.SeverityCritical, 10},
		{model.SeverityWarning, 3},
		{model.SeverityInfo, 1},
		{model.Severity("unknown"), 0},
	}
	for _, tt := range tests {
		got := deduction(tt.severity)
		if got != tt.want {
			t.Errorf("deduction(%q) = %f, want %f", tt.severity, got, tt.want)
		}
	}
}
