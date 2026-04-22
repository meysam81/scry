package checks

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/meysam81/scry/core/model"
)

const (
	// hstsMinMaxAge is the minimum recommended max-age for HSTS (1 year).
	hstsMinMaxAge = 31536000
)

// insecureReferrerPolicies lists Referrer-Policy values considered insecure
// because they leak the full URL on cross-origin or downgrade requests.
var insecureReferrerPolicies = []string{
	"unsafe-url",
	"no-referrer-when-downgrade",
}

// SecurityChecker analyses pages for missing or weak HTTP security headers.
type SecurityChecker struct{}

// NewSecurityChecker returns a new SecurityChecker.
func NewSecurityChecker() *SecurityChecker {
	return &SecurityChecker{}
}

// Name returns the checker name.
func (c *SecurityChecker) Name() string { return "security" }

// Check runs per-page security header checks.
//
// Only pages with status code 200 are examined; error pages are skipped
// because their headers are often different from the intended configuration.
func (c *SecurityChecker) Check(_ context.Context, page *model.Page) []model.Issue {
	if page.StatusCode != 200 {
		return nil
	}

	var issues []model.Issue

	issues = append(issues, c.checkHSTS(page)...)
	issues = append(issues, c.checkCSP(page)...)
	issues = append(issues, c.checkCSPUnsafe(page)...)
	issues = append(issues, c.checkContentTypeOptions(page)...)
	issues = append(issues, c.checkFrameOptions(page)...)
	issues = append(issues, c.checkReferrerPolicy(page)...)
	issues = append(issues, c.checkPermissionsPolicy(page)...)
	issues = append(issues, c.checkInsecureCookies(page)...)
	issues = append(issues, c.checkCORSWildcard(page)...)
	issues = append(issues, c.checkMissingSRI(page)...)

	return issues
}

// CheckSite runs site-wide security checks across all crawled pages.
func (c *SecurityChecker) CheckSite(_ context.Context, pages []*model.Page) []model.Issue {
	return c.checkMissingSecurityTxt(pages)
}

func (c *SecurityChecker) checkHSTS(page *model.Page) []model.Issue {
	// HSTS is only relevant for pages served over HTTPS.
	if !strings.HasPrefix(page.URL, "https://") {
		return nil
	}

	val := page.Headers.Get("Strict-Transport-Security")
	if val == "" {
		return []model.Issue{{
			CheckName: "security/missing-strict-transport-security",
			Severity:  model.SeverityWarning,
			URL:       page.URL,
			Message:   "HTTPS page is missing the Strict-Transport-Security header",
		}}
	}

	maxAge := parseHSTSMaxAge(val)
	if maxAge >= 0 && maxAge < hstsMinMaxAge {
		return []model.Issue{{
			CheckName: "security/weak-hsts",
			Severity:  model.SeverityInfo,
			URL:       page.URL,
			Message:   fmt.Sprintf("HSTS max-age is %d seconds, recommended minimum is %d (1 year)", maxAge, hstsMinMaxAge),
		}}
	}

	return nil
}

// parseHSTSMaxAge extracts the max-age value from an HSTS header.
// It returns -1 if the directive is missing or unparseable.
func parseHSTSMaxAge(header string) int64 {
	for _, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		if after, ok := strings.CutPrefix(strings.ToLower(part), "max-age="); ok {
			val, err := strconv.ParseInt(strings.TrimSpace(after), 10, 64)
			if err != nil {
				return -1
			}
			return val
		}
	}
	return -1
}

func (c *SecurityChecker) checkCSP(page *model.Page) []model.Issue {
	if page.Headers.Get("Content-Security-Policy") == "" {
		return []model.Issue{{
			CheckName: "security/missing-content-security-policy",
			Severity:  model.SeverityWarning,
			URL:       page.URL,
			Message:   "page is missing the Content-Security-Policy header",
		}}
	}
	return nil
}

func (c *SecurityChecker) checkContentTypeOptions(page *model.Page) []model.Issue {
	val := page.Headers.Get("X-Content-Type-Options")
	if !strings.EqualFold(val, "nosniff") {
		return []model.Issue{{
			CheckName: "security/missing-x-content-type-options",
			Severity:  model.SeverityWarning,
			URL:       page.URL,
			Message:   "page is missing X-Content-Type-Options: nosniff",
		}}
	}
	return nil
}

func (c *SecurityChecker) checkFrameOptions(page *model.Page) []model.Issue {
	val := strings.ToUpper(page.Headers.Get("X-Frame-Options"))
	if val != "DENY" && val != "SAMEORIGIN" {
		return []model.Issue{{
			CheckName: "security/missing-x-frame-options",
			Severity:  model.SeverityInfo,
			URL:       page.URL,
			Message:   "page is missing X-Frame-Options (expected DENY or SAMEORIGIN)",
		}}
	}
	return nil
}

func (c *SecurityChecker) checkReferrerPolicy(page *model.Page) []model.Issue {
	val := page.Headers.Get("Referrer-Policy")
	if val == "" {
		return []model.Issue{{
			CheckName: "security/missing-referrer-policy",
			Severity:  model.SeverityInfo,
			URL:       page.URL,
			Message:   "page is missing the Referrer-Policy header",
		}}
	}

	normalized := strings.ToLower(strings.TrimSpace(val))
	for _, insecure := range insecureReferrerPolicies {
		if normalized == insecure {
			return []model.Issue{{
				CheckName: "security/insecure-referrer-policy",
				Severity:  model.SeverityWarning,
				URL:       page.URL,
				Message:   fmt.Sprintf("Referrer-Policy %q leaks the full URL on cross-origin requests", val),
			}}
		}
	}

	return nil
}

func (c *SecurityChecker) checkPermissionsPolicy(page *model.Page) []model.Issue {
	if page.Headers.Get("Permissions-Policy") == "" {
		return []model.Issue{{
			CheckName: "security/missing-permissions-policy",
			Severity:  model.SeverityInfo,
			URL:       page.URL,
			Message:   "page is missing the Permissions-Policy header",
		}}
	}
	return nil
}

// checkCSPUnsafe warns when a Content-Security-Policy header contains
// 'unsafe-inline' or 'unsafe-eval', which weaken CSP protections.
func (c *SecurityChecker) checkCSPUnsafe(page *model.Page) []model.Issue {
	csp := page.Headers.Get("Content-Security-Policy")
	if csp == "" {
		return nil
	}

	lower := strings.ToLower(csp)
	var found []string
	if strings.Contains(lower, "'unsafe-inline'") {
		found = append(found, "'unsafe-inline'")
	}
	if strings.Contains(lower, "'unsafe-eval'") {
		found = append(found, "'unsafe-eval'")
	}

	if len(found) == 0 {
		return nil
	}

	return []model.Issue{{
		CheckName: "security/csp-unsafe",
		Severity:  model.SeverityWarning,
		URL:       page.URL,
		Message:   fmt.Sprintf("Content-Security-Policy contains %s", strings.Join(found, " and ")),
	}}
}

// checkInsecureCookies warns when Set-Cookie headers are missing HttpOnly,
// Secure, or SameSite flags.
func (c *SecurityChecker) checkInsecureCookies(page *model.Page) []model.Issue {
	cookies := page.Headers.Values("Set-Cookie")
	if len(cookies) == 0 {
		return nil
	}

	var issues []model.Issue
	for _, raw := range cookies {
		// Extract cookie name (everything before the first '=').
		name := raw
		if idx := strings.Index(raw, "="); idx >= 0 {
			name = strings.TrimSpace(raw[:idx])
		}

		lower := strings.ToLower(raw)
		var missing []string
		if !strings.Contains(lower, "httponly") {
			missing = append(missing, "HttpOnly")
		}
		if !strings.Contains(lower, "secure") {
			missing = append(missing, "Secure")
		}
		if !strings.Contains(lower, "samesite") {
			missing = append(missing, "SameSite")
		}

		if len(missing) > 0 {
			issues = append(issues, model.Issue{
				CheckName: "security/insecure-cookies",
				Severity:  model.SeverityWarning,
				URL:       page.URL,
				Message:   fmt.Sprintf("cookie %q is missing %s flag(s)", name, strings.Join(missing, ", ")),
			})
		}
	}

	return issues
}

// checkCORSWildcard warns when Access-Control-Allow-Origin is set to "*".
func (c *SecurityChecker) checkCORSWildcard(page *model.Page) []model.Issue {
	val := page.Headers.Get("Access-Control-Allow-Origin")
	if strings.TrimSpace(val) == "*" {
		return []model.Issue{{
			CheckName: "security/cors-wildcard",
			Severity:  model.SeverityWarning,
			URL:       page.URL,
			Message:   "Access-Control-Allow-Origin is set to wildcard (*)",
		}}
	}
	return nil
}

// checkMissingSRI reports external <script> and <link rel="stylesheet"> tags
// that lack a Subresource Integrity (integrity) attribute.
func (c *SecurityChecker) checkMissingSRI(page *model.Page) []model.Issue {
	if !isHTMLContent(page) || len(page.Body) == 0 {
		return nil
	}

	doc := parseHTMLDocLog(page.Body, page.URL)
	if doc == nil {
		return nil
	}

	pageHost := extractHost(page.URL)
	var issues []model.Issue

	// Check <script src="..."> tags.
	for _, node := range findNodes(doc, "script") {
		src, ok := getAttr(node, "src")
		if !ok || src == "" {
			continue
		}
		if !isExternalOrigin(src, pageHost) {
			continue
		}
		if _, has := getAttr(node, "integrity"); !has {
			issues = append(issues, model.Issue{
				CheckName: "security/missing-sri",
				Severity:  model.SeverityInfo,
				URL:       page.URL,
				Message:   fmt.Sprintf("external script %q is missing integrity attribute", src),
			})
		}
	}

	// Check <link rel="stylesheet" href="..."> tags.
	for _, node := range findNodes(doc, "link") {
		rel, _ := getAttr(node, "rel")
		if !strings.EqualFold(rel, "stylesheet") {
			continue
		}
		href, ok := getAttr(node, "href")
		if !ok || href == "" {
			continue
		}
		if !isExternalOrigin(href, pageHost) {
			continue
		}
		if _, has := getAttr(node, "integrity"); !has {
			issues = append(issues, model.Issue{
				CheckName: "security/missing-sri",
				Severity:  model.SeverityInfo,
				URL:       page.URL,
				Message:   fmt.Sprintf("external stylesheet %q is missing integrity attribute", href),
			})
		}
	}

	return issues
}

// isExternalOrigin returns true when the given resource URL is hosted on a
// different origin than the page host. Protocol-relative URLs (//cdn.example.com)
// and absolute URLs with a different host are considered external.
func isExternalOrigin(rawURL, pageHost string) bool {
	// Protocol-relative URL.
	if strings.HasPrefix(rawURL, "//") {
		rawURL = "https:" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Relative URLs are same-origin.
	if parsed.Host == "" {
		return false
	}

	return !strings.EqualFold(parsed.Hostname(), pageHost)
}

// checkMissingSecurityTxt is a site-wide check that reports if none of the
// crawled pages correspond to /.well-known/security.txt.
func (c *SecurityChecker) checkMissingSecurityTxt(pages []*model.Page) []model.Issue {
	for _, p := range pages {
		if strings.HasSuffix(p.URL, "/.well-known/security.txt") {
			return nil
		}
	}

	// Use the first page URL as the issue URL for context.
	issueURL := ""
	if len(pages) > 0 {
		issueURL = pages[0].URL
	}

	return []model.Issue{{
		CheckName: "security/missing-security-txt",
		Severity:  model.SeverityInfo,
		URL:       issueURL,
		Message:   "site does not have a /.well-known/security.txt file",
	}}
}
