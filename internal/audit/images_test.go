package audit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/model"
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
	checker.SetHTTPClient(ts.Client())

	page := htmlPage(fmt.Sprintf(`<html><body><img src="%s/small.jpg" alt="small"></body></html>`, ts.URL))
	issues := checker.Check(context.Background(), page)

	for _, iss := range issues {
		if iss.CheckName == "images/large-image" || iss.CheckName == "images/broken-src" {
			t.Fatalf("did not expect issue %s", iss.CheckName)
		}
	}
}
