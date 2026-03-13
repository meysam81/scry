package audit

import (
	"context"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func htmlPage(body string) *model.Page {
	return &model.Page{
		URL:         "https://example.com",
		StatusCode:  200,
		ContentType: "text/html; charset=utf-8",
		Body:        []byte(body),
	}
}

func TestSEOChecker_Check(t *testing.T) {
	checker := NewSEOChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantCheck  string
		wantSev    model.Severity
		wantIssue  bool
		wantSubstr string
	}{
		{
			name:      "missing title",
			html:      `<html><head></head><body></body></html>`,
			wantCheck: "seo/missing-title",
			wantSev:   model.SeverityCritical,
			wantIssue: true,
		},
		{
			name:      "good title",
			html:      `<html lang="en"><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"><link rel="canonical" href="/"><meta property="og:title" content="t"><meta property="og:description" content="d"><meta property="og:image" content="i"><meta name="twitter:card" content="summary"><meta name="viewport" content="width=device-width"></head><body><h1>Hello</h1></body></html>`,
			wantIssue: false,
		},
		{
			name:       "short title",
			html:       `<html><head><title>Short</title></head><body></body></html>`,
			wantCheck:  "seo/title-length",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "5 characters",
		},
		{
			name:       "long title",
			html:       `<html><head><title>` + strings.Repeat("x", 70) + `</title></head><body></body></html>`,
			wantCheck:  "seo/title-length",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "70 characters",
		},
		{
			name:      "missing meta description",
			html:      `<html><head><title>` + strings.Repeat("x", 40) + `</title></head><body></body></html>`,
			wantCheck: "seo/missing-meta-description",
			wantSev:   model.SeverityWarning,
			wantIssue: true,
		},
		{
			name:       "short meta description",
			html:       `<html><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="short"></head><body></body></html>`,
			wantCheck:  "seo/meta-description-length",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "5 characters",
		},
		{
			name:      "missing h1",
			html:      `<html><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"></head><body></body></html>`,
			wantCheck: "seo/missing-h1",
			wantSev:   model.SeverityWarning,
			wantIssue: true,
		},
		{
			name:       "multiple h1",
			html:       `<html><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"></head><body><h1>One</h1><h1>Two</h1></body></html>`,
			wantCheck:  "seo/multiple-h1",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "2 <h1> tags",
		},
		{
			name:      "missing canonical",
			html:      `<html><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"></head><body><h1>Hi</h1></body></html>`,
			wantCheck: "seo/missing-canonical",
			wantSev:   model.SeverityWarning,
			wantIssue: true,
		},
		{
			name:      "missing lang",
			html:      `<html><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"><link rel="canonical" href="/"></head><body><h1>Hi</h1></body></html>`,
			wantCheck: "seo/missing-lang",
			wantSev:   model.SeverityWarning,
			wantIssue: true,
		},
		{
			name:      "missing og:title",
			html:      `<html lang="en"><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"><link rel="canonical" href="/"></head><body><h1>Hi</h1></body></html>`,
			wantCheck: "seo/missing-og-title",
			wantSev:   model.SeverityInfo,
			wantIssue: true,
		},
		{
			name:       "multiple canonical links",
			html:       `<html lang="en"><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"><link rel="canonical" href="/a"><link rel="canonical" href="/b"><meta name="viewport" content="width=device-width"></head><body><h1>Hi</h1></body></html>`,
			wantCheck:  "seo/multiple-canonical",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "2 canonical links",
		},
		{
			name:       "meta robots conflict noindex with canonical",
			html:       `<html lang="en"><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"><meta name="robots" content="noindex"><link rel="canonical" href="/"><meta name="viewport" content="width=device-width"></head><body><h1>Hi</h1></body></html>`,
			wantCheck:  "seo/meta-robots-conflict",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "noindex",
		},
		{
			name:      "missing twitter card",
			html:      `<html lang="en"><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"><link rel="canonical" href="/"><meta property="og:title" content="t"><meta property="og:description" content="d"><meta property="og:image" content="i"><meta name="viewport" content="width=device-width"></head><body><h1>Hi</h1></body></html>`,
			wantCheck: "seo/missing-twitter-card",
			wantSev:   model.SeverityInfo,
			wantIssue: true,
		},
		{
			name:      "twitter card via property attr",
			html:      `<html lang="en"><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"><link rel="canonical" href="/"><meta property="og:title" content="t"><meta property="og:description" content="d"><meta property="og:image" content="i"><meta property="twitter:card" content="summary"><meta name="viewport" content="width=device-width"></head><body><h1>Hi</h1></body></html>`,
			wantIssue: false,
		},
		{
			name:      "missing viewport",
			html:      `<html lang="en"><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"><link rel="canonical" href="/"><meta property="og:title" content="t"><meta property="og:description" content="d"><meta property="og:image" content="i"><meta name="twitter:card" content="summary"></head><body><h1>Hi</h1></body></html>`,
			wantCheck: "seo/missing-viewport",
			wantSev:   model.SeverityCritical,
			wantIssue: true,
		},
		{
			name:      "non-html page skipped",
			html:      "",
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			if tt.name == "non-html page skipped" {
				page.ContentType = "application/json"
			}

			issues := checker.Check(ctx, page)

			if !tt.wantIssue {
				if len(issues) > 0 {
					t.Fatalf("expected no issues, got %d: %+v", len(issues), issues)
				}
				return
			}

			found := false
			for _, iss := range issues {
				if iss.CheckName == tt.wantCheck {
					found = true
					if iss.Severity != tt.wantSev {
						t.Errorf("check %s: expected severity %s, got %s", tt.wantCheck, tt.wantSev, iss.Severity)
					}
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("check %s: expected message to contain %q, got %q", tt.wantCheck, tt.wantSubstr, iss.Message)
					}
				}
			}
			if !found {
				t.Errorf("expected issue %s not found in %+v", tt.wantCheck, issues)
			}
		})
	}
}

func TestSEOChecker_CheckSite_NoindexInSitemap(t *testing.T) {
	checker := NewSEOChecker()
	ctx := context.Background()

	tests := []struct {
		name      string
		pages     []*model.Page
		wantIssue bool
	}{
		{
			name: "noindex page in sitemap",
			pages: []*model.Page{
				{
					URL:         "https://example.com/hidden",
					ContentType: "text/html",
					InSitemap:   true,
					Body:        []byte(`<html><head><meta name="robots" content="noindex"></head><body></body></html>`),
				},
			},
			wantIssue: true,
		},
		{
			name: "noindex page not in sitemap",
			pages: []*model.Page{
				{
					URL:         "https://example.com/hidden",
					ContentType: "text/html",
					InSitemap:   false,
					Body:        []byte(`<html><head><meta name="robots" content="noindex"></head><body></body></html>`),
				},
			},
			wantIssue: false,
		},
		{
			name: "indexable page in sitemap",
			pages: []*model.Page{
				{
					URL:         "https://example.com/visible",
					ContentType: "text/html",
					InSitemap:   true,
					Body:        []byte(`<html><head></head><body></body></html>`),
				},
			},
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checker.CheckSite(ctx, tt.pages)
			found := false
			for _, iss := range issues {
				if iss.CheckName == "seo/noindex-in-sitemap" {
					found = true
				}
			}
			if tt.wantIssue && !found {
				t.Error("expected seo/noindex-in-sitemap issue, got none")
			}
			if !tt.wantIssue && found {
				t.Error("did not expect seo/noindex-in-sitemap issue")
			}
		})
	}
}

func TestSEOChecker_URLIssues(t *testing.T) {
	checker := NewSEOChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		url        string
		wantIssue  bool
		wantSubstr string
	}{
		{
			name:      "clean short URL",
			url:       "https://example.com/about",
			wantIssue: false,
		},
		{
			name:       "URL with underscore in path",
			url:        "https://example.com/my_page",
			wantIssue:  true,
			wantSubstr: "underscores",
		},
		{
			name:       "URL with uppercase in path",
			url:        "https://example.com/About",
			wantIssue:  true,
			wantSubstr: "uppercase",
		},
		{
			name:       "long URL over 100 chars",
			url:        "https://example.com/" + strings.Repeat("a", 90),
			wantIssue:  true,
			wantSubstr: ">100",
		},
		{
			name:       "URL with multiple issues",
			url:        "https://example.com/" + strings.Repeat("A_", 50),
			wantIssue:  true,
			wantSubstr: "underscores",
		},
		{
			name:      "uppercase in query params only is fine",
			url:       "https://example.com/page?Key=Value",
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &model.Page{
				URL:         tt.url,
				StatusCode:  200,
				ContentType: "text/html; charset=utf-8",
				Body:        []byte(`<html lang="en"><head><title>` + strings.Repeat("x", 40) + `</title><meta name="description" content="` + strings.Repeat("y", 100) + `"><link rel="canonical" href="/"><meta property="og:title" content="t"><meta property="og:description" content="d"><meta property="og:image" content="i"><meta name="twitter:card" content="summary"><meta name="viewport" content="width=device-width"></head><body><h1>Hi</h1></body></html>`),
			}
			issues := checker.Check(ctx, page)
			found := false
			for _, iss := range issues {
				if iss.CheckName == "seo/url-issues" {
					found = true
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message to contain %q, got %q", tt.wantSubstr, iss.Message)
					}
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected seo/url-issues issue, got none; issues: %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect seo/url-issues issue, got one")
			}
		})
	}
}

func TestSEOChecker_CheckSite_DuplicateTitles(t *testing.T) {
	checker := NewSEOChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		pages      []*model.Page
		wantIssue  bool
		wantSubstr string
	}{
		{
			name: "two pages same title",
			pages: []*model.Page{
				{
					URL:         "https://example.com/a",
					ContentType: "text/html",
					Body:        []byte(`<html><head><title>Same Title</title></head><body></body></html>`),
				},
				{
					URL:         "https://example.com/b",
					ContentType: "text/html",
					Body:        []byte(`<html><head><title>Same Title</title></head><body></body></html>`),
				},
			},
			wantIssue:  true,
			wantSubstr: "2 pages",
		},
		{
			name: "three pages same title",
			pages: []*model.Page{
				{
					URL:         "https://example.com/a",
					ContentType: "text/html",
					Body:        []byte(`<html><head><title>Shared</title></head><body></body></html>`),
				},
				{
					URL:         "https://example.com/b",
					ContentType: "text/html",
					Body:        []byte(`<html><head><title>Shared</title></head><body></body></html>`),
				},
				{
					URL:         "https://example.com/c",
					ContentType: "text/html",
					Body:        []byte(`<html><head><title>Shared</title></head><body></body></html>`),
				},
			},
			wantIssue:  true,
			wantSubstr: "3 pages",
		},
		{
			name: "unique titles no issue",
			pages: []*model.Page{
				{
					URL:         "https://example.com/a",
					ContentType: "text/html",
					Body:        []byte(`<html><head><title>Title A</title></head><body></body></html>`),
				},
				{
					URL:         "https://example.com/b",
					ContentType: "text/html",
					Body:        []byte(`<html><head><title>Title B</title></head><body></body></html>`),
				},
			},
			wantIssue: false,
		},
		{
			name: "empty titles not flagged",
			pages: []*model.Page{
				{
					URL:         "https://example.com/a",
					ContentType: "text/html",
					Body:        []byte(`<html><head><title></title></head><body></body></html>`),
				},
				{
					URL:         "https://example.com/b",
					ContentType: "text/html",
					Body:        []byte(`<html><head><title></title></head><body></body></html>`),
				},
			},
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checker.CheckSite(ctx, tt.pages)
			found := false
			for _, iss := range issues {
				if iss.CheckName == "seo/duplicate-titles" {
					found = true
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message to contain %q, got %q", tt.wantSubstr, iss.Message)
					}
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected seo/duplicate-titles issue, got none; issues: %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect seo/duplicate-titles issue")
			}
		})
	}
}
