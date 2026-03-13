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

	// defaultHeaders provides gzip + cache-control + alt-svc to avoid
	// triggering unrelated checks in tests that don't target them.
	defaultHeaders := http.Header{
		"Content-Encoding": []string{"gzip"},
		"Cache-Control":    []string{"max-age=3600"},
		"Alt-Svc":          []string{`h2=":443"`},
	}

	tests := []struct {
		name       string
		page       *model.Page
		wantCheck  string
		wantSev    model.Severity
		wantIssue  bool
		wantSubstr string
	}{
		// --- existing checks ---
		{
			name: "large html",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        make([]byte, 150*1024),
				Headers:     defaultHeaders,
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
				Headers:     defaultHeaders,
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
				Headers:     http.Header{"Cache-Control": []string{"max-age=3600"}, "Alt-Svc": []string{`h2=":443"`}},
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
				Headers:     defaultHeaders,
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
				Headers: http.Header{
					"Content-Encoding": []string{"br"},
					"Cache-Control":    []string{"max-age=3600"},
					"Alt-Svc":          []string{`h2=":443"`},
				},
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
				Headers:     defaultHeaders,
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
				Headers:     defaultHeaders,
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
				Headers:     defaultHeaders,
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
				Headers:     defaultHeaders,
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
				Headers: defaultHeaders,
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
				Headers: defaultHeaders,
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

		// --- 1. render-blocking-css ---
		{
			name: "render blocking css without media",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><link rel="stylesheet" href="style.css"></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck:  "performance/render-blocking-css",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "style.css",
		},
		{
			name: "render blocking css with media all",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><link rel="stylesheet" href="style.css" media="all"></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck:  "performance/render-blocking-css",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "style.css",
		},
		{
			name: "non-blocking css with media print",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><link rel="stylesheet" href="print.css" media="print"></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/render-blocking-css",
			wantIssue: false,
		},
		{
			name: "non-blocking css with media query",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><link rel="stylesheet" href="mobile.css" media="(max-width: 600px)"></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/render-blocking-css",
			wantIssue: false,
		},

		// --- 2. missing-resource-hints ---
		{
			name: "missing resource hints",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><link rel="stylesheet" href="style.css"></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/missing-resource-hints",
			wantSev:   model.SeverityInfo,
			wantIssue: true,
		},
		{
			name: "has preconnect no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><link rel="preconnect" href="https://fonts.gstatic.com"></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/missing-resource-hints",
			wantIssue: false,
		},
		{
			name: "has dns-prefetch no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><link rel="dns-prefetch" href="https://cdn.example.com"></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/missing-resource-hints",
			wantIssue: false,
		},

		// --- 3. font-loading ---
		{
			name: "font-face missing font-display",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><style>@font-face { font-family: "MyFont"; src: url(font.woff2); }</style></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck:  "performance/font-loading",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "missing font-display",
		},
		{
			name: "font-face with font-display block",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><style>@font-face { font-family: "MyFont"; src: url(font.woff2); font-display: block; }</style></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck:  "performance/font-loading",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "font-display: block",
		},
		{
			name: "font-face with font-display swap no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><style>@font-face { font-family: "MyFont"; src: url(font.woff2); font-display: swap; }</style></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/font-loading",
			wantIssue: false,
		},

		// --- 4. excessive-third-party ---
		{
			name: "excessive third party scripts",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body: []byte(`<html><head></head><body>
					<script src="https://a.com/a.js"></script>
					<script src="https://b.com/b.js"></script>
					<script src="https://c.com/c.js"></script>
					<script src="https://d.com/d.js"></script>
					<script src="https://e.com/e.js"></script>
					<script src="https://f.com/f.js"></script>
				</body></html>`),
				Headers: defaultHeaders,
			},
			wantCheck:  "performance/excessive-third-party",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "6 external origins",
		},
		{
			name: "5 third party scripts no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body: []byte(`<html><head></head><body>
					<script src="https://a.com/a.js"></script>
					<script src="https://b.com/b.js"></script>
					<script src="https://c.com/c.js"></script>
					<script src="https://d.com/d.js"></script>
					<script src="https://e.com/e.js"></script>
				</body></html>`),
				Headers: defaultHeaders,
			},
			wantCheck: "performance/excessive-third-party",
			wantIssue: false,
		},
		{
			name: "same-origin scripts no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body: []byte(`<html><head></head><body>
					<script src="https://example.com/a.js"></script>
					<script src="https://example.com/b.js"></script>
					<script src="/c.js"></script>
				</body></html>`),
				Headers: defaultHeaders,
			},
			wantCheck: "performance/excessive-third-party",
			wantIssue: false,
		},

		// --- 5. excessive-dom-size ---
		{
			name: "excessive dom size",
			page: func() *model.Page {
				var sb strings.Builder
				sb.WriteString("<html><head></head><body>")
				for i := 0; i < 1500; i++ {
					sb.WriteString("<div></div>")
				}
				sb.WriteString("</body></html>")
				return &model.Page{
					URL:         "https://example.com",
					ContentType: "text/html",
					Body:        []byte(sb.String()),
					Headers:     defaultHeaders,
				}
			}(),
			wantCheck:  "performance/excessive-dom-size",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "element nodes",
		},
		{
			name: "small dom no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head></head><body><div><p>Hello</p></div></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/excessive-dom-size",
			wantIssue: false,
		},

		// --- 6. inline-bloat ---
		{
			name: "inline bloat exceeds threshold",
			page: func() *model.Page {
				bigScript := strings.Repeat("x", 60*1024)
				body := `<html><head></head><body><script>` + bigScript + `</script></body></html>`
				return &model.Page{
					URL:         "https://example.com",
					ContentType: "text/html",
					Body:        []byte(body),
					Headers:     defaultHeaders,
				}
			}(),
			wantCheck:  "performance/inline-bloat",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "inline <script> and <style>",
		},
		{
			name: "small inline content no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head><style>body{margin:0}</style></head><body><script>var x=1;</script></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/inline-bloat",
			wantIssue: false,
		},
		{
			name: "external script src not counted for inline bloat",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head></head><body><script src="big.js">` + strings.Repeat("x", 60*1024) + `</script></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/inline-bloat",
			wantIssue: false,
		},

		// --- 7. missing-cache-headers ---
		{
			name: "missing cache control",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head></head><body></body></html>`),
				Headers: http.Header{
					"Content-Encoding": []string{"gzip"},
					"Alt-Svc":          []string{`h2=":443"`},
				},
			},
			wantCheck: "performance/missing-cache-headers",
			wantSev:   model.SeverityWarning,
			wantIssue: true,
		},
		{
			name: "has cache control no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/missing-cache-headers",
			wantIssue: false,
		},

		// --- 8. excessive-webfonts ---
		{
			name: "excessive webfonts",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body: []byte(`<html><head><style>
					@font-face { font-family: "A"; src: url(a.woff2); font-display: swap; }
					@font-face { font-family: "B"; src: url(b.woff2); font-display: swap; }
					@font-face { font-family: "C"; src: url(c.woff2); font-display: swap; }
					@font-face { font-family: "D"; src: url(d.woff2); font-display: swap; }
					@font-face { font-family: "E"; src: url(e.woff2); font-display: swap; }
				</style></head><body></body></html>`),
				Headers: defaultHeaders,
			},
			wantCheck:  "performance/excessive-webfonts",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "5 @font-face",
		},
		{
			name: "4 webfonts no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body: []byte(`<html><head><style>
					@font-face { font-family: "A"; src: url(a.woff2); font-display: swap; }
					@font-face { font-family: "B"; src: url(b.woff2); font-display: swap; }
					@font-face { font-family: "C"; src: url(c.woff2); font-display: swap; }
					@font-face { font-family: "D"; src: url(d.woff2); font-display: swap; }
				</style></head><body></body></html>`),
				Headers: defaultHeaders,
			},
			wantCheck: "performance/excessive-webfonts",
			wantIssue: false,
		},

		// --- 9. unminified-resources ---
		{
			name: "unminified script with multi-line comment",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head></head><body><script>/* This is a long comment explaining the code */var x=1;</script></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck:  "performance/unminified-resources",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "unminified",
		},
		{
			name: "unminified script with excessive whitespace",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte("<html><head></head><body><script>var x = 1;\n\n\n    var y = 2;</script></body></html>"),
				Headers:     defaultHeaders,
			},
			wantCheck:  "performance/unminified-resources",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "unminified",
		},
		{
			name: "minified script no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head></head><body><script>var x=1;var y=2;var z=3;</script></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/unminified-resources",
			wantIssue: false,
		},
		{
			name: "external script not checked for minification",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head></head><body><script src="app.js">/* big comment here */</script></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/unminified-resources",
			wantIssue: false,
		},

		// --- 10. no-http2 ---
		{
			name: "no http2 headers",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head></head><body></body></html>`),
				Headers: http.Header{
					"Content-Encoding": []string{"gzip"},
					"Cache-Control":    []string{"max-age=3600"},
				},
			},
			wantCheck: "performance/no-http2",
			wantSev:   model.SeverityInfo,
			wantIssue: true,
		},
		{
			name: "has h2 alt-svc no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head></head><body></body></html>`),
				Headers:     defaultHeaders,
			},
			wantCheck: "performance/no-http2",
			wantIssue: false,
		},
		{
			name: "has h3 alt-svc no issue",
			page: &model.Page{
				URL:         "https://example.com",
				ContentType: "text/html",
				Body:        []byte(`<html><head></head><body></body></html>`),
				Headers: http.Header{
					"Content-Encoding": []string{"gzip"},
					"Cache-Control":    []string{"max-age=3600"},
					"Alt-Svc":          []string{`h3=":443"; ma=86400`},
				},
			},
			wantCheck: "performance/no-http2",
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
