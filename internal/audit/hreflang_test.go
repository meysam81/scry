package audit

import (
	"context"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func hreflangPage(url, body string) *model.Page {
	return &model.Page{
		URL:         url,
		StatusCode:  200,
		ContentType: "text/html; charset=utf-8",
		Body:        []byte(body),
	}
}

func TestHreflangChecker_Name(t *testing.T) {
	checker := NewHreflangChecker()
	if checker.Name() != "hreflang" {
		t.Errorf("expected name 'hreflang', got %q", checker.Name())
	}
}

func TestHreflangChecker_Check(t *testing.T) {
	checker := NewHreflangChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		url        string
		html       string
		ct         string
		wantCheck  string
		wantSev    model.Severity
		wantIssue  bool
		wantSubstr string
	}{
		{
			name:      "valid hreflang with x-default - no issue",
			url:       "https://example.com",
			html:      `<html><head><link rel="alternate" hreflang="en" href="https://example.com"><link rel="alternate" hreflang="fr" href="https://example.com/fr"><link rel="alternate" hreflang="x-default" href="https://example.com"></head><body></body></html>`,
			ct:        "text/html",
			wantIssue: false,
		},
		{
			name:       "invalid language code - too long",
			url:        "https://example.com",
			html:       `<html><head><link rel="alternate" hreflang="english" href="https://example.com"><link rel="alternate" hreflang="x-default" href="https://example.com"></head><body></body></html>`,
			ct:         "text/html",
			wantCheck:  "hreflang/invalid-language-code",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "english",
		},
		{
			name:       "invalid language code with numbers",
			url:        "https://example.com",
			html:       `<html><head><link rel="alternate" hreflang="en123" href="https://example.com"><link rel="alternate" hreflang="x-default" href="https://example.com"></head><body></body></html>`,
			ct:         "text/html",
			wantCheck:  "hreflang/invalid-language-code",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "en123",
		},
		{
			name:      "missing x-default",
			url:       "https://example.com",
			html:      `<html><head><link rel="alternate" hreflang="en" href="https://example.com"><link rel="alternate" hreflang="fr" href="https://example.com/fr"></head><body></body></html>`,
			ct:        "text/html",
			wantCheck: "hreflang/missing-x-default",
			wantSev:   model.SeverityInfo,
			wantIssue: true,
		},
		{
			name:      "no hreflang annotations - no issue",
			url:       "https://example.com",
			html:      `<html><head><link rel="stylesheet" href="/style.css"></head><body></body></html>`,
			ct:        "text/html",
			wantIssue: false,
		},
		{
			name:      "non-html page skipped",
			url:       "https://example.com/api",
			html:      `{}`,
			ct:        "application/json",
			wantIssue: false,
		},
		{
			name:      "valid en-US subtag",
			url:       "https://example.com",
			html:      `<html><head><link rel="alternate" hreflang="en-US" href="https://example.com"><link rel="alternate" hreflang="x-default" href="https://example.com"></head><body></body></html>`,
			ct:        "text/html",
			wantIssue: false,
		},
		{
			name:      "valid zh-Hans-CN subtag",
			url:       "https://example.com",
			html:      `<html><head><link rel="alternate" hreflang="zh-Hans-CN" href="https://example.com"><link rel="alternate" hreflang="x-default" href="https://example.com"></head><body></body></html>`,
			ct:        "text/html",
			wantIssue: false,
		},
		{
			name:      "valid three-letter primary tag",
			url:       "https://example.com",
			html:      `<html><head><link rel="alternate" hreflang="haw" href="https://example.com"><link rel="alternate" hreflang="x-default" href="https://example.com"></head><body></body></html>`,
			ct:        "text/html",
			wantIssue: false,
		},
		{
			name:       "uppercase language code is invalid",
			url:        "https://example.com",
			html:       `<html><head><link rel="alternate" hreflang="EN" href="https://example.com"><link rel="alternate" hreflang="x-default" href="https://example.com"></head><body></body></html>`,
			ct:         "text/html",
			wantCheck:  "hreflang/invalid-language-code",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "EN",
		},
		{
			name:      "empty hreflang value is not extracted",
			url:       "https://example.com",
			html:      `<html><head><link rel="alternate" hreflang="" href="https://example.com"><link rel="alternate" hreflang="en" href="https://example.com"></head><body></body></html>`,
			ct:        "text/html",
			wantCheck: "hreflang/missing-x-default",
			wantSev:   model.SeverityInfo,
			wantIssue: true,
		},
		{
			name:       "single letter primary tag is invalid",
			url:        "https://example.com",
			html:       `<html><head><link rel="alternate" hreflang="e" href="https://example.com"><link rel="alternate" hreflang="x-default" href="https://example.com"></head><body></body></html>`,
			ct:         "text/html",
			wantCheck:  "hreflang/invalid-language-code",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: `"e"`,
		},
		{
			name:       "underscore separator is invalid",
			url:        "https://example.com",
			html:       `<html><head><link rel="alternate" hreflang="en_US" href="https://example.com"><link rel="alternate" hreflang="x-default" href="https://example.com"></head><body></body></html>`,
			ct:         "text/html",
			wantCheck:  "hreflang/invalid-language-code",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "en_US",
		},
		{
			name:       "subtag too long is invalid",
			url:        "https://example.com",
			html:       `<html><head><link rel="alternate" hreflang="en-USAAA" href="https://example.com"><link rel="alternate" hreflang="x-default" href="https://example.com"></head><body></body></html>`,
			ct:         "text/html",
			wantCheck:  "hreflang/invalid-language-code",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "en-USAAA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &model.Page{
				URL:         tt.url,
				StatusCode:  200,
				ContentType: tt.ct,
				Body:        []byte(tt.html),
			}

			issues := checker.Check(ctx, page)

			if !tt.wantIssue {
				for _, iss := range issues {
					if tt.wantCheck != "" && iss.CheckName == tt.wantCheck {
						t.Fatalf("did not expect issue %s, got %+v", tt.wantCheck, iss)
					}
				}
				if tt.wantCheck == "" && len(issues) > 0 {
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
						t.Errorf("check %s: expected message containing %q, got %q", tt.wantCheck, tt.wantSubstr, iss.Message)
					}
				}
			}
			if !found {
				t.Errorf("expected issue %s not found in %+v", tt.wantCheck, issues)
			}
		})
	}
}

func TestHreflangChecker_CheckSite_MissingReturnLink(t *testing.T) {
	checker := NewHreflangChecker()
	ctx := context.Background()

	tests := []struct {
		name      string
		pages     []*model.Page
		wantIssue bool
		wantURL   string
	}{
		{
			name: "mutual return links - no issue",
			pages: []*model.Page{
				hreflangPage("https://example.com/en", `<html><head><link rel="alternate" hreflang="en" href="https://example.com/en"><link rel="alternate" hreflang="fr" href="https://example.com/fr"></head><body></body></html>`),
				hreflangPage("https://example.com/fr", `<html><head><link rel="alternate" hreflang="en" href="https://example.com/en"><link rel="alternate" hreflang="fr" href="https://example.com/fr"></head><body></body></html>`),
			},
			wantIssue: false,
		},
		{
			name: "page A links to B but B does not link back",
			pages: []*model.Page{
				hreflangPage("https://example.com/en", `<html><head><link rel="alternate" hreflang="en" href="https://example.com/en"><link rel="alternate" hreflang="fr" href="https://example.com/fr"></head><body></body></html>`),
				hreflangPage("https://example.com/fr", `<html><head><link rel="alternate" hreflang="fr" href="https://example.com/fr"></head><body></body></html>`),
			},
			wantIssue: true,
			wantURL:   "https://example.com/en",
		},
		{
			name: "target page not in crawl set - no issue",
			pages: []*model.Page{
				hreflangPage("https://example.com/en", `<html><head><link rel="alternate" hreflang="en" href="https://example.com/en"><link rel="alternate" hreflang="de" href="https://example.com/de"></head><body></body></html>`),
			},
			wantIssue: false,
		},
		{
			name: "self-referencing only - no return link issue",
			pages: []*model.Page{
				hreflangPage("https://example.com/en", `<html><head><link rel="alternate" hreflang="en" href="https://example.com/en"><link rel="alternate" hreflang="x-default" href="https://example.com/en"></head><body></body></html>`),
			},
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checker.CheckSite(ctx, tt.pages)
			found := false
			for _, iss := range issues {
				if iss.CheckName == "hreflang/missing-return-link" {
					found = true
					if tt.wantURL != "" && iss.URL != tt.wantURL {
						t.Errorf("expected URL %s, got %s", tt.wantURL, iss.URL)
					}
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected hreflang/missing-return-link issue, got none; all: %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect hreflang/missing-return-link issue; all: %+v", issues)
			}
		})
	}
}

func TestHreflangChecker_CheckSite_SelfReferenceMissing(t *testing.T) {
	checker := NewHreflangChecker()
	ctx := context.Background()

	tests := []struct {
		name      string
		pages     []*model.Page
		wantIssue bool
	}{
		{
			name: "self-reference present - no issue",
			pages: []*model.Page{
				hreflangPage("https://example.com/en", `<html><head><link rel="alternate" hreflang="en" href="https://example.com/en"><link rel="alternate" hreflang="fr" href="https://example.com/fr"></head><body></body></html>`),
			},
			wantIssue: false,
		},
		{
			name: "self-reference missing",
			pages: []*model.Page{
				hreflangPage("https://example.com/en", `<html><head><link rel="alternate" hreflang="fr" href="https://example.com/fr"></head><body></body></html>`),
			},
			wantIssue: true,
		},
		{
			name: "self-reference with trailing slash normalization",
			pages: []*model.Page{
				hreflangPage("https://example.com/en", `<html><head><link rel="alternate" hreflang="en" href="https://example.com/en/"><link rel="alternate" hreflang="fr" href="https://example.com/fr"></head><body></body></html>`),
			},
			wantIssue: false,
		},
		{
			name: "no hreflang annotations - no issue",
			pages: []*model.Page{
				hreflangPage("https://example.com/en", `<html><head></head><body></body></html>`),
			},
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checker.CheckSite(ctx, tt.pages)
			found := false
			for _, iss := range issues {
				if iss.CheckName == "hreflang/self-reference-missing" {
					found = true
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected hreflang/self-reference-missing issue, got none; all: %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect hreflang/self-reference-missing issue; all: %+v", issues)
			}
		})
	}
}

func TestHreflangChecker_CheckSite_ReturnLinkWithTrailingSlash(t *testing.T) {
	checker := NewHreflangChecker()
	ctx := context.Background()

	pages := []*model.Page{
		hreflangPage("https://example.com/en", `<html><head><link rel="alternate" hreflang="en" href="https://example.com/en"><link rel="alternate" hreflang="fr" href="https://example.com/fr/"></head><body></body></html>`),
		hreflangPage("https://example.com/fr", `<html><head><link rel="alternate" hreflang="en" href="https://example.com/en/"><link rel="alternate" hreflang="fr" href="https://example.com/fr"></head><body></body></html>`),
	}

	issues := checker.CheckSite(ctx, pages)
	for _, iss := range issues {
		if iss.CheckName == "hreflang/missing-return-link" {
			t.Errorf("should not flag missing return link when trailing slash differs; got %+v", iss)
		}
	}
}

func TestHreflangChecker_SkipsNonHTML(t *testing.T) {
	checker := NewHreflangChecker()
	ctx := context.Background()

	page := &model.Page{
		URL:         "https://example.com/data.json",
		ContentType: "application/json",
		Body:        []byte(`{}`),
	}

	issues := checker.Check(ctx, page)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for non-HTML, got %+v", issues)
	}
}

func TestBCP47Regex(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"en", true},
		{"fr", true},
		{"zh", true},
		{"haw", true},
		{"en-US", true},
		{"en-GB", true},
		{"zh-Hans", true},
		{"zh-Hans-CN", true},
		{"x-default", true},
		{"EN", false},
		{"English", false},
		{"e", false},
		{"en-", false},
		{"123", false},
		{"en_US", false},
		{"", false},
		{"en-USAAA", false},
	}

	for _, tt := range tests {
		got := bcp47Re.MatchString(tt.input)
		if got != tt.valid {
			t.Errorf("bcp47Re.MatchString(%q) = %v, want %v", tt.input, got, tt.valid)
		}
	}
}
