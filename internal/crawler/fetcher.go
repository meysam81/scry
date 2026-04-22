// Package crawler implements a concurrent web crawler with BFS traversal,
// robots.txt compliance, sitemap parsing, and HTML link extraction.
package crawler

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/meysam81/scry/core/model"
	"github.com/meysam81/scry/internal/browser"
	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/internal/safenet"
)

// maxBodySize caps the response body read to prevent OOM (10 MB).
const maxBodySize = 10 * 1024 * 1024

// maxRedirects is the maximum number of redirects to follow before giving up.
// RFC 7231 recommends clients handle "a limited number of redirections"
// (typically 5–10). We use 10 as a reasonable upper bound.
const maxRedirects = 10

// ErrRedirectLoop is returned when a redirect cycle is detected.
var ErrRedirectLoop = errors.New("redirect loop detected")

// Fetcher fetches a URL and returns a Page.
type Fetcher interface {
	Fetch(ctx context.Context, url string) (*model.Page, error)
}

// HTTPFetcher implements Fetcher using net/http.
type HTTPFetcher struct {
	userAgent    string
	timeout      time.Duration
	transport    *http.Transport
	log          logger.Logger
	allowPrivate bool // skip SSRF checks (for testing only)
}

// NewHTTPFetcher creates a new HTTPFetcher with the given user agent and per-request timeout.
func NewHTTPFetcher(userAgent string, timeout time.Duration, l logger.Logger) *HTTPFetcher {
	return &HTTPFetcher{
		userAgent: userAgent,
		timeout:   timeout,
		log:       l,
		transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true,
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
	req.Header.Set("Accept-Encoding", "gzip")

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
			if !f.allowPrivate && !safenet.IsSafeURL(target) {
				return fmt.Errorf("redirect to unsafe URL: %s", target)
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
	defer func() {
		if err := resp.Body.Close(); err != nil {
			f.log.Warn().Err(err).Str("url", rawURL).Msg("resp body close failed")
		}
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("fetch %s: read body: %w", rawURL, err)
	}

	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gr, gzErr := gzip.NewReader(bytes.NewReader(body))
		if gzErr != nil {
			return nil, fmt.Errorf("fetch %s: gzip init: %w", rawURL, gzErr)
		}
		body, err = io.ReadAll(io.LimitReader(gr, maxBodySize))
		if cerr := gr.Close(); cerr != nil && err == nil {
			err = cerr
		}
		if err != nil {
			return nil, fmt.Errorf("fetch %s: gzip decompress: %w", rawURL, err)
		}
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
func NewFetcher(cfg *config.Config, l logger.Logger) (Fetcher, func(), error) {
	if cfg.BrowserMode {
		bf, err := browser.NewFetcher(cfg.BrowserlessURL, cfg.UserAgent, cfg.RequestTimeout, l)
		if err != nil {
			return nil, nil, fmt.Errorf("create browser fetcher: %w", err)
		}
		closer := func() {
			if err := bf.Close(); err != nil {
				l.Warn().Err(err).Msg("browser fetcher close failed")
			}
		}
		return bf, closer, nil
	}

	f := NewHTTPFetcher(cfg.UserAgent, cfg.RequestTimeout, l)
	noop := func() {}
	return f, noop, nil
}
