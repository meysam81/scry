package crawler

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/logger"
)

func TestHTTPFetcher_BasicFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>hello</body></html>"))
	}))
	defer srv.Close()

	f := NewHTTPFetcher("test-agent/1.0", 5*time.Second, logger.Nop())
	page, err := f.Fetch(context.Background(), srv.URL+"/page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if page.StatusCode != 200 {
		t.Errorf("status = %d, want 200", page.StatusCode)
	}
	if page.ContentType != "text/html" {
		t.Errorf("content type = %q, want %q", page.ContentType, "text/html")
	}
	if string(page.Body) != "<html><body>hello</body></html>" {
		t.Errorf("body = %q, want %q", string(page.Body), "<html><body>hello</body></html>")
	}
	if page.FetchDuration <= 0 {
		t.Error("fetch duration should be positive")
	}
	if page.FetchedAt.IsZero() {
		t.Error("fetched at should not be zero")
	}
}

func TestHTTPFetcher_RedirectChain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a":
			http.Redirect(w, r, "/b", http.StatusFound)
		case "/b":
			http.Redirect(w, r, "/c", http.StatusFound)
		case "/c":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("final"))
		}
	}))
	defer srv.Close()

	f := NewHTTPFetcher("test-agent/1.0", 5*time.Second, logger.Nop())
	f.allowPrivate = true
	page, err := f.Fetch(context.Background(), srv.URL+"/a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(page.RedirectChain) != 2 {
		t.Fatalf("redirect chain length = %d, want 2; chain = %v", len(page.RedirectChain), page.RedirectChain)
	}
	if page.StatusCode != 200 {
		t.Errorf("final status = %d, want 200", page.StatusCode)
	}
	if string(page.Body) != "final" {
		t.Errorf("body = %q, want %q", string(page.Body), "final")
	}
}

func TestHTTPFetcher_RedirectLoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a":
			http.Redirect(w, r, "/b", http.StatusFound)
		case "/b":
			http.Redirect(w, r, "/a", http.StatusFound)
		}
	}))
	defer srv.Close()

	f := NewHTTPFetcher("test-agent/1.0", 5*time.Second, logger.Nop())
	f.allowPrivate = true
	_, err := f.Fetch(context.Background(), srv.URL+"/a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrRedirectLoop) {
		t.Errorf("expected ErrRedirectLoop, got %v", err)
	}
}

func TestHTTPFetcher_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewHTTPFetcher("test-agent/1.0", 10*time.Second, logger.Nop())
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := f.Fetch(ctx, srv.URL+"/slow")
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestHTTPFetcher_UserAgent(t *testing.T) {
	var receivedUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewHTTPFetcher("my-custom-agent/2.0", 5*time.Second, logger.Nop())
	_, err := f.Fetch(context.Background(), srv.URL+"/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedUA != "my-custom-agent/2.0" {
		t.Errorf("user agent = %q, want %q", receivedUA, "my-custom-agent/2.0")
	}
}

func TestHTTPFetcher_GzipDecompression(t *testing.T) {
	const want = "<html><body>compressed</body></html>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept-Encoding") != "gzip" {
			t.Error("expected Accept-Encoding: gzip header")
		}
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, _ = gz.Write([]byte(want))
		_ = gz.Close() // writing to in-memory buffer; never fails

		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	f := NewHTTPFetcher("test-agent/1.0", 5*time.Second, logger.Nop())
	page, err := f.Fetch(context.Background(), srv.URL+"/gzip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(page.Body) != want {
		t.Errorf("body = %q, want %q", string(page.Body), want)
	}
}

func TestNewFetcher_HTTPMode(t *testing.T) {
	cfg := &config.Config{
		BrowserMode:    false,
		UserAgent:      "test-agent/1.0",
		RequestTimeout: 5 * time.Second,
	}

	fetcher, closer, err := NewFetcher(cfg, logger.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer closer()

	if fetcher == nil {
		t.Fatal("expected non-nil fetcher")
	}

	// Verify it works by fetching from a test server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>factory test</body></html>"))
	}))
	defer srv.Close()

	page, err := fetcher.Fetch(context.Background(), srv.URL+"/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if page.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", page.StatusCode, http.StatusOK)
	}
	if string(page.Body) != "<html><body>factory test</body></html>" {
		t.Errorf("body = %q, want %q", string(page.Body), "<html><body>factory test</body></html>")
	}
}

func TestNewFetcher_BrowserModeFailsWithoutBrowser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping browser test in short mode")
	}

	cfg := &config.Config{
		BrowserMode:    true,
		BrowserlessURL: "http://localhost:99999", // invalid port, should fail
		UserAgent:      "test-agent/1.0",
		RequestTimeout: 5 * time.Second,
	}

	_, _, err := NewFetcher(cfg, logger.Nop())
	if err == nil {
		t.Fatal("expected error when connecting to invalid browserless URL")
	}
}
