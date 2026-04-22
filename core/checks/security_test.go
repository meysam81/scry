package checks

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/meysam81/scry/core/model"
)

func TestSecurityChecker_Name(t *testing.T) {
	c := NewSecurityChecker()
	if got := c.Name(); got != "security" {
		t.Errorf("Name() = %q, want %q", got, "security")
	}
}

func TestSecurityChecker_SkipsNon200(t *testing.T) {
	checker := NewSecurityChecker()
	ctx := context.Background()

	for _, code := range []int{301, 302, 404, 500, 503} {
		page := &model.Page{
			URL:        "https://example.com",
			StatusCode: code,
			Headers:    http.Header{},
		}
		issues := checker.Check(ctx, page)
		if len(issues) != 0 {
			t.Errorf("status %d: expected no issues, got %d", code, len(issues))
		}
	}
}

func TestSecurityChecker_Check(t *testing.T) {
	checker := NewSecurityChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		page       *model.Page
		wantCheck  string
		wantSev    model.Severity
		wantIssue  bool
		wantSubstr string
	}{
		// --- Strict-Transport-Security ---
		{
			name: "missing hsts on https page",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers:    http.Header{},
			},
			wantCheck: "security/missing-strict-transport-security",
			wantSev:   model.SeverityWarning,
			wantIssue: true,
		},
		{
			name: "hsts present and strong",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Strict-Transport-Security": []string{"max-age=63072000; includeSubDomains; preload"},
				},
			},
			wantCheck: "security/missing-strict-transport-security",
			wantIssue: false,
		},
		{
			name: "hsts not checked on http page",
			page: &model.Page{
				URL:        "http://example.com",
				StatusCode: 200,
				Headers:    http.Header{},
			},
			wantCheck: "security/missing-strict-transport-security",
			wantIssue: false,
		},
		{
			name: "weak hsts max-age below 1 year",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Strict-Transport-Security": []string{"max-age=86400"},
				},
			},
			wantCheck:  "security/weak-hsts",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "86400",
		},
		{
			name: "hsts max-age exactly 1 year no weak issue",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Strict-Transport-Security": []string{"max-age=31536000"},
				},
			},
			wantCheck: "security/weak-hsts",
			wantIssue: false,
		},
		{
			name: "hsts max-age zero",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Strict-Transport-Security": []string{"max-age=0"},
				},
			},
			wantCheck:  "security/weak-hsts",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "0 seconds",
		},
		{
			name: "hsts with unparseable max-age treated as missing directive",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Strict-Transport-Security": []string{"max-age=abc"},
				},
			},
			wantCheck: "security/weak-hsts",
			wantIssue: false,
		},

		// --- Content-Security-Policy ---
		{
			name: "missing csp",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers:    http.Header{},
			},
			wantCheck: "security/missing-content-security-policy",
			wantSev:   model.SeverityWarning,
			wantIssue: true,
		},
		{
			name: "csp present",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Content-Security-Policy": []string{"default-src 'self'"},
				},
			},
			wantCheck: "security/missing-content-security-policy",
			wantIssue: false,
		},

		// --- X-Content-Type-Options ---
		{
			name: "missing x-content-type-options",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers:    http.Header{},
			},
			wantCheck: "security/missing-x-content-type-options",
			wantSev:   model.SeverityWarning,
			wantIssue: true,
		},
		{
			name: "x-content-type-options nosniff",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"X-Content-Type-Options": []string{"nosniff"},
				},
			},
			wantCheck: "security/missing-x-content-type-options",
			wantIssue: false,
		},
		{
			name: "x-content-type-options case insensitive",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"X-Content-Type-Options": []string{"NOSNIFF"},
				},
			},
			wantCheck: "security/missing-x-content-type-options",
			wantIssue: false,
		},
		{
			name: "x-content-type-options wrong value",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"X-Content-Type-Options": []string{"nosnif"},
				},
			},
			wantCheck: "security/missing-x-content-type-options",
			wantSev:   model.SeverityWarning,
			wantIssue: true,
		},

		// --- X-Frame-Options ---
		{
			name: "missing x-frame-options",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers:    http.Header{},
			},
			wantCheck: "security/missing-x-frame-options",
			wantSev:   model.SeverityInfo,
			wantIssue: true,
		},
		{
			name: "x-frame-options DENY",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"X-Frame-Options": []string{"DENY"},
				},
			},
			wantCheck: "security/missing-x-frame-options",
			wantIssue: false,
		},
		{
			name: "x-frame-options SAMEORIGIN",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"X-Frame-Options": []string{"SAMEORIGIN"},
				},
			},
			wantCheck: "security/missing-x-frame-options",
			wantIssue: false,
		},
		{
			name: "x-frame-options lowercase sameorigin",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"X-Frame-Options": []string{"sameorigin"},
				},
			},
			wantCheck: "security/missing-x-frame-options",
			wantIssue: false,
		},
		{
			name: "x-frame-options ALLOW-FROM triggers issue",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"X-Frame-Options": []string{"ALLOW-FROM https://other.com"},
				},
			},
			wantCheck: "security/missing-x-frame-options",
			wantSev:   model.SeverityInfo,
			wantIssue: true,
		},

		// --- Referrer-Policy ---
		{
			name: "missing referrer-policy",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers:    http.Header{},
			},
			wantCheck: "security/missing-referrer-policy",
			wantSev:   model.SeverityInfo,
			wantIssue: true,
		},
		{
			name: "referrer-policy strict-origin-when-cross-origin",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Referrer-Policy": []string{"strict-origin-when-cross-origin"},
				},
			},
			wantCheck: "security/missing-referrer-policy",
			wantIssue: false,
		},
		{
			name: "insecure referrer-policy unsafe-url",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Referrer-Policy": []string{"unsafe-url"},
				},
			},
			wantCheck:  "security/insecure-referrer-policy",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "unsafe-url",
		},
		{
			name: "insecure referrer-policy no-referrer-when-downgrade",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Referrer-Policy": []string{"no-referrer-when-downgrade"},
				},
			},
			wantCheck:  "security/insecure-referrer-policy",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "no-referrer-when-downgrade",
		},
		{
			name: "referrer-policy no-referrer is safe",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Referrer-Policy": []string{"no-referrer"},
				},
			},
			wantCheck: "security/insecure-referrer-policy",
			wantIssue: false,
		},

		// --- Permissions-Policy ---
		{
			name: "missing permissions-policy",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers:    http.Header{},
			},
			wantCheck: "security/missing-permissions-policy",
			wantSev:   model.SeverityInfo,
			wantIssue: true,
		},
		{
			name: "permissions-policy present",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Permissions-Policy": []string{"geolocation=(), camera=()"},
				},
			},
			wantCheck: "security/missing-permissions-policy",
			wantIssue: false,
		},

		// --- CSP Unsafe ---
		{
			name: "csp with unsafe-inline",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Content-Security-Policy": []string{"default-src 'self'; script-src 'unsafe-inline'"},
				},
			},
			wantCheck:  "security/csp-unsafe",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "'unsafe-inline'",
		},
		{
			name: "csp with unsafe-eval",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Content-Security-Policy": []string{"default-src 'self'; script-src 'unsafe-eval'"},
				},
			},
			wantCheck:  "security/csp-unsafe",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "'unsafe-eval'",
		},
		{
			name: "csp with both unsafe-inline and unsafe-eval",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Content-Security-Policy": []string{"script-src 'unsafe-inline' 'unsafe-eval'"},
				},
			},
			wantCheck:  "security/csp-unsafe",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "'unsafe-inline' and 'unsafe-eval'",
		},
		{
			name: "csp safe no unsafe directives",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Content-Security-Policy": []string{"default-src 'self'; script-src 'nonce-abc123'"},
				},
			},
			wantCheck: "security/csp-unsafe",
			wantIssue: false,
		},
		{
			name: "csp absent does not trigger csp-unsafe",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers:    http.Header{},
			},
			wantCheck: "security/csp-unsafe",
			wantIssue: false,
		},
		{
			name: "csp unsafe-inline case insensitive",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Content-Security-Policy": []string{"script-src 'UNSAFE-INLINE'"},
				},
			},
			wantCheck:  "security/csp-unsafe",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "'unsafe-inline'",
		},

		// --- CORS Wildcard ---
		{
			name: "cors wildcard origin",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Access-Control-Allow-Origin": []string{"*"},
				},
			},
			wantCheck:  "security/cors-wildcard",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "wildcard",
		},
		{
			name: "cors specific origin no issue",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers: http.Header{
					"Access-Control-Allow-Origin": []string{"https://trusted.com"},
				},
			},
			wantCheck: "security/cors-wildcard",
			wantIssue: false,
		},
		{
			name: "cors absent no issue",
			page: &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers:    http.Header{},
			},
			wantCheck: "security/cors-wildcard",
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

func TestSecurityChecker_AllHeaders_Present(t *testing.T) {
	checker := NewSecurityChecker()
	ctx := context.Background()

	page := &model.Page{
		URL:        "https://example.com",
		StatusCode: 200,
		Headers: http.Header{
			"Strict-Transport-Security": []string{"max-age=63072000; includeSubDomains"},
			"Content-Security-Policy":   []string{"default-src 'self'"},
			"X-Content-Type-Options":    []string{"nosniff"},
			"X-Frame-Options":           []string{"DENY"},
			"Referrer-Policy":           []string{"strict-origin-when-cross-origin"},
			"Permissions-Policy":        []string{"geolocation=()"},
		},
	}

	issues := checker.Check(ctx, page)
	if len(issues) != 0 {
		t.Errorf("expected zero issues for fully-secured page, got %d: %+v", len(issues), issues)
	}
}

func TestSecurityChecker_AllHeaders_Missing(t *testing.T) {
	checker := NewSecurityChecker()
	ctx := context.Background()

	page := &model.Page{
		URL:        "https://example.com",
		StatusCode: 200,
		Headers:    http.Header{},
	}

	issues := checker.Check(ctx, page)

	expected := map[string]bool{
		"security/missing-strict-transport-security": false,
		"security/missing-content-security-policy":   false,
		"security/missing-x-content-type-options":    false,
		"security/missing-x-frame-options":           false,
		"security/missing-referrer-policy":           false,
		"security/missing-permissions-policy":        false,
	}

	for _, iss := range issues {
		if _, ok := expected[iss.CheckName]; ok {
			expected[iss.CheckName] = true
		}
	}

	for check, found := range expected {
		if !found {
			t.Errorf("expected issue %s was not reported", check)
		}
	}
}

func TestSecurityChecker_InsecureCookies(t *testing.T) {
	checker := NewSecurityChecker()
	ctx := context.Background()

	tests := []struct {
		name        string
		cookies     []string
		wantIssues  int
		wantSubstrs []string // one per expected issue
	}{
		{
			name:       "no cookies no issue",
			cookies:    nil,
			wantIssues: 0,
		},
		{
			name:        "cookie missing all flags",
			cookies:     []string{"session=abc123; Path=/"},
			wantIssues:  1,
			wantSubstrs: []string{"HttpOnly, Secure, SameSite"},
		},
		{
			name:       "cookie with all flags present",
			cookies:    []string{"session=abc123; Path=/; HttpOnly; Secure; SameSite=Strict"},
			wantIssues: 0,
		},
		{
			name:        "cookie missing only HttpOnly",
			cookies:     []string{"session=abc123; Secure; SameSite=Lax"},
			wantIssues:  1,
			wantSubstrs: []string{"HttpOnly"},
		},
		{
			name:        "cookie missing only Secure",
			cookies:     []string{"session=abc123; HttpOnly; SameSite=Lax"},
			wantIssues:  1,
			wantSubstrs: []string{"Secure"},
		},
		{
			name:        "cookie missing only SameSite",
			cookies:     []string{"session=abc123; HttpOnly; Secure"},
			wantIssues:  1,
			wantSubstrs: []string{"SameSite"},
		},
		{
			name: "multiple cookies one secure one insecure",
			cookies: []string{
				"safe=yes; HttpOnly; Secure; SameSite=Strict",
				"tracking=123; Path=/",
			},
			wantIssues:  1,
			wantSubstrs: []string{"tracking"},
		},
		{
			name: "multiple insecure cookies",
			cookies: []string{
				"a=1; Path=/",
				"b=2; Path=/",
			},
			wantIssues: 2,
		},
		{
			name:       "flags case insensitive",
			cookies:    []string{"session=abc123; httponly; secure; samesite=Lax"},
			wantIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			for _, c := range tt.cookies {
				headers.Add("Set-Cookie", c)
			}

			page := &model.Page{
				URL:        "https://example.com",
				StatusCode: 200,
				Headers:    headers,
			}

			allIssues := checker.Check(ctx, page)
			var cookieIssues []model.Issue
			for _, iss := range allIssues {
				if iss.CheckName == "security/insecure-cookies" {
					cookieIssues = append(cookieIssues, iss)
				}
			}

			if len(cookieIssues) != tt.wantIssues {
				t.Fatalf("expected %d cookie issues, got %d: %+v", tt.wantIssues, len(cookieIssues), cookieIssues)
			}

			for i, substr := range tt.wantSubstrs {
				if i >= len(cookieIssues) {
					break
				}
				if !strings.Contains(cookieIssues[i].Message, substr) {
					t.Errorf("issue %d: expected message containing %q, got %q", i, substr, cookieIssues[i].Message)
				}
				if cookieIssues[i].Severity != model.SeverityWarning {
					t.Errorf("issue %d: expected severity %s, got %s", i, model.SeverityWarning, cookieIssues[i].Severity)
				}
			}
		})
	}
}

func TestSecurityChecker_MissingSRI(t *testing.T) {
	checker := NewSecurityChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantIssues int
		wantSubstr string
	}{
		{
			name:       "external script without integrity",
			html:       `<html><head><script src="https://cdn.example.com/lib.js"></script></head><body></body></html>`,
			wantIssues: 1,
			wantSubstr: "cdn.example.com/lib.js",
		},
		{
			name:       "external script with integrity",
			html:       `<html><head><script src="https://cdn.example.com/lib.js" integrity="sha384-abc"></script></head><body></body></html>`,
			wantIssues: 0,
		},
		{
			name:       "same-origin script no issue",
			html:       `<html><head><script src="/js/app.js"></script></head><body></body></html>`,
			wantIssues: 0,
		},
		{
			name:       "external stylesheet without integrity",
			html:       `<html><head><link rel="stylesheet" href="https://cdn.example.com/style.css"></head><body></body></html>`,
			wantIssues: 1,
			wantSubstr: "cdn.example.com/style.css",
		},
		{
			name:       "external stylesheet with integrity",
			html:       `<html><head><link rel="stylesheet" href="https://cdn.example.com/style.css" integrity="sha384-xyz"></head><body></body></html>`,
			wantIssues: 0,
		},
		{
			name:       "same-origin stylesheet no issue",
			html:       `<html><head><link rel="stylesheet" href="/css/main.css"></head><body></body></html>`,
			wantIssues: 0,
		},
		{
			name: "multiple external resources mixed",
			html: `<html><head>
				<script src="https://cdn.example.com/a.js"></script>
				<script src="https://cdn.example.com/b.js" integrity="sha256-abc"></script>
				<link rel="stylesheet" href="https://cdn.example.com/c.css">
			</head><body></body></html>`,
			wantIssues: 2,
		},
		{
			name:       "protocol-relative external script",
			html:       `<html><head><script src="//cdn.example.com/lib.js"></script></head><body></body></html>`,
			wantIssues: 1,
			wantSubstr: "cdn.example.com/lib.js",
		},
		{
			name:       "inline script no issue",
			html:       `<html><head><script>var x = 1;</script></head><body></body></html>`,
			wantIssues: 0,
		},
		{
			name:       "link rel=icon not stylesheet no issue",
			html:       `<html><head><link rel="icon" href="https://cdn.example.com/favicon.ico"></head><body></body></html>`,
			wantIssues: 0,
		},
		{
			name:       "non-html content type skipped",
			html:       `{"key": "value"}`,
			wantIssues: -1, // special: test with non-HTML content type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearDocCache()

			contentType := "text/html; charset=utf-8"
			if tt.wantIssues == -1 {
				contentType = "application/json"
				tt.wantIssues = 0
			}

			page := &model.Page{
				URL:         "https://example.com/page",
				StatusCode:  200,
				ContentType: contentType,
				Headers:     http.Header{},
				Body:        []byte(tt.html),
			}

			allIssues := checker.Check(ctx, page)
			var sriIssues []model.Issue
			for _, iss := range allIssues {
				if iss.CheckName == "security/missing-sri" {
					sriIssues = append(sriIssues, iss)
				}
			}

			if len(sriIssues) != tt.wantIssues {
				t.Fatalf("expected %d SRI issues, got %d: %+v", tt.wantIssues, len(sriIssues), sriIssues)
			}

			if tt.wantSubstr != "" && len(sriIssues) > 0 {
				if !strings.Contains(sriIssues[0].Message, tt.wantSubstr) {
					t.Errorf("expected message containing %q, got %q", tt.wantSubstr, sriIssues[0].Message)
				}
				if sriIssues[0].Severity != model.SeverityInfo {
					t.Errorf("expected severity %s, got %s", model.SeverityInfo, sriIssues[0].Severity)
				}
			}
		})
	}
}

func TestSecurityChecker_MissingSecurityTxt(t *testing.T) {
	checker := NewSecurityChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		pages      []*model.Page
		wantIssue  bool
		wantSubstr string
	}{
		{
			name: "security.txt present",
			pages: []*model.Page{
				{URL: "https://example.com/", StatusCode: 200},
				{URL: "https://example.com/.well-known/security.txt", StatusCode: 200},
			},
			wantIssue: false,
		},
		{
			name: "security.txt absent",
			pages: []*model.Page{
				{URL: "https://example.com/", StatusCode: 200},
				{URL: "https://example.com/about", StatusCode: 200},
			},
			wantIssue:  true,
			wantSubstr: "security.txt",
		},
		{
			name:      "empty pages list",
			pages:     []*model.Page{},
			wantIssue: true,
		},
		{
			name: "single page no security.txt",
			pages: []*model.Page{
				{URL: "https://example.com/", StatusCode: 200},
			},
			wantIssue:  true,
			wantSubstr: "security.txt",
		},
		{
			name: "security.txt in wrong path not matched",
			pages: []*model.Page{
				{URL: "https://example.com/", StatusCode: 200},
				{URL: "https://example.com/security.txt", StatusCode: 200},
			},
			wantIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checker.CheckSite(ctx, tt.pages)

			var found *model.Issue
			for i := range issues {
				if issues[i].CheckName == "security/missing-security-txt" {
					found = &issues[i]
					break
				}
			}

			if tt.wantIssue {
				if found == nil {
					t.Fatal("expected security/missing-security-txt issue, got none")
				}
				if found.Severity != model.SeverityInfo {
					t.Errorf("expected severity %s, got %s", model.SeverityInfo, found.Severity)
				}
				if tt.wantSubstr != "" && !strings.Contains(found.Message, tt.wantSubstr) {
					t.Errorf("expected message containing %q, got %q", tt.wantSubstr, found.Message)
				}
			} else if found != nil {
				t.Errorf("did not expect security/missing-security-txt issue, got %+v", *found)
			}
		})
	}
}

func TestSecurityChecker_SiteCheckerInterface(t *testing.T) {
	checker := NewSecurityChecker()
	// Verify SecurityChecker implements SiteChecker at compile time.
	var _ SiteChecker = checker
}

func TestParseHSTSMaxAge(t *testing.T) {
	tests := []struct {
		header string
		want   int64
	}{
		{"max-age=31536000", 31536000},
		{"max-age=0", 0},
		{"max-age=63072000; includeSubDomains; preload", 63072000},
		{"  max-age=86400 ; includeSubDomains", 86400},
		{"includeSubDomains; max-age=12345", 12345},
		{"max-age=abc", -1},
		{"includeSubDomains", -1},
		{"", -1},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := parseHSTSMaxAge(tt.header)
			if got != tt.want {
				t.Errorf("parseHSTSMaxAge(%q) = %d, want %d", tt.header, got, tt.want)
			}
		})
	}
}
