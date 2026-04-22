package checks

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/meysam81/scry/core/model"
)

var versionRe = regexp.MustCompile(`\d+\.\d+`)

const (
	maxRedirectHops = 2
	maxTTFB         = 2 * time.Second
)

// HealthChecker analyses pages for HTTP health issues.
type HealthChecker struct{}

// NewHealthChecker returns a new HealthChecker.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{}
}

// Name returns the checker name.
func (c *HealthChecker) Name() string { return "health" }

// Check runs per-page health checks.
func (c *HealthChecker) Check(_ context.Context, page *model.Page) []model.Issue {
	var issues []model.Issue

	issues = append(issues, c.checkStatusCode(page)...)
	issues = append(issues, c.checkRedirects(page)...)
	issues = append(issues, c.checkTTFB(page)...)
	issues = append(issues, c.checkMixedContent(page)...)
	issues = append(issues, c.checkServerVersionLeak(page)...)
	issues = append(issues, c.checkHTTPSRedirectNotPermanent(page)...)
	issues = append(issues, c.checkMissingCharset(page)...)

	return issues
}

func (c *HealthChecker) checkStatusCode(page *model.Page) []model.Issue {
	switch {
	case page.StatusCode >= 400 && page.StatusCode <= 499:
		return []model.Issue{{
			CheckName: "health/4xx",
			Severity:  model.SeverityCritical,
			URL:       page.URL,
			Message:   fmt.Sprintf("page returned HTTP %d", page.StatusCode),
		}}
	case page.StatusCode >= 500 && page.StatusCode <= 599:
		return []model.Issue{{
			CheckName: "health/5xx",
			Severity:  model.SeverityCritical,
			URL:       page.URL,
			Message:   fmt.Sprintf("page returned HTTP %d", page.StatusCode),
		}}
	}
	return nil
}

func (c *HealthChecker) checkRedirects(page *model.Page) []model.Issue {
	var issues []model.Issue

	if len(page.RedirectChain) > maxRedirectHops {
		issues = append(issues, model.Issue{
			CheckName: "health/redirect-chain",
			Severity:  model.SeverityWarning,
			URL:       page.URL,
			Message:   fmt.Sprintf("redirect chain has %d hops, maximum is %d", len(page.RedirectChain), maxRedirectHops),
		})
	}

	if slices.Contains(page.RedirectChain, page.URL) {
		issues = append(issues, model.Issue{
			CheckName: "health/redirect-loop",
			Severity:  model.SeverityCritical,
			URL:       page.URL,
			Message:   "page URL appears in its own redirect chain",
		})
	}

	return issues
}

func (c *HealthChecker) checkTTFB(page *model.Page) []model.Issue {
	if page.FetchDuration > maxTTFB {
		return []model.Issue{{
			CheckName: "health/slow-ttfb",
			Severity:  model.SeverityWarning,
			URL:       page.URL,
			Message:   fmt.Sprintf("time to first byte is %s, maximum is %s", page.FetchDuration.Round(time.Millisecond), maxTTFB),
		}}
	}
	return nil
}

func (c *HealthChecker) checkMixedContent(page *model.Page) []model.Issue {
	if !strings.HasPrefix(page.URL, "https://") {
		return nil
	}

	var offending []string
	for _, asset := range page.Assets {
		if strings.HasPrefix(asset, "http://") {
			offending = append(offending, asset)
		}
	}

	if len(offending) == 0 {
		return nil
	}

	return []model.Issue{{
		CheckName: "health/mixed-content",
		Severity:  model.SeverityWarning,
		URL:       page.URL,
		Message:   fmt.Sprintf("HTTPS page loads %d HTTP assets", len(offending)),
		Detail:    strings.Join(offending, "\n"),
	}}
}

func (c *HealthChecker) checkServerVersionLeak(page *model.Page) []model.Issue {
	if page.Headers == nil {
		return nil
	}
	server := page.Headers.Get("Server")
	if server == "" {
		return nil
	}
	if versionRe.MatchString(server) {
		return []model.Issue{{
			CheckName: "health/server-version-leak",
			Severity:  model.SeverityInfo,
			URL:       page.URL,
			Message:   fmt.Sprintf("Server header reveals version: %s", server),
		}}
	}
	return nil
}

func (c *HealthChecker) checkHTTPSRedirectNotPermanent(page *model.Page) []model.Issue {
	if !strings.HasPrefix(page.URL, "https://") {
		return nil
	}
	if len(page.RedirectChain) == 0 {
		return nil
	}
	first := page.RedirectChain[0]
	if strings.HasPrefix(first, "http://") {
		// The redirect chain starts with an HTTP URL, which means there was an
		// HTTP->HTTPS redirect. Check the status code: for the initial redirect
		// we infer non-permanent when the first entry is HTTP (the crawler stores
		// the URLs, not the status codes). However, we can only flag this if the
		// page records redirect status codes. Since model.Page does not carry
		// per-hop status codes, we use a heuristic: if the page itself reports a
		// non-permanent redirect status (302 or 307), flag it.
		if page.StatusCode == 302 || page.StatusCode == 307 {
			return []model.Issue{{
				CheckName: "health/https-redirect-not-permanent",
				Severity:  model.SeverityWarning,
				URL:       page.URL,
				Message:   fmt.Sprintf("HTTP to HTTPS redirect uses non-permanent status %d; prefer 301 or 308", page.StatusCode),
			}}
		}
	}
	return nil
}

func (c *HealthChecker) checkMissingCharset(page *model.Page) []model.Issue {
	ct := strings.ToLower(page.ContentType)
	if strings.Contains(ct, "text/html") && !strings.Contains(ct, "charset") {
		return []model.Issue{{
			CheckName: "health/missing-charset",
			Severity:  model.SeverityWarning,
			URL:       page.URL,
			Message:   fmt.Sprintf("Content-Type header is missing charset: %s", page.ContentType),
		}}
	}
	return nil
}
