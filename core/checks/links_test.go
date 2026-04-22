package checks

import (
	"context"
	"strings"
	"testing"

	"github.com/meysam81/scry/core/model"
)

func TestLinkChecker_Check_NonHTML_ReturnsNil(t *testing.T) {
	checker := NewLinkChecker()
	page := &model.Page{
		URL:         "https://example.com/api",
		StatusCode:  200,
		ContentType: "application/json",
		Body:        []byte(`{"ok":true}`),
	}
	issues := checker.Check(context.Background(), page)
	if len(issues) > 0 {
		t.Fatalf("expected no issues from Check for non-HTML, got %+v", issues)
	}
}

func TestLinkChecker_ExcessiveLinks(t *testing.T) {
	checker := NewLinkChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		linkCount  int
		wantIssue  bool
		wantSubstr string
	}{
		{
			name:      "50 links is fine",
			linkCount: 50,
			wantIssue: false,
		},
		{
			name:      "100 links is fine",
			linkCount: 100,
			wantIssue: false,
		},
		{
			name:       "101 links is excessive",
			linkCount:  101,
			wantIssue:  true,
			wantSubstr: "101 links",
		},
		{
			name:       "200 links is excessive",
			linkCount:  200,
			wantIssue:  true,
			wantSubstr: "200 links",
		},
		{
			name:      "0 links is fine",
			linkCount: 0,
			wantIssue: false,
		},
		{
			name:      "anchors without href not counted",
			linkCount: -1, // sentinel: use custom HTML
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body string
			if tt.linkCount == -1 {
				// 150 anchors without href should not trigger the check.
				body = "<html><body>"
				for i := 0; i < 150; i++ {
					body += "<a name=\"anchor\">Anchor</a>"
				}
				body += "</body></html>"
			} else {
				body = "<html><body>"
				for i := 0; i < tt.linkCount; i++ {
					body += "<a href=\"/page\">Link</a>"
				}
				body += "</body></html>"
			}

			page := htmlPage(body)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "links/excessive-links")

			if tt.wantIssue && !found {
				t.Errorf("expected links/excessive-links issue, got none in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect links/excessive-links issue, got %+v", issues)
			}
			if tt.wantSubstr != "" {
				for _, iss := range issues {
					if iss.CheckName == "links/excessive-links" {
						if !strings.Contains(iss.Message, tt.wantSubstr) {
							t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
						}
					}
				}
			}
		})
	}
}

func TestLinkChecker_GenericAnchorText(t *testing.T) {
	checker := NewLinkChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantIssue  bool
		wantSubstr string
		wantCount  int
	}{
		{
			name:       "click here",
			html:       `<html><body><a href="/page">Click here</a></body></html>`,
			wantIssue:  true,
			wantSubstr: "click here",
		},
		{
			name:       "read more",
			html:       `<html><body><a href="/page">Read more</a></body></html>`,
			wantIssue:  true,
			wantSubstr: "read more",
		},
		{
			name:       "learn more",
			html:       `<html><body><a href="/page">Learn more</a></body></html>`,
			wantIssue:  true,
			wantSubstr: "learn more",
		},
		{
			name:       "here",
			html:       `<html><body><a href="/page">here</a></body></html>`,
			wantIssue:  true,
			wantSubstr: `"here"`,
		},
		{
			name:       "more",
			html:       `<html><body><a href="/page">more</a></body></html>`,
			wantIssue:  true,
			wantSubstr: `"more"`,
		},
		{
			name:       "link",
			html:       `<html><body><a href="/page">link</a></body></html>`,
			wantIssue:  true,
			wantSubstr: `"link"`,
		},
		{
			name:       "this",
			html:       `<html><body><a href="/page">this</a></body></html>`,
			wantIssue:  true,
			wantSubstr: `"this"`,
		},
		{
			name:      "descriptive text is fine",
			html:      `<html><body><a href="/pricing">View pricing details</a></body></html>`,
			wantIssue: false,
		},
		{
			name:       "case insensitive",
			html:       `<html><body><a href="/page">CLICK HERE</a></body></html>`,
			wantIssue:  true,
			wantSubstr: "click here",
		},
		{
			name:       "whitespace trimmed",
			html:       `<html><body><a href="/page">  Read more  </a></body></html>`,
			wantIssue:  true,
			wantSubstr: "read more",
		},
		{
			name:      "read more as part of longer text is fine",
			html:      `<html><body><a href="/page">Read more about our services</a></body></html>`,
			wantIssue: false,
		},
		{
			name:      "multiple generic links",
			html:      `<html><body><a href="/a">Click here</a><a href="/b">Read more</a><a href="/c">Learn more</a></body></html>`,
			wantIssue: true,
			wantCount: 3,
		},
		{
			name:      "empty link text not flagged as generic",
			html:      `<html><body><a href="/page"></a></body></html>`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "links/generic-anchor-text")

			if tt.wantIssue && !found {
				t.Errorf("expected links/generic-anchor-text issue, got none in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect links/generic-anchor-text issue, got %+v", issues)
			}
			if tt.wantSubstr != "" {
				for _, iss := range issues {
					if iss.CheckName == "links/generic-anchor-text" {
						if strings.Contains(iss.Message, tt.wantSubstr) {
							goto substringFound
						}
					}
				}
				t.Errorf("no links/generic-anchor-text issue message contains %q", tt.wantSubstr)
			substringFound:
			}
			if tt.wantCount > 0 {
				count := countCheck(issues, "links/generic-anchor-text")
				if count != tt.wantCount {
					t.Errorf("expected %d generic-anchor-text issues, got %d", tt.wantCount, count)
				}
			}
		})
	}
}

func TestLinkChecker_CheckSite(t *testing.T) {
	checker := NewLinkChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		pages      []*model.Page
		wantCheck  string
		wantIssue  bool
		wantSubstr string
	}{
		{
			name: "broken internal link",
			pages: []*model.Page{
				{
					URL:        "https://example.com",
					StatusCode: 200,
					Depth:      0,
					Links:      []string{"https://example.com/broken"},
				},
				{
					URL:        "https://example.com/broken",
					StatusCode: 404,
					Depth:      1,
				},
			},
			wantCheck:  "links/broken-internal",
			wantIssue:  true,
			wantSubstr: "linked from https://example.com",
		},
		{
			name: "no broken links",
			pages: []*model.Page{
				{
					URL:        "https://example.com",
					StatusCode: 200,
					Depth:      0,
					Links:      []string{"https://example.com/page"},
				},
				{
					URL:        "https://example.com/page",
					StatusCode: 200,
					Depth:      1,
				},
			},
			wantCheck: "links/broken-internal",
			wantIssue: false,
		},
		{
			name: "orphan page",
			pages: []*model.Page{
				{
					URL:        "https://example.com",
					StatusCode: 200,
					Depth:      0,
				},
				{
					URL:        "https://example.com/orphan",
					StatusCode: 200,
					Depth:      1,
				},
			},
			wantCheck: "links/orphan-page",
			wantIssue: true,
		},
		{
			name: "root page not orphan",
			pages: []*model.Page{
				{
					URL:        "https://example.com",
					StatusCode: 200,
					Depth:      0,
				},
			},
			wantCheck: "links/orphan-page",
			wantIssue: false,
		},
		{
			name: "deep page",
			pages: []*model.Page{
				{
					URL:        "https://example.com",
					StatusCode: 200,
					Depth:      0,
					Links:      []string{"https://example.com/deep"},
				},
				{
					URL:        "https://example.com/deep",
					StatusCode: 200,
					Depth:      5,
				},
			},
			wantCheck:  "links/deep-page",
			wantIssue:  true,
			wantSubstr: "depth 5",
		},
		{
			name: "shallow page no depth issue",
			pages: []*model.Page{
				{
					URL:        "https://example.com",
					StatusCode: 200,
					Depth:      0,
					Links:      []string{"https://example.com/near"},
				},
				{
					URL:        "https://example.com/near",
					StatusCode: 200,
					Depth:      2,
				},
			},
			wantCheck: "links/deep-page",
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checker.CheckSite(ctx, tt.pages)

			if !tt.wantIssue {
				for _, iss := range issues {
					if iss.CheckName == tt.wantCheck {
						t.Fatalf("did not expect issue %s, got %+v", tt.wantCheck, iss)
					}
				}
				return
			}

			found := false
			for _, iss := range issues {
				if iss.CheckName == tt.wantCheck {
					found = true
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
					}
				}
			}
			if !found {
				t.Errorf("expected issue %s not found in %+v", tt.wantCheck, issues)
			}
		})
	}
}
