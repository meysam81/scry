package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/logger"
)

// newTestConfig returns a config suitable for testing.
func newTestConfig() *config.Config {
	return &config.Config{
		MaxDepth:       5,
		MaxPages:       500,
		Concurrency:    2,
		RequestTimeout: 5 * time.Second,
		RateLimit:      100,
		UserAgent:      "test-crawler/1.0",
		RespectRobots:  false,
	}
}

func TestCrawler_BasicCrawl(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprintf(w, `<html><body>
				<a href="/page1">P1</a>
				<a href="/page2">P2</a>
			</body></html>`)
		case "/page1":
			_, _ = fmt.Fprint(w, `<html><body><a href="/page2">P2</a></body></html>`)
		case "/page2":
			_, _ = fmt.Fprint(w, `<html><body>end</body></html>`)
		case "/sitemap.xml":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cfg := newTestConfig()
	fetcher := NewHTTPFetcher(cfg.UserAgent, cfg.RequestTimeout, logger.Nop())
	c := NewCrawler(cfg, fetcher, logger.Nop())

	result, err := c.Run(context.Background(), srv.URL+"/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Pages) < 3 {
		t.Errorf("got %d pages, want at least 3", len(result.Pages))
	}
	if result.SeedURL != srv.URL+"/" {
		t.Errorf("seed URL = %q, want %q", result.SeedURL, srv.URL+"/")
	}
	if result.Duration <= 0 {
		t.Error("duration should be positive")
	}
}

func TestCrawler_MaxDepth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprint(w, `<html><body><a href="/depth1">D1</a></body></html>`)
		case "/depth1":
			_, _ = fmt.Fprint(w, `<html><body><a href="/depth2">D2</a></body></html>`)
		case "/depth2":
			_, _ = fmt.Fprint(w, `<html><body><a href="/depth3">D3</a></body></html>`)
		case "/depth3":
			_, _ = fmt.Fprint(w, `<html><body>too deep</body></html>`)
		case "/sitemap.xml":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cfg := newTestConfig()
	cfg.MaxDepth = 2
	fetcher := NewHTTPFetcher(cfg.UserAgent, cfg.RequestTimeout, logger.Nop())
	c := NewCrawler(cfg, fetcher, logger.Nop())

	result, err := c.Run(context.Background(), srv.URL+"/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// At depth=2, we should have /, /depth1, /depth2 but not /depth3.
	for _, p := range result.Pages {
		if p.Depth > 2 {
			t.Errorf("found page at depth %d (%s), but maxDepth=2", p.Depth, p.URL)
		}
	}

	// Verify /depth3 was not crawled.
	for _, p := range result.Pages {
		if p.URL == srv.URL+"/depth3" {
			t.Error("/depth3 should not be crawled with maxDepth=2")
		}
	}
}

func TestCrawler_MaxPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprint(w, `<html><body>
				<a href="/p1">1</a>
				<a href="/p2">2</a>
				<a href="/p3">3</a>
				<a href="/p4">4</a>
				<a href="/p5">5</a>
			</body></html>`)
		case "/sitemap.xml":
			w.WriteHeader(http.StatusNotFound)
		default:
			_, _ = fmt.Fprint(w, `<html><body>page</body></html>`)
		}
	}))
	defer srv.Close()

	cfg := newTestConfig()
	cfg.MaxPages = 3
	fetcher := NewHTTPFetcher(cfg.UserAgent, cfg.RequestTimeout, logger.Nop())
	c := NewCrawler(cfg, fetcher, logger.Nop())

	result, err := c.Run(context.Background(), srv.URL+"/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Pages) > 3 {
		t.Errorf("got %d pages, want at most 3 (maxPages=3)", len(result.Pages))
	}
}

func TestCrawler_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			// Generate many links to keep the crawler busy.
			_, _ = fmt.Fprint(w, `<html><body>`)
			for i := range 100 {
				_, _ = fmt.Fprintf(w, `<a href="/page%d">P%d</a>`, i, i)
			}
			_, _ = fmt.Fprint(w, `</body></html>`)
		case "/sitemap.xml":
			w.WriteHeader(http.StatusNotFound)
		default:
			time.Sleep(100 * time.Millisecond)
			_, _ = fmt.Fprint(w, `<html><body>page</body></html>`)
		}
	}))
	defer srv.Close()

	cfg := newTestConfig()
	cfg.RateLimit = 1000 // Don't bottleneck on rate.
	fetcher := NewHTTPFetcher(cfg.UserAgent, cfg.RequestTimeout, logger.Nop())
	c := NewCrawler(cfg, fetcher, logger.Nop())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	result, err := c.Run(ctx, srv.URL+"/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not have crawled all 100 pages.
	if len(result.Pages) >= 100 {
		t.Errorf("got %d pages; expected fewer due to context cancellation", len(result.Pages))
	}
}

func TestCrawler_SitemapSeeding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		switch r.URL.Path {
		case "/":
			_, _ = fmt.Fprint(w, `<html><body>home</body></html>`)
		case "/sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>http://%s/from-sitemap</loc></url>
</urlset>`, r.Host)
		case "/from-sitemap":
			_, _ = fmt.Fprint(w, `<html><body>from sitemap</body></html>`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cfg := newTestConfig()
	fetcher := NewHTTPFetcher(cfg.UserAgent, cfg.RequestTimeout, logger.Nop())
	c := NewCrawler(cfg, fetcher, logger.Nop())

	result, err := c.Run(context.Background(), srv.URL+"/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundSitemapPage := false
	for _, p := range result.Pages {
		if p.URL == srv.URL+"/from-sitemap" {
			foundSitemapPage = true
			if !p.InSitemap {
				t.Error("/from-sitemap should have InSitemap=true")
			}
		}
	}
	if !foundSitemapPage {
		t.Error("page /from-sitemap should be crawled via sitemap seeding")
	}
}
