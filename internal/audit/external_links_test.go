package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestExternalLinkChecker_Check_ReturnsNil(t *testing.T) {
	checker := NewExternalLinkChecker()
	page := &model.Page{URL: "https://example.com"}
	issues := checker.Check(context.Background(), page)
	if issues != nil {
		t.Fatalf("expected nil from Check, got %+v", issues)
	}
}

func TestExternalLinkChecker_Name(t *testing.T) {
	checker := NewExternalLinkChecker()
	if checker.Name() != "external-links" {
		t.Fatalf("expected name 'external-links', got %q", checker.Name())
	}
}

func TestExternalLinkChecker_CheckSite(t *testing.T) {
	// Set up mock servers.
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okServer.Close()

	brokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer brokenServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "https://destination.example.com/new")
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer redirectServer.Close()

	head405Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer head405Server.Close()

	head405BrokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer head405BrokenServer.Close()

	ctx := context.Background()

	tests := []struct {
		name       string
		pages      []*model.Page
		wantCheck  string
		wantIssue  bool
		wantSubstr string
	}{
		{
			name: "external link returns 200 no issue",
			pages: []*model.Page{
				{
					URL:        "https://seed.example.com",
					StatusCode: 200,
					Links:      []string{okServer.URL + "/page"},
				},
			},
			wantCheck: "external-links/broken",
			wantIssue: false,
		},
		{
			name: "broken external link",
			pages: []*model.Page{
				{
					URL:        "https://seed.example.com",
					StatusCode: 200,
					Links:      []string{brokenServer.URL + "/missing"},
				},
			},
			wantCheck:  "external-links/broken",
			wantIssue:  true,
			wantSubstr: "HTTP 404",
		},
		{
			name: "redirect external link",
			pages: []*model.Page{
				{
					URL:        "https://seed.example.com",
					StatusCode: 200,
					Links:      []string{redirectServer.URL + "/old"},
				},
			},
			wantCheck:  "external-links/redirect",
			wantIssue:  true,
			wantSubstr: "redirects",
		},
		{
			name: "HEAD 405 fallback to GET success",
			pages: []*model.Page{
				{
					URL:        "https://seed.example.com",
					StatusCode: 200,
					Links:      []string{head405Server.URL + "/page"},
				},
			},
			wantCheck: "external-links/broken",
			wantIssue: false,
		},
		{
			name: "HEAD 405 fallback to GET broken",
			pages: []*model.Page{
				{
					URL:        "https://seed.example.com",
					StatusCode: 200,
					Links:      []string{head405BrokenServer.URL + "/page"},
				},
			},
			wantCheck:  "external-links/broken",
			wantIssue:  true,
			wantSubstr: "HTTP 500",
		},
		{
			name: "internal link not checked",
			pages: []*model.Page{
				{
					URL:        "https://seed.example.com",
					StatusCode: 200,
					Links:      []string{"https://seed.example.com/about"},
				},
			},
			wantCheck: "external-links/broken",
			wantIssue: false,
		},
		{
			name: "external link from assets",
			pages: []*model.Page{
				{
					URL:        "https://seed.example.com",
					StatusCode: 200,
					Assets:     []string{brokenServer.URL + "/style.css"},
				},
			},
			wantCheck:  "external-links/broken",
			wantIssue:  true,
			wantSubstr: "HTTP 404",
		},
		{
			name:      "no pages returns nil",
			pages:     nil,
			wantCheck: "external-links/broken",
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewExternalLinkChecker()
			checker.allowPrivate = true

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

func TestExternalLinkChecker_Deduplication(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	checker := NewExternalLinkChecker()
	checker.allowPrivate = true

	pages := []*model.Page{
		{
			URL:        "https://seed.example.com",
			StatusCode: 200,
			Links:      []string{ts.URL + "/shared"},
		},
		{
			URL:        "https://seed.example.com/page2",
			StatusCode: 200,
			Links:      []string{ts.URL + "/shared"},
		},
	}

	issues := checker.CheckSite(context.Background(), pages)

	// Should only produce one broken issue for the deduplicated URL.
	brokenCount := 0
	for _, iss := range issues {
		if iss.CheckName == "external-links/broken" {
			brokenCount++
			// Detail should mention both source pages.
			if !strings.Contains(iss.Detail, "seed.example.com") {
				t.Errorf("expected detail to reference source pages, got %q", iss.Detail)
			}
		}
	}
	if brokenCount != 1 {
		t.Errorf("expected 1 broken issue for deduplicated URL, got %d", brokenCount)
	}

	// The server should have been called only once (or at most twice for
	// HEAD+GET fallback), not once per page that references the link.
	if callCount > 2 {
		t.Errorf("expected at most 2 requests (HEAD+GET fallback), got %d", callCount)
	}
}

func TestExternalLinkChecker_RedirectDetail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "https://new.example.com/page")
		w.WriteHeader(http.StatusFound)
	}))
	defer ts.Close()

	checker := NewExternalLinkChecker()
	checker.allowPrivate = true

	pages := []*model.Page{
		{
			URL:        "https://seed.example.com",
			StatusCode: 200,
			Links:      []string{ts.URL + "/old"},
		},
	}

	issues := checker.CheckSite(context.Background(), pages)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "external-links/redirect" {
			found = true
			if !strings.Contains(iss.Detail, "https://new.example.com/page") {
				t.Errorf("expected detail to contain redirect destination, got %q", iss.Detail)
			}
		}
	}
	if !found {
		t.Errorf("expected external-links/redirect issue, got %+v", issues)
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		rawURL string
		want   string
	}{
		{"https://example.com/page", "example.com"},
		{"https://Example.COM/Page", "example.com"},
		{"https://example.com:8080/page", "example.com"},
		{"http://sub.example.com", "sub.example.com"},
		{"not-a-url", ""},
	}

	for _, tt := range tests {
		got := extractHost(tt.rawURL)
		if got != tt.want {
			t.Errorf("extractHost(%q) = %q, want %q", tt.rawURL, got, tt.want)
		}
	}
}

func TestIsExternalLink(t *testing.T) {
	tests := []struct {
		link     string
		seedHost string
		want     bool
	}{
		{"https://other.com/page", "example.com", true},
		{"https://example.com/page", "example.com", false},
		{"https://other.com/about", "example.com", true},
		{"/relative/path", "example.com", false},
		{"mailto:test@example.com", "example.com", false},
		{"https://sub.example.com/page", "example.com", true},
	}

	for _, tt := range tests {
		got := isExternalLink(tt.link, tt.seedHost)
		if got != tt.want {
			t.Errorf("isExternalLink(%q, %q) = %v, want %v", tt.link, tt.seedHost, got, tt.want)
		}
	}
}

func TestDedupStrings(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b"}
	got := dedupStrings(input)
	want := []string{"a", "b", "c"}

	if len(got) != len(want) {
		t.Fatalf("dedupStrings returned %v, want %v", got, want)
	}
	for i, v := range got {
		if v != want[i] {
			t.Errorf("dedupStrings[%d] = %q, want %q", i, v, want[i])
		}
	}
}
