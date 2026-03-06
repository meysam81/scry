package audit

import (
	"context"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestLinkChecker_Check_ReturnsNil(t *testing.T) {
	checker := NewLinkChecker()
	page := &model.Page{URL: "https://example.com"}
	issues := checker.Check(context.Background(), page)
	if issues != nil {
		t.Fatalf("expected nil from Check, got %+v", issues)
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
