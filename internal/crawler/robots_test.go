package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/logger"
)

func TestRobotsChecker_DisallowBlocks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = fmt.Fprint(w, "User-agent: *\nDisallow: /admin\nDisallow: /private/\n")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rc := NewRobotsChecker("scry", logger.Nop())

	if rc.IsAllowed(context.Background(), srv.URL+"/admin") {
		t.Error("/admin should be disallowed")
	}
	if rc.IsAllowed(context.Background(), srv.URL+"/admin/settings") {
		t.Error("/admin/settings should be disallowed")
	}
	if rc.IsAllowed(context.Background(), srv.URL+"/private/data") {
		t.Error("/private/data should be disallowed")
	}
	if !rc.IsAllowed(context.Background(), srv.URL+"/public") {
		t.Error("/public should be allowed")
	}
}

func TestRobotsChecker_AllowOverridesDisallow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = fmt.Fprint(w, "User-agent: *\nDisallow: /docs\nAllow: /docs/public\n")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rc := NewRobotsChecker("scry", logger.Nop())

	if rc.IsAllowed(context.Background(), srv.URL+"/docs/secret") {
		t.Error("/docs/secret should be disallowed")
	}
	if !rc.IsAllowed(context.Background(), srv.URL+"/docs/public") {
		t.Error("/docs/public should be allowed (Allow overrides Disallow)")
	}
	if !rc.IsAllowed(context.Background(), srv.URL+"/docs/public/page") {
		t.Error("/docs/public/page should be allowed")
	}
}

func TestRobotsChecker_MissingRobotsTxt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rc := NewRobotsChecker("scry", logger.Nop())

	if !rc.IsAllowed(context.Background(), srv.URL+"/anything") {
		t.Error("missing robots.txt should allow all URLs")
	}
}

func TestRobotsChecker_MultiUAGroupMatchFirst(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = fmt.Fprint(w, "User-agent: scry\nUser-agent: googlebot\nDisallow: /shared-blocked\n")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rc := NewRobotsChecker("scry/1.0", logger.Nop())

	if rc.IsAllowed(context.Background(), srv.URL+"/shared-blocked") {
		t.Error("/shared-blocked should be disallowed for scry in multi-UA group (match first)")
	}
	if !rc.IsAllowed(context.Background(), srv.URL+"/other") {
		t.Error("/other should be allowed")
	}
}

func TestRobotsChecker_MultiUAGroupMatchLast(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = fmt.Fprint(w, "User-agent: googlebot\nUser-agent: scry\nDisallow: /shared-blocked\n")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rc := NewRobotsChecker("scry/1.0", logger.Nop())

	if rc.IsAllowed(context.Background(), srv.URL+"/shared-blocked") {
		t.Error("/shared-blocked should be disallowed for scry in multi-UA group (match last)")
	}
	if !rc.IsAllowed(context.Background(), srv.URL+"/other") {
		t.Error("/other should be allowed")
	}
}

func TestRobotsChecker_SpecificUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = fmt.Fprint(w, "User-agent: scry\nDisallow: /scry-blocked\n\nUser-agent: *\nDisallow: /all-blocked\n")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rc := NewRobotsChecker("scry", logger.Nop())

	if rc.IsAllowed(context.Background(), srv.URL+"/scry-blocked") {
		t.Error("/scry-blocked should be disallowed for scry agent")
	}
	// When specific rules are found, wildcard rules are not used.
	if !rc.IsAllowed(context.Background(), srv.URL+"/all-blocked") {
		t.Error("/all-blocked should be allowed for scry agent (specific rules take precedence)")
	}
}

func TestRobotsChecker_WildcardStar(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = fmt.Fprint(w, "User-agent: *\nDisallow: /*.pdf\nDisallow: /admin*\n")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rc := NewRobotsChecker("scry", logger.Nop())

	tests := []struct {
		path    string
		allowed bool
	}{
		{"/docs/file.pdf", false},
		{"/docs/file.pdf?v=1", false}, // no $ anchor, so query string still matches
		{"/file.pdf", false},
		{"/docs/file.txt", true},
		{"/admin", false},
		{"/admin/", false},
		{"/admin/page", false},
		{"/public", true},
	}

	for _, tt := range tests {
		got := rc.IsAllowed(context.Background(), srv.URL+tt.path)
		if got != tt.allowed {
			t.Errorf("path %q: got allowed=%v, want %v", tt.path, got, tt.allowed)
		}
	}
}

func TestRobotsChecker_EndAnchor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = fmt.Fprint(w, "User-agent: *\nDisallow: /*.pdf$\n")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rc := NewRobotsChecker("scry", logger.Nop())

	tests := []struct {
		path    string
		allowed bool
	}{
		{"/docs/file.pdf", false},
		{"/docs/file.pdf?v=1", true}, // $ means must end with .pdf
		{"/file.pdf", false},
		{"/docs/file.txt", true},
	}

	for _, tt := range tests {
		got := rc.IsAllowed(context.Background(), srv.URL+tt.path)
		if got != tt.allowed {
			t.Errorf("path %q: got allowed=%v, want %v", tt.path, got, tt.allowed)
		}
	}
}

func TestRobotsChecker_WildcardWithEndAnchor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = fmt.Fprint(w, "User-agent: *\nDisallow: /private/*/secret$\n")
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	rc := NewRobotsChecker("scry", logger.Nop())

	tests := []struct {
		path    string
		allowed bool
	}{
		{"/private/foo/secret", false},
		{"/private/bar/baz/secret", false},
		{"/private/foo/secret/more", true}, // $ anchor prevents this match
		{"/private/foo/public", true},
	}

	for _, tt := range tests {
		got := rc.IsAllowed(context.Background(), srv.URL+tt.path)
		if got != tt.allowed {
			t.Errorf("path %q: got allowed=%v, want %v", tt.path, got, tt.allowed)
		}
	}
}

func TestRobotsChecker_PlainPrefixStillWorks(t *testing.T) {
	// Ensure patterns without wildcards continue to use prefix matching.
	rules := parseRobotsTxt(strings.NewReader("User-agent: *\nDisallow: /admin\nAllow: /admin/public\n"), "scry")

	tests := []struct {
		path    string
		allowed bool
	}{
		{"/admin", false},
		{"/admin/settings", false},
		{"/admin/public", true},
		{"/admin/public/page", true},
		{"/other", true},
	}

	for _, tt := range tests {
		got := rules.isAllowed(tt.path)
		if got != tt.allowed {
			t.Errorf("path %q: got allowed=%v, want %v", tt.path, got, tt.allowed)
		}
	}
}
