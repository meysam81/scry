package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meysam81/scry/internal/logger"
)

func TestFetcher_BasicFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body><h1>hello browser</h1></body></html>"))
	}))
	defer srv.Close()

	bf, err := NewFetcher("", "test-browser-agent/1.0", 30*time.Second, logger.Nop())
	if err != nil {
		t.Fatalf("failed to create Fetcher: %v", err)
	}
	defer func() { _ = bf.Close() }()

	page, err := bf.Fetch(context.Background(), srv.URL+"/page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := string(page.Body)
	if !strings.Contains(body, "hello browser") {
		t.Errorf("body does not contain expected text; got %q", body)
	}
}

func TestFetcher_StatusCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>ok</body></html>"))
	}))
	defer srv.Close()

	bf, err := NewFetcher("", "test-agent/1.0", 30*time.Second, logger.Nop())
	if err != nil {
		t.Fatalf("failed to create Fetcher: %v", err)
	}
	defer func() { _ = bf.Close() }()

	page, err := bf.Fetch(context.Background(), srv.URL+"/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want %d", page.StatusCode, http.StatusOK)
	}
}

func TestFetcher_FetchDuration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>timing</body></html>"))
	}))
	defer srv.Close()

	bf, err := NewFetcher("", "test-agent/1.0", 30*time.Second, logger.Nop())
	if err != nil {
		t.Fatalf("failed to create Fetcher: %v", err)
	}
	defer func() { _ = bf.Close() }()

	page, err := bf.Fetch(context.Background(), srv.URL+"/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page.FetchDuration <= 0 {
		t.Error("fetch duration should be positive")
	}
	if page.FetchedAt.IsZero() {
		t.Error("fetched at should not be zero")
	}
}

func TestFetcher_ContentType(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>content type check</body></html>"))
	}))
	defer srv.Close()

	bf, err := NewFetcher("", "test-agent/1.0", 30*time.Second, logger.Nop())
	if err != nil {
		t.Fatalf("failed to create Fetcher: %v", err)
	}
	defer func() { _ = bf.Close() }()

	page, err := bf.Fetch(context.Background(), srv.URL+"/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page.ContentType != defaultContentType {
		t.Errorf("content type = %q, want %q", page.ContentType, defaultContentType)
	}
}
