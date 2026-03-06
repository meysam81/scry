// Package crawler implements a concurrent web crawler with BFS traversal,
// robots.txt compliance, sitemap parsing, and HTML link extraction.
package crawler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/meysam81/scry/internal/browser"
	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/model"
)

// maxBodySize caps the response body read to prevent OOM (10 MB).
const maxBodySize = 10 * 1024 * 1024

// maxRedirects is the maximum number of redirects to follow before giving up.
const maxRedirects = 20

// ErrRedirectLoop is returned when a redirect cycle is detected.
var ErrRedirectLoop = errors.New("redirect loop detected")

// Fetcher fetches a URL and returns a Page.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (*model.Page, error)
}

// HTTPFetcher implements Fetcher using net/http.
type HTTPFetcher struct {
	userAgent string
	timeout   time.Duration
	transport *http.Transport
}

// NewHTTPFetcher creates a new HTTPFetcher with the given user agent and per-request timeout.
func NewHTTPFetcher(userAgent string, timeout time.Duration) *HTTPFetcher {
	return &HTTPFetcher{
		userAgent: userAgent,
		timeout:   timeout,
		transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// Fetch retrieves the given URL and returns a populated Page.
func (f *HTTPFetcher) Fetch(ctx context.Context, rawURL string) (*model.Page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	req.Header.Set("User-Agent", f.userAgent)

	var redirectChain []string
	seen := make(map[string]bool)

	client := &http.Client{
		Transport: f.transport,
		Timeout:   f.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			target := req.URL.String()
			if seen[target] {
				return ErrRedirectLoop
			}
			if len(via) >= maxRedirects {
				return ErrRedirectLoop
			}
			seen[target] = true
			redirectChain = append(redirectChain, target)
			return nil
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("fetch %s: read body: %w", rawURL, err)
	}
	duration := time.Since(start)

	page := &model.Page{
		URL:           resp.Request.URL.String(),
		StatusCode:    resp.StatusCode,
		ContentType:   parseContentType(resp.Header.Get("Content-Type")),
		Body:          body,
		Headers:       resp.Header,
		RedirectChain: redirectChain,
		FetchedAt:     start,
		FetchDuration: duration,
	}

	return page, nil
}

// parseContentType extracts the MIME type without parameters.
func parseContentType(ct string) string {
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = ct[:idx]
	}
	return strings.TrimSpace(ct)
}

// NewFetcher creates the appropriate Fetcher based on configuration.
// It returns the fetcher, a cleanup function that should be called when done,
// and any error encountered during creation.
func NewFetcher(cfg *config.Config) (Fetcher, func(), error) {
	if cfg.BrowserMode {
		bf, err := browser.NewBrowserFetcher(cfg.BrowserlessURL, cfg.UserAgent, cfg.RequestTimeout)
		if err != nil {
			return nil, nil, fmt.Errorf("create browser fetcher: %w", err)
		}
		closer := func() { _ = bf.Close() }
		return bf, closer, nil
	}

	f := NewHTTPFetcher(cfg.UserAgent, cfg.RequestTimeout)
	noop := func() {}
	return f, noop, nil
}
