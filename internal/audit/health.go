package audit

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/meysam81/scry/internal/model"
)

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
