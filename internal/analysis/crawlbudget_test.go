package analysis

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestCrawlBudgetAnalyzer_Name(t *testing.T) {
	c := NewCrawlBudgetAnalyzer()
	if c.Name() != "crawl-budget" {
		t.Fatalf("expected name %q, got %q", "crawl-budget", c.Name())
	}
}

func TestCrawlBudgetAnalyzer_Check_ReturnsNil(t *testing.T) {
	c := NewCrawlBudgetAnalyzer()
	page := &model.Page{URL: "https://example.com"}
	issues := c.Check(context.Background(), page)
	if issues != nil {
		t.Fatalf("expected nil from Check, got %+v", issues)
	}
}

func TestCrawlBudgetAnalyzer_CheckSite_EmptyPages(t *testing.T) {
	analyzer := NewCrawlBudgetAnalyzer()
	issues := analyzer.CheckSite(context.Background(), nil)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for nil pages, got %+v", issues)
	}

	issues = analyzer.CheckSite(context.Background(), []*model.Page{})
	if len(issues) != 0 {
		t.Fatalf("expected no issues for empty pages, got %+v", issues)
	}
}

func TestCrawlBudgetAnalyzer_CheckSite_DeepPages(t *testing.T) {
	analyzer := NewCrawlBudgetAnalyzer()
	ctx := context.Background()

	tests := []struct {
		name       string
		pages      []*model.Page
		wantIssue  bool
		wantSubstr string
	}{
		{
			name: "all pages shallow - no issue",
			pages: []*model.Page{
				{URL: "https://example.com", Depth: 0},
				{URL: "https://example.com/a", Depth: 1},
				{URL: "https://example.com/b", Depth: 2},
				{URL: "https://example.com/c", Depth: 3},
			},
			wantIssue: false,
		},
		{
			name: "exactly 10% deep - no issue (not exceeding)",
			pages: func() []*model.Page {
				// 10 pages: 9 at depth <= 3, 1 at depth > 3 = 10%
				pages := make([]*model.Page, 10)
				for i := range 9 {
					pages[i] = &model.Page{URL: fmt.Sprintf("https://example.com/%d", i), Depth: 1}
				}
				pages[9] = &model.Page{URL: "https://example.com/deep/9", Depth: 5}
				return pages
			}(),
			wantIssue: false,
		},
		{
			name: "more than 10% deep - issue",
			pages: func() []*model.Page {
				// 10 pages: 8 at depth <= 3, 2 at depth > 3 = 20%
				pages := make([]*model.Page, 10)
				for i := range 8 {
					pages[i] = &model.Page{URL: fmt.Sprintf("https://example.com/%d", i), Depth: 1}
				}
				for i := 8; i < 10; i++ {
					pages[i] = &model.Page{URL: fmt.Sprintf("https://example.com/deep/%d", i), Depth: 5}
				}
				return pages
			}(),
			wantIssue:  true,
			wantSubstr: "20%",
		},
		{
			name: "depth exactly 4 counts as deep",
			pages: func() []*model.Page {
				// 5 pages: 3 at depth <= 3, 2 at depth 4 = 40%
				return []*model.Page{
					{URL: "https://example.com", Depth: 0},
					{URL: "https://example.com/a", Depth: 1},
					{URL: "https://example.com/b", Depth: 2},
					{URL: "https://example.com/c", Depth: 4},
					{URL: "https://example.com/d", Depth: 4},
				}
			}(),
			wantIssue:  true,
			wantSubstr: "40%",
		},
		{
			name: "depth exactly 3 does not count as deep",
			pages: func() []*model.Page {
				return []*model.Page{
					{URL: "https://example.com/a", Depth: 3},
					{URL: "https://example.com/b", Depth: 3},
					{URL: "https://example.com/c", Depth: 3},
				}
			}(),
			wantIssue: false,
		},
		{
			name: "all deep - 100%",
			pages: []*model.Page{
				{URL: "https://example.com/a", Depth: 5},
				{URL: "https://example.com/b", Depth: 6},
				{URL: "https://example.com/c", Depth: 10},
			},
			wantIssue:  true,
			wantSubstr: "3 of 3",
		},
		{
			name: "single page at depth > 3 - 100% triggers",
			pages: []*model.Page{
				{URL: "https://example.com/deep", Depth: 4},
			},
			wantIssue:  true,
			wantSubstr: "1 of 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := analyzer.CheckSite(ctx, tt.pages)
			found := false
			for _, iss := range issues {
				if iss.CheckName == "crawl-budget/deep-pages" {
					found = true
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
					}
					if iss.Severity != model.SeverityWarning {
						t.Errorf("expected severity %s, got %s", model.SeverityWarning, iss.Severity)
					}
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected crawl-budget/deep-pages issue, got none; issues=%+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect crawl-budget/deep-pages issue")
			}
		})
	}
}

func TestCrawlBudgetAnalyzer_CheckSite_ParameterURLs(t *testing.T) {
	analyzer := NewCrawlBudgetAnalyzer()
	ctx := context.Background()

	tests := []struct {
		name       string
		pages      []*model.Page
		wantIssue  bool
		wantSubstr string
	}{
		{
			name: "no parameter URLs - no issue",
			pages: []*model.Page{
				{URL: "https://example.com"},
				{URL: "https://example.com/a"},
				{URL: "https://example.com/b"},
			},
			wantIssue: false,
		},
		{
			name: "single parameter URL - issue",
			pages: []*model.Page{
				{URL: "https://example.com"},
				{URL: "https://example.com/search?q=foo"},
			},
			wantIssue:  true,
			wantSubstr: "1 URLs contain query parameters",
		},
		{
			name: "multiple parameter URLs - issue with count",
			pages: []*model.Page{
				{URL: "https://example.com"},
				{URL: "https://example.com/search?q=foo"},
				{URL: "https://example.com/filter?type=bar"},
				{URL: "https://example.com/page?id=123&sort=asc"},
			},
			wantIssue:  true,
			wantSubstr: "3 URLs contain query parameters",
		},
		{
			name: "all parameter URLs",
			pages: []*model.Page{
				{URL: "https://example.com/search?q=foo"},
				{URL: "https://example.com/filter?type=bar"},
			},
			wantIssue:  true,
			wantSubstr: "2 URLs contain query parameters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := analyzer.CheckSite(ctx, tt.pages)
			found := false
			for _, iss := range issues {
				if iss.CheckName == "crawl-budget/parameter-urls" {
					found = true
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
					}
					if iss.Severity != model.SeverityInfo {
						t.Errorf("expected severity %s, got %s", model.SeverityInfo, iss.Severity)
					}
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected crawl-budget/parameter-urls issue, got none; issues=%+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect crawl-budget/parameter-urls issue")
			}
		})
	}
}

func TestCrawlBudgetAnalyzer_CheckSite_LowInternalLinks(t *testing.T) {
	analyzer := NewCrawlBudgetAnalyzer()
	ctx := context.Background()

	tests := []struct {
		name       string
		pages      []*model.Page
		wantIssue  bool
		wantSubstr string
	}{
		{
			name: "no links at all - issue",
			pages: []*model.Page{
				{URL: "https://example.com"},
				{URL: "https://example.com/a"},
			},
			wantIssue:  true,
			wantSubstr: "0.0",
		},
		{
			name: "average exactly 3.0 - no issue (not less than)",
			pages: []*model.Page{
				{URL: "https://example.com", Links: []string{"/a", "/b", "/c"}},
				{URL: "https://example.com/a", Links: []string{"/b", "/c", "/d"}},
			},
			wantIssue: false,
		},
		{
			name: "average below 3.0 - issue",
			pages: []*model.Page{
				{URL: "https://example.com", Links: []string{"/a", "/b"}},
				{URL: "https://example.com/a", Links: []string{"/b"}},
			},
			wantIssue:  true,
			wantSubstr: "1.5",
		},
		{
			name: "average above 3.0 - no issue",
			pages: []*model.Page{
				{URL: "https://example.com", Links: []string{"/a", "/b", "/c", "/d"}},
				{URL: "https://example.com/a", Links: []string{"/b", "/c", "/d", "/e"}},
			},
			wantIssue: false,
		},
		{
			name: "mixed links some pages with none",
			pages: []*model.Page{
				{URL: "https://example.com", Links: []string{"/a", "/b", "/c", "/d", "/e"}},
				{URL: "https://example.com/a", Links: nil},
				{URL: "https://example.com/b", Links: []string{"/c"}},
			},
			// total links: 5 + 0 + 1 = 6, avg = 6/3 = 2.0
			wantIssue:  true,
			wantSubstr: "2.0",
		},
		{
			name: "single page with many links - no issue",
			pages: []*model.Page{
				{URL: "https://example.com", Links: []string{"/a", "/b", "/c", "/d", "/e"}},
			},
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := analyzer.CheckSite(ctx, tt.pages)
			found := false
			for _, iss := range issues {
				if iss.CheckName == "crawl-budget/low-internal-links" {
					found = true
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
					}
					if iss.Severity != model.SeverityInfo {
						t.Errorf("expected severity %s, got %s", model.SeverityInfo, iss.Severity)
					}
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected crawl-budget/low-internal-links issue, got none; issues=%+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect crawl-budget/low-internal-links issue")
			}
		})
	}
}

func TestCrawlBudgetAnalyzer_CheckSite_LargeSitemapGap(t *testing.T) {
	analyzer := NewCrawlBudgetAnalyzer()
	ctx := context.Background()

	tests := []struct {
		name       string
		pages      []*model.Page
		wantIssue  bool
		wantSubstr string
	}{
		{
			name: "all pages in sitemap - no issue",
			pages: []*model.Page{
				{URL: "https://example.com", InSitemap: true},
				{URL: "https://example.com/a", InSitemap: true},
				{URL: "https://example.com/b", InSitemap: true},
			},
			wantIssue: false,
		},
		{
			name: "exactly 20% not in sitemap - no issue (not exceeding)",
			pages: func() []*model.Page {
				// 10 pages: 8 in sitemap, 2 not = 20%
				pages := make([]*model.Page, 10)
				for i := range 8 {
					pages[i] = &model.Page{
						URL:       fmt.Sprintf("https://example.com/%d", i),
						InSitemap: true,
					}
				}
				for i := 8; i < 10; i++ {
					pages[i] = &model.Page{
						URL:       fmt.Sprintf("https://example.com/orphan/%d", i),
						InSitemap: false,
					}
				}
				return pages
			}(),
			wantIssue: false,
		},
		{
			name: "more than 20% not in sitemap - issue",
			pages: func() []*model.Page {
				// 10 pages: 7 in sitemap, 3 not = 30%
				pages := make([]*model.Page, 10)
				for i := range 7 {
					pages[i] = &model.Page{
						URL:       fmt.Sprintf("https://example.com/%d", i),
						InSitemap: true,
					}
				}
				for i := 7; i < 10; i++ {
					pages[i] = &model.Page{
						URL:       fmt.Sprintf("https://example.com/orphan/%d", i),
						InSitemap: false,
					}
				}
				return pages
			}(),
			wantIssue:  true,
			wantSubstr: "30%",
		},
		{
			name: "none in sitemap - issue",
			pages: []*model.Page{
				{URL: "https://example.com", InSitemap: false},
				{URL: "https://example.com/a", InSitemap: false},
				{URL: "https://example.com/b", InSitemap: false},
			},
			wantIssue:  true,
			wantSubstr: "3 of 3",
		},
		{
			name: "single page not in sitemap - 100% gap",
			pages: []*model.Page{
				{URL: "https://example.com/orphan", InSitemap: false},
			},
			wantIssue:  true,
			wantSubstr: "1 of 1",
		},
		{
			name: "single page in sitemap - no issue",
			pages: []*model.Page{
				{URL: "https://example.com", InSitemap: true},
			},
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := analyzer.CheckSite(ctx, tt.pages)
			found := false
			for _, iss := range issues {
				if iss.CheckName == "crawl-budget/large-sitemap-gap" {
					found = true
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
					}
					if iss.Severity != model.SeverityWarning {
						t.Errorf("expected severity %s, got %s", model.SeverityWarning, iss.Severity)
					}
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected crawl-budget/large-sitemap-gap issue, got none; issues=%+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect crawl-budget/large-sitemap-gap issue")
			}
		})
	}
}

// TestCrawlBudgetAnalyzer_CheckSite_NoFalseCheckNames verifies that no
// removed check names (e.g. deep-structure, thin-depth-level) are emitted.
func TestCrawlBudgetAnalyzer_CheckSite_NoRemovedCheckNames(t *testing.T) {
	analyzer := NewCrawlBudgetAnalyzer()
	ctx := context.Background()

	// Build pages that would have triggered old checks.
	pages := []*model.Page{
		{URL: "https://example.com", Depth: 0, Links: []string{"/a"}},
		{URL: "https://example.com/a", Depth: 1, Links: []string{"/b"}},
		{URL: "https://example.com/b", Depth: 2},
		{URL: "https://example.com/c?q=1", Depth: 3},
		{URL: "https://example.com/d", Depth: 5},
		{URL: "https://example.com/e", Depth: 6},
	}

	removedChecks := []string{
		"crawl-budget/deep-structure",
		"crawl-budget/thin-depth-level",
	}

	issues := analyzer.CheckSite(ctx, pages)
	for _, iss := range issues {
		for _, removed := range removedChecks {
			if iss.CheckName == removed {
				t.Errorf("unexpected removed check name %q in issues", removed)
			}
		}
	}
}

// TestCrawlBudgetAnalyzer_CheckSite_MultipleIssuesCombined verifies that
// a single page set can trigger multiple checks simultaneously.
func TestCrawlBudgetAnalyzer_CheckSite_MultipleIssuesCombined(t *testing.T) {
	analyzer := NewCrawlBudgetAnalyzer()
	ctx := context.Background()

	// Pages that trigger all four checks:
	// - deep pages: 3 of 5 = 60% at depth > 3
	// - parameter URLs: 2 URLs with ?
	// - low internal links: avg = (0+0+1+0+0)/5 = 0.2
	// - sitemap gap: 4 of 5 = 80% not in sitemap
	pages := []*model.Page{
		{URL: "https://example.com?page=1", Depth: 4, InSitemap: false},
		{URL: "https://example.com/a?sort=asc", Depth: 5, InSitemap: false},
		{URL: "https://example.com/b", Depth: 6, Links: []string{"/c"}, InSitemap: true},
		{URL: "https://example.com/c", Depth: 1, InSitemap: false},
		{URL: "https://example.com/d", Depth: 2, InSitemap: false},
	}

	issues := analyzer.CheckSite(ctx, pages)

	expectedChecks := map[string]bool{
		"crawl-budget/deep-pages":         false,
		"crawl-budget/parameter-urls":     false,
		"crawl-budget/low-internal-links": false,
		"crawl-budget/large-sitemap-gap":  false,
	}

	for _, iss := range issues {
		if _, ok := expectedChecks[iss.CheckName]; ok {
			expectedChecks[iss.CheckName] = true
		}
	}

	for check, found := range expectedChecks {
		if !found {
			t.Errorf("expected check %q to be triggered, but it was not; issues=%+v", check, issues)
		}
	}
}

// TestCrawlBudgetAnalyzer_CheckSite_CleanSite verifies that a well-structured
// site produces no issues at all.
func TestCrawlBudgetAnalyzer_CheckSite_CleanSite(t *testing.T) {
	analyzer := NewCrawlBudgetAnalyzer()
	ctx := context.Background()

	pages := []*model.Page{
		{URL: "https://example.com", Depth: 0, Links: []string{"/a", "/b", "/c"}, InSitemap: true},
		{URL: "https://example.com/a", Depth: 1, Links: []string{"/", "/b", "/c", "/d"}, InSitemap: true},
		{URL: "https://example.com/b", Depth: 1, Links: []string{"/", "/a", "/c"}, InSitemap: true},
		{URL: "https://example.com/c", Depth: 2, Links: []string{"/", "/a", "/b"}, InSitemap: true},
		{URL: "https://example.com/d", Depth: 2, Links: []string{"/", "/a", "/b", "/c"}, InSitemap: true},
	}

	issues := analyzer.CheckSite(ctx, pages)
	if len(issues) != 0 {
		t.Errorf("expected no issues for clean site, got %+v", issues)
	}
}
