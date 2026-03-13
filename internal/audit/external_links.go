package audit

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/meysam81/scry/internal/model"
	"github.com/meysam81/scry/internal/safenet"
	"golang.org/x/time/rate"
)

const (
	externalLinkTimeout     = 10 * time.Second
	externalLinkConcurrency = 5
	externalLinkRatePerSec  = 2
)

// ExternalLinkChecker validates external links found across all crawled pages.
// It implements SiteChecker because all work happens in CheckSite where
// external URLs are deduplicated and checked concurrently.
type ExternalLinkChecker struct {
	client       *http.Client
	allowPrivate bool // skip SSRF checks (for testing only)
}

// NewExternalLinkChecker returns a new ExternalLinkChecker.
func NewExternalLinkChecker() *ExternalLinkChecker {
	return &ExternalLinkChecker{
		client: &http.Client{
			Timeout: externalLinkTimeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// SetHTTPClient sets the HTTP client used for external link checks.
func (c *ExternalLinkChecker) SetHTTPClient(client *http.Client) {
	c.client = client
}

// Name returns the checker name.
func (c *ExternalLinkChecker) Name() string { return "external-links" }

// Check returns nil. All work is done in CheckSite.
func (c *ExternalLinkChecker) Check(_ context.Context, _ *model.Page) []model.Issue {
	return nil
}

// externalLinkResult holds the outcome of checking a single external URL.
type externalLinkResult struct {
	url        string
	statusCode int
	location   string // redirect destination if applicable
	timeout    bool
	err        error
}

// CheckSite collects all external links from every page, deduplicates them,
// and checks each one concurrently with per-domain rate limiting.
func (c *ExternalLinkChecker) CheckSite(ctx context.Context, pages []*model.Page) []model.Issue {
	if len(pages) == 0 {
		return nil
	}

	seedHost := extractHost(pages[0].URL)

	// Collect external URLs and track which pages reference each.
	urlPages := make(map[string][]string) // external URL -> list of page URLs
	for _, p := range pages {
		allLinks := make([]string, 0, len(p.Links)+len(p.Assets))
		allLinks = append(allLinks, p.Links...)
		allLinks = append(allLinks, p.Assets...)

		for _, link := range allLinks {
			if !isExternalLink(link, seedHost) {
				continue
			}
			urlPages[link] = append(urlPages[link], p.URL)
		}
	}

	if len(urlPages) == 0 {
		return nil
	}

	// Build work queue of unique external URLs.
	urls := make([]string, 0, len(urlPages))
	for u := range urlPages {
		urls = append(urls, u)
	}

	// Per-domain rate limiters.
	var limiterMu sync.Mutex
	limiters := make(map[string]*rate.Limiter)
	getLimiter := func(host string) *rate.Limiter {
		limiterMu.Lock()
		defer limiterMu.Unlock()
		if l, ok := limiters[host]; ok {
			return l
		}
		l := rate.NewLimiter(rate.Limit(externalLinkRatePerSec), 1)
		limiters[host] = l
		return l
	}

	// Check external URLs concurrently.
	type resultEntry struct {
		url    string
		result externalLinkResult
	}

	results := make([]resultEntry, len(urls))
	var wg sync.WaitGroup
	sem := make(chan struct{}, externalLinkConcurrency)

	for i, u := range urls {
		wg.Add(1)
		go func(idx int, targetURL string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				return
			}

			// SSRF protection.
			if !c.allowPrivate && !safenet.IsSafeURL(targetURL) {
				return
			}

			// Per-domain rate limiting.
			host := extractHost(targetURL)
			limiter := getLimiter(host)
			if err := limiter.Wait(ctx); err != nil {
				return
			}

			res := c.checkExternalURL(ctx, targetURL)
			results[idx] = resultEntry{url: targetURL, result: res}
		}(i, u)
	}
	wg.Wait()

	// Build issues from results.
	var issues []model.Issue
	for _, entry := range results {
		if entry.url == "" {
			continue // skipped (SSRF or context cancelled)
		}
		res := entry.result
		sources := urlPages[entry.url]
		detail := "linked from: " + strings.Join(dedupStrings(sources), ", ")

		if res.timeout {
			issues = append(issues, model.Issue{
				CheckName: "external-links/timeout",
				Severity:  model.SeverityInfo,
				URL:       entry.url,
				Message:   "external link timed out",
				Detail:    detail,
			})
			continue
		}

		if res.err != nil {
			issues = append(issues, model.Issue{
				CheckName: "external-links/broken",
				Severity:  model.SeverityWarning,
				URL:       entry.url,
				Message:   fmt.Sprintf("external link returned error: %v", res.err),
				Detail:    detail,
			})
			continue
		}

		if res.statusCode >= 300 && res.statusCode < 400 && res.location != "" {
			issues = append(issues, model.Issue{
				CheckName: "external-links/redirect",
				Severity:  model.SeverityInfo,
				URL:       entry.url,
				Message:   fmt.Sprintf("external link redirects (HTTP %d)", res.statusCode),
				Detail:    fmt.Sprintf("redirects to: %s; %s", res.location, detail),
			})
			continue
		}

		if res.statusCode >= 400 {
			issues = append(issues, model.Issue{
				CheckName: "external-links/broken",
				Severity:  model.SeverityWarning,
				URL:       entry.url,
				Message:   fmt.Sprintf("external link returned HTTP %d", res.statusCode),
				Detail:    detail,
			})
		}
	}

	return issues
}

// checkExternalURL performs a HEAD request (falling back to GET on 405) and
// returns the result.
func (c *ExternalLinkChecker) checkExternalURL(ctx context.Context, targetURL string) externalLinkResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, targetURL, nil)
	if err != nil {
		return externalLinkResult{url: targetURL, err: err}
	}
	req.Header.Set("User-Agent", "scry-link-checker/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		if isTimeoutError(err) {
			return externalLinkResult{url: targetURL, timeout: true}
		}
		return externalLinkResult{url: targetURL, err: err}
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			auditLogger.Warn().Err(cerr).Str("url", targetURL).Msg("resp body close failed")
		}
	}()

	// Fall back to GET if HEAD is not allowed.
	if resp.StatusCode == http.StatusMethodNotAllowed {
		req2, err2 := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err2 != nil {
			return externalLinkResult{url: targetURL, err: err2}
		}
		req2.Header.Set("User-Agent", "scry-link-checker/1.0")

		resp2, err2 := c.client.Do(req2)
		if err2 != nil {
			if isTimeoutError(err2) {
				return externalLinkResult{url: targetURL, timeout: true}
			}
			return externalLinkResult{url: targetURL, err: err2}
		}
		defer func() {
			if cerr := resp2.Body.Close(); cerr != nil {
				auditLogger.Warn().Err(cerr).Str("url", targetURL).Msg("resp body close failed")
			}
		}()

		return externalLinkResult{
			url:        targetURL,
			statusCode: resp2.StatusCode,
			location:   resp2.Header.Get("Location"),
		}
	}

	return externalLinkResult{
		url:        targetURL,
		statusCode: resp.StatusCode,
		location:   resp.Header.Get("Location"),
	}
}

// extractHost returns the hostname (without port) from a URL string.
func extractHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

// isExternalLink returns true if the link's host differs from seedHost.
func isExternalLink(link, seedHost string) bool {
	if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
		return false
	}
	linkHost := extractHost(link)
	return linkHost != "" && linkHost != seedHost
}

// isTimeoutError checks whether an error is a timeout.
func isTimeoutError(err error) bool {
	type timeouter interface {
		Timeout() bool
	}
	if te, ok := err.(timeouter); ok {
		return te.Timeout()
	}
	return strings.Contains(err.Error(), "timeout")
}

// dedupStrings returns a deduplicated copy of ss preserving order.
func dedupStrings(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
