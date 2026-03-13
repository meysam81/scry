package report

import (
	"fmt"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestComputeSummaryEmpty(t *testing.T) {
	result := &model.CrawlResult{}
	stats := ComputeSummary(result)

	if stats.TotalIssues != 0 {
		t.Errorf("TotalIssues = %d, want 0", stats.TotalIssues)
	}
	if stats.PagesScanned != 0 {
		t.Errorf("PagesScanned = %d, want 0", stats.PagesScanned)
	}
	if len(stats.TopURLs) != 0 {
		t.Errorf("TopURLs length = %d, want 0", len(stats.TopURLs))
	}
	for sev, count := range stats.BySeverity {
		if count != 0 {
			t.Errorf("BySeverity[%s] = %d, want 0", sev, count)
		}
	}
	if len(stats.ByCategory) != 0 {
		t.Errorf("ByCategory length = %d, want 0", len(stats.ByCategory))
	}
}

func TestComputeSummaryNilResult(t *testing.T) {
	stats := ComputeSummary(nil)

	if stats.TotalIssues != 0 {
		t.Errorf("TotalIssues = %d, want 0", stats.TotalIssues)
	}
	if stats.PagesScanned != 0 {
		t.Errorf("PagesScanned = %d, want 0", stats.PagesScanned)
	}
}

func TestComputeSummaryMultipleIssues(t *testing.T) {
	result := &model.CrawlResult{
		Pages: []*model.Page{
			{URL: "https://example.com"},
			{URL: "https://example.com/about"},
			{URL: "https://example.com/contact"},
		},
		Issues: []model.Issue{
			{CheckName: "seo/missing-title", Severity: model.SeverityCritical, URL: "https://example.com"},
			{CheckName: "seo/meta-description", Severity: model.SeverityWarning, URL: "https://example.com"},
			{CheckName: "health/4xx", Severity: model.SeverityCritical, URL: "https://example.com/broken"},
			{CheckName: "perf/large-image", Severity: model.SeverityInfo, URL: "https://example.com/about"},
			{CheckName: "a11y/missing-alt", Severity: model.SeverityWarning, URL: "https://example.com"},
			{CheckName: "security/no-https", Severity: model.SeverityCritical, URL: "https://example.com/contact"},
		},
	}

	stats := ComputeSummary(result)

	if stats.TotalIssues != 6 {
		t.Errorf("TotalIssues = %d, want 6", stats.TotalIssues)
	}
	if stats.PagesScanned != 3 {
		t.Errorf("PagesScanned = %d, want 3", stats.PagesScanned)
	}

	// Check severity counts.
	wantSev := map[model.Severity]int{
		model.SeverityCritical: 3,
		model.SeverityWarning:  2,
		model.SeverityInfo:     1,
	}
	for sev, want := range wantSev {
		if got := stats.BySeverity[sev]; got != want {
			t.Errorf("BySeverity[%s] = %d, want %d", sev, got, want)
		}
	}

	// Check category counts.
	wantCat := map[string]int{
		"seo":      2,
		"health":   1,
		"perf":     1,
		"a11y":     1,
		"security": 1,
	}
	for cat, want := range wantCat {
		if got := stats.ByCategory[cat]; got != want {
			t.Errorf("ByCategory[%s] = %d, want %d", cat, got, want)
		}
	}

	// Check TopURLs ordering (example.com has 3 issues, should be first).
	if len(stats.TopURLs) != 4 {
		t.Fatalf("TopURLs length = %d, want 4", len(stats.TopURLs))
	}
	if stats.TopURLs[0].URL != "https://example.com" || stats.TopURLs[0].Count != 3 {
		t.Errorf("TopURLs[0] = %+v, want {URL: https://example.com, Count: 3}", stats.TopURLs[0])
	}
}

func TestComputeSummaryTopURLsCappedAt10(t *testing.T) {
	issues := make([]model.Issue, 0, 15)
	for i := range 15 {
		issues = append(issues, model.Issue{
			CheckName: "seo/test",
			Severity:  model.SeverityWarning,
			URL:       fmt.Sprintf("https://example.com/page-%02d", i),
		})
	}

	result := &model.CrawlResult{Issues: issues}
	stats := ComputeSummary(result)

	if len(stats.TopURLs) != 10 {
		t.Errorf("TopURLs length = %d, want 10", len(stats.TopURLs))
	}
}

func TestComputeSummaryCategoryWithoutSlash(t *testing.T) {
	result := &model.CrawlResult{
		Issues: []model.Issue{
			{CheckName: "orphan-page", Severity: model.SeverityInfo, URL: "https://example.com"},
		},
	}

	stats := ComputeSummary(result)

	if got := stats.ByCategory["orphan-page"]; got != 1 {
		t.Errorf("ByCategory[orphan-page] = %d, want 1", got)
	}
}
