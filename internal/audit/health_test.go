package audit

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/meysam81/scry/internal/model"
)

func TestHealthChecker_Check(t *testing.T) {
	checker := NewHealthChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		page       *model.Page
		wantCheck  string
		wantSev    model.Severity
		wantIssue  bool
		wantSubstr string
	}{
		{
			name: "4xx status",
			page: &model.Page{
				URL:        "https://example.com/missing",
				StatusCode: 404,
			},
			wantCheck:  "health/4xx",
			wantSev:    model.SeverityCritical,
			wantIssue:  true,
			wantSubstr: "404",
		},
		{
			name: "5xx status",
			page: &model.Page{
				URL:        "https://example.com/error",
				StatusCode: 500,
			},
			wantCheck:  "health/5xx",
			wantSev:    model.SeverityCritical,
			wantIssue:  true,
			wantSubstr: "500",
		},
		{
			name: "200 status no issue",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers:    http.Header{},
			},
			wantCheck: "health/4xx",
			wantIssue: false,
		},
		{
			name: "redirect chain too long",
			page: &model.Page{
				URL:           "https://example.com/redir",
				StatusCode:    200,
				RedirectChain: []string{"https://a.com", "https://b.com", "https://c.com"},
				Headers:       http.Header{},
			},
			wantCheck:  "health/redirect-chain",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "3 hops",
		},
		{
			name: "redirect chain ok",
			page: &model.Page{
				URL:           "https://example.com/redir",
				StatusCode:    200,
				RedirectChain: []string{"https://a.com"},
				Headers:       http.Header{},
			},
			wantCheck: "health/redirect-chain",
			wantIssue: false,
		},
		{
			name: "redirect loop",
			page: &model.Page{
				URL:           "https://example.com/loop",
				StatusCode:    200,
				RedirectChain: []string{"https://example.com/loop"},
				Headers:       http.Header{},
			},
			wantCheck: "health/redirect-loop",
			wantSev:   model.SeverityCritical,
			wantIssue: true,
		},
		{
			name: "slow ttfb",
			page: &model.Page{
				URL:           "https://example.com/slow",
				StatusCode:    200,
				FetchDuration: 3 * time.Second,
				Headers:       http.Header{},
			},
			wantCheck:  "health/slow-ttfb",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "3s",
		},
		{
			name: "fast ttfb no issue",
			page: &model.Page{
				URL:           "https://example.com/fast",
				StatusCode:    200,
				FetchDuration: 500 * time.Millisecond,
				Headers:       http.Header{},
			},
			wantCheck: "health/slow-ttfb",
			wantIssue: false,
		},
		{
			name: "mixed content",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Assets:     []string{"http://cdn.example.com/style.css", "https://cdn.example.com/app.js"},
				Headers:    http.Header{},
			},
			wantCheck:  "health/mixed-content",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "1 HTTP assets",
		},
		{
			name: "no mixed content on http page",
			page: &model.Page{
				URL:        "http://example.com",
				StatusCode: 200,
				Assets:     []string{"http://cdn.example.com/style.css"},
				Headers:    http.Header{},
			},
			wantCheck: "health/mixed-content",
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checker.Check(ctx, tt.page)

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
					if iss.Severity != tt.wantSev {
						t.Errorf("expected severity %s, got %s", tt.wantSev, iss.Severity)
					}
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
