package checks

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meysam81/scry/core/model"
)

func TestImageChecker_Check(t *testing.T) {
	checker := NewImageChecker()
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
			name:       "missing alt",
			html:       `<html><body><img src="photo.jpg"></body></html>`,
			wantCheck:  "images/missing-alt",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "photo.jpg",
		},
		{
			name:      "alt present no issue",
			html:      `<html><body><img src="photo.jpg" alt="A photo"></body></html>`,
			wantCheck: "images/missing-alt",
			wantIssue: false,
		},
		{
			name:       "empty alt in link",
			html:       `<html><body><a href="/"><img src="logo.png" alt=""></a></body></html>`,
			wantCheck:  "images/empty-alt-in-link",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "logo.png",
		},
		{
			name:      "empty alt not in link no issue",
			html:      `<html><body><img src="spacer.gif" alt=""></body></html>`,
			wantCheck: "images/empty-alt-in-link",
			wantIssue: false,
		},
		{
			name:      "non-html page skipped",
			html:      "",
			wantIssue: false,
		},
		// legacy-format checks
		{
			name:       "legacy jpg format",
			html:       `<html><body><img src="photo.jpg" alt="photo"></body></html>`,
			wantCheck:  "images/legacy-format",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: ".jpg",
		},
		{
			name:       "legacy jpeg format",
			html:       `<html><body><img src="photo.jpeg" alt="photo"></body></html>`,
			wantCheck:  "images/legacy-format",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: ".jpeg",
		},
		{
			name:       "legacy gif format",
			html:       `<html><body><img src="animation.gif" alt="anim"></body></html>`,
			wantCheck:  "images/legacy-format",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: ".gif",
		},
		{
			name:       "legacy bmp format",
			html:       `<html><body><img src="old.bmp" alt="old"></body></html>`,
			wantCheck:  "images/legacy-format",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: ".bmp",
		},
		{
			name:       "legacy tiff format",
			html:       `<html><body><img src="scan.tiff" alt="scan"></body></html>`,
			wantCheck:  "images/legacy-format",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: ".tiff",
		},
		{
			name:       "legacy format case insensitive",
			html:       `<html><body><img src="PHOTO.JPG" alt="photo"></body></html>`,
			wantCheck:  "images/legacy-format",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: ".jpg",
		},
		{
			name:      "modern webp no issue",
			html:      `<html><body><img src="photo.webp" alt="photo"></body></html>`,
			wantCheck: "images/legacy-format",
			wantIssue: false,
		},
		{
			name:      "modern avif no issue",
			html:      `<html><body><img src="photo.avif" alt="photo"></body></html>`,
			wantCheck: "images/legacy-format",
			wantIssue: false,
		},
		{
			name:       "legacy format with query string",
			html:       `<html><body><img src="photo.jpg?v=123" alt="photo"></body></html>`,
			wantCheck:  "images/legacy-format",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: ".jpg",
		},
		// missing-lazy-loading checks
		{
			name:       "4th image missing lazy loading",
			html:       `<html><body><img src="1.webp" alt="1"><img src="2.webp" alt="2"><img src="3.webp" alt="3"><img src="4.webp" alt="4"></body></html>`,
			wantCheck:  "images/missing-lazy-loading",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "4.webp",
		},
		{
			name:      "first 3 images skip lazy loading check",
			html:      `<html><body><img src="1.webp" alt="1"><img src="2.webp" alt="2"><img src="3.webp" alt="3"></body></html>`,
			wantCheck: "images/missing-lazy-loading",
			wantIssue: false,
		},
		{
			name:      "4th image with lazy loading no issue",
			html:      `<html><body><img src="1.webp" alt="1"><img src="2.webp" alt="2"><img src="3.webp" alt="3"><img src="4.webp" alt="4" loading="lazy"></body></html>`,
			wantCheck: "images/missing-lazy-loading",
			wantIssue: false,
		},
		{
			name:      "image in header skipped",
			html:      `<html><body><img src="1.webp" alt="1"><img src="2.webp" alt="2"><img src="3.webp" alt="3"><header><img src="logo.webp" alt="logo"></header></body></html>`,
			wantCheck: "images/missing-lazy-loading",
			wantIssue: false,
		},
		// missing-dimensions checks
		{
			name:       "image missing both dimensions",
			html:       `<html><body><img src="photo.webp" alt="photo"></body></html>`,
			wantCheck:  "images/missing-dimensions",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "photo.webp",
		},
		{
			name:      "image with width only no issue",
			html:      `<html><body><img src="photo.webp" alt="photo" width="100"></body></html>`,
			wantCheck: "images/missing-dimensions",
			wantIssue: false,
		},
		{
			name:      "image with height only no issue",
			html:      `<html><body><img src="photo.webp" alt="photo" height="100"></body></html>`,
			wantCheck: "images/missing-dimensions",
			wantIssue: false,
		},
		{
			name:      "image with both dimensions no issue",
			html:      `<html><body><img src="photo.webp" alt="photo" width="100" height="100"></body></html>`,
			wantCheck: "images/missing-dimensions",
			wantIssue: false,
		},
		// missing-responsive checks
		{
			name:       "remote image missing srcset and sizes",
			html:       `<html><body><img src="https://cdn.example.com/photo.webp" alt="photo"></body></html>`,
			wantCheck:  "images/missing-responsive",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "srcset and sizes",
		},
		{
			name:      "remote image with srcset no issue",
			html:      `<html><body><img src="https://cdn.example.com/photo.webp" alt="photo" srcset="photo-2x.webp 2x"></body></html>`,
			wantCheck: "images/missing-responsive",
			wantIssue: false,
		},
		{
			name:      "remote image with sizes no issue",
			html:      `<html><body><img src="https://cdn.example.com/photo.webp" alt="photo" sizes="(max-width: 600px) 100vw"></body></html>`,
			wantCheck: "images/missing-responsive",
			wantIssue: false,
		},
		{
			name:      "local image skipped",
			html:      `<html><body><img src="/images/photo.webp" alt="photo"></body></html>`,
			wantCheck: "images/missing-responsive",
			wantIssue: false,
		},
		{
			name:      "decorative image skipped",
			html:      `<html><body><img src="https://cdn.example.com/spacer.gif" alt="" role="presentation"></body></html>`,
			wantCheck: "images/missing-responsive",
			wantIssue: false,
		},
		{
			name:       "remote image with alt but no role not skipped",
			html:       `<html><body><img src="https://cdn.example.com/photo.webp" alt="photo" role="presentation"></body></html>`,
			wantCheck:  "images/missing-responsive",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "srcset and sizes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			if tt.name == "non-html page skipped" {
				page.ContentType = "image/png"
			}

			issues := checker.Check(ctx, page)

			if !tt.wantIssue {
				for _, iss := range issues {
					if iss.CheckName == tt.wantCheck {
						t.Fatalf("did not expect issue %s", tt.wantCheck)
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

func TestImageChecker_BrokenSrc(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	checker := NewImageChecker()
	checker.allowPrivate = true
	checker.SetHTTPClient(ts.Client())

	page := htmlPage(fmt.Sprintf(`<html><body><img src="%s/broken.jpg" alt="broken"></body></html>`, ts.URL))
	issues := checker.Check(context.Background(), page)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "images/broken-src" {
			found = true
			if iss.Severity != model.SeverityCritical {
				t.Errorf("expected critical severity, got %s", iss.Severity)
			}
		}
	}
	if !found {
		t.Error("expected images/broken-src issue")
	}
}

func TestImageChecker_LargeImage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "600000")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	checker := NewImageChecker()
	checker.allowPrivate = true
	checker.SetHTTPClient(ts.Client())

	page := htmlPage(fmt.Sprintf(`<html><body><img src="%s/big.jpg" alt="big"></body></html>`, ts.URL))
	issues := checker.Check(context.Background(), page)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "images/large-image" {
			found = true
			if iss.Severity != model.SeverityWarning {
				t.Errorf("expected warning severity, got %s", iss.Severity)
			}
		}
	}
	if !found {
		t.Error("expected images/large-image issue")
	}
}

func TestImageChecker_SmallImage_NoIssue(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1024")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	checker := NewImageChecker()
	checker.allowPrivate = true
	checker.SetHTTPClient(ts.Client())

	page := htmlPage(fmt.Sprintf(`<html><body><img src="%s/small.jpg" alt="small"></body></html>`, ts.URL))
	issues := checker.Check(context.Background(), page)

	for _, iss := range issues {
		if iss.CheckName == "images/large-image" || iss.CheckName == "images/broken-src" {
			t.Fatalf("did not expect issue %s", iss.CheckName)
		}
	}
}
