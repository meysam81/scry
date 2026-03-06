package audit

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestPerformanceChecker_Check(t *testing.T) {
	checker := NewPerformanceChecker()
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
			name: "large html",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        make([]byte, 150*1024),
				Headers:     http.Header{"Content-Encoding": []string{"gzip"}},
			},
			wantCheck:  "performance/large-html",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "153600 bytes",
		},
		{
			name: "small html no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte("<html><head></head><body>small</body></html>"),
				Headers:     http.Header{"Content-Encoding": []string{"gzip"}},
			},
			wantCheck: "performance/large-html",
			wantIssue: false,
		},
		{
			name: "no compression",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte("<html><head></head><body></body></html>"),
				Headers:     http.Header{},
			},
			wantCheck: "performance/no-compression",
			wantSev:   model.SeverityWarning,
			wantIssue: true,
		},
		{
			name: "gzip compression no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte("<html><head></head><body></body></html>"),
				Headers:     http.Header{"Content-Encoding": []string{"gzip"}},
			},
			wantCheck: "performance/no-compression",
			wantIssue: false,
		},
		{
			name: "br compression no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte("<html><head></head><body></body></html>"),
				Headers:     http.Header{"Content-Encoding": []string{"br"}},
			},
			wantCheck: "performance/no-compression",
			wantIssue: false,
		},
		{
			name: "render blocking script",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><script src="app.js"></script></head><body></body></html>`),
				Headers:     http.Header{"Content-Encoding": []string{"gzip"}},
			},
			wantCheck:  "performance/render-blocking-script",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "app.js",
		},
		{
			name: "async script no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><script src="app.js" async></script></head><body></body></html>`),
				Headers:     http.Header{"Content-Encoding": []string{"gzip"}},
			},
			wantCheck: "performance/render-blocking-script",
			wantIssue: false,
		},
		{
			name: "defer script no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><script src="app.js" defer></script></head><body></body></html>`),
				Headers:     http.Header{"Content-Encoding": []string{"gzip"}},
			},
			wantCheck: "performance/render-blocking-script",
			wantIssue: false,
		},
		{
			name: "inline script no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><script>var x = 1;</script></head><body></body></html>`),
				Headers:     http.Header{"Content-Encoding": []string{"gzip"}},
			},
			wantCheck: "performance/render-blocking-script",
			wantIssue: false,
		},
		{
			name: "excessive css",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body: []byte(`<html><head>
					<link rel="stylesheet" href="a.css">
					<link rel="stylesheet" href="b.css">
					<link rel="stylesheet" href="c.css">
					<link rel="stylesheet" href="d.css">
				</head><body></body></html>`),
				Headers: http.Header{"Content-Encoding": []string{"gzip"}},
			},
			wantCheck:  "performance/excessive-css",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "4 stylesheets",
		},
		{
			name: "3 css no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body: []byte(`<html><head>
					<link rel="stylesheet" href="a.css">
					<link rel="stylesheet" href="b.css">
					<link rel="stylesheet" href="c.css">
				</head><body></body></html>`),
				Headers: http.Header{"Content-Encoding": []string{"gzip"}},
			},
			wantCheck: "performance/excessive-css",
			wantIssue: false,
		},
		{
			name: "non-html page skipped",
			page: &model.Page{
				URL:         "https://example.com/file.pdf",
				ContentType: "application/pdf",
				Body:        make([]byte, 150*1024),
				Headers:     http.Header{},
			},
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
