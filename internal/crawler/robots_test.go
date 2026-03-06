package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
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

	rc := NewRobotsChecker("scry")

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

	rc := NewRobotsChecker("scry")

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

	rc := NewRobotsChecker("scry")

	if !rc.IsAllowed(context.Background(), srv.URL+"/anything") {
		t.Error("missing robots.txt should allow all URLs")
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

	rc := NewRobotsChecker("scry")

	if rc.IsAllowed(context.Background(), srv.URL+"/scry-blocked") {
		t.Error("/scry-blocked should be disallowed for scry agent")
	}
	// When specific rules are found, wildcard rules are not used.
	if !rc.IsAllowed(context.Background(), srv.URL+"/all-blocked") {
		t.Error("/all-blocked should be allowed for scry agent (specific rules take precedence)")
	}
}
