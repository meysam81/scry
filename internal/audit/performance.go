package audit

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/meysam81/scry/internal/model"
	"golang.org/x/net/html"
)

const (
	maxHTMLSize          = 100 * 1024
	maxCSSInHead         = 3
	maxThirdPartyScripts = 5
	maxDOMElements       = 1500
	maxInlineBloatBytes  = 50 * 1024
	maxWebfonts          = 4
)

// multiLineCommentRe matches C-style multi-line comments.
var multiLineCommentRe = regexp.MustCompile(`/\*[\s\S]{4,}?\*/`)

// excessiveWhitespaceRe matches 3+ consecutive whitespace characters (not just spaces).
var excessiveWhitespaceRe = regexp.MustCompile(`\s{3,}`)

// PerformanceChecker analyses pages for performance issues.
type PerformanceChecker struct{}

// NewPerformanceChecker returns a new PerformanceChecker.
func NewPerformanceChecker() *PerformanceChecker {
	return &PerformanceChecker{}
}

// Name returns the checker name.
func (c *PerformanceChecker) Name() string { return "performance" }

// Check runs per-page performance checks.
func (c *PerformanceChecker) Check(_ context.Context, page *model.Page) []model.Issue {
	if !isHTMLContent(page) {
		return nil
	}

	var issues []model.Issue

	issues = append(issues, c.checkLargeHTML(page)...)
	issues = append(issues, c.checkCompression(page)...)
	issues = append(issues, c.checkMissingCacheHeaders(page)...)
	issues = append(issues, c.checkNoHTTP2(page)...)

	doc := parseHTMLDocLog(page.Body, page.URL)
	if doc == nil {
		return issues
	}

	issues = append(issues, c.checkRenderBlockingScripts(doc, page.URL)...)
	issues = append(issues, c.checkExcessiveCSS(doc, page.URL)...)
	issues = append(issues, c.checkRenderBlockingCSS(doc, page.URL)...)
	issues = append(issues, c.checkMissingResourceHints(doc, page.URL)...)
	issues = append(issues, c.checkFontLoading(doc, page.URL)...)
	issues = append(issues, c.checkExcessiveThirdParty(doc, page)...)
	issues = append(issues, c.checkExcessiveDOMSize(doc, page.URL)...)
	issues = append(issues, c.checkInlineBloat(doc, page.URL)...)
	issues = append(issues, c.checkExcessiveWebfonts(doc, page.URL)...)
	issues = append(issues, c.checkUnminifiedResources(doc, page.URL)...)

	return issues
}

func (c *PerformanceChecker) checkLargeHTML(page *model.Page) []model.Issue {
	size := len(page.Body)
	if size > maxHTMLSize {
		return []model.Issue{{
			CheckName: "performance/large-html",
			Severity:  model.SeverityWarning,
			URL:       page.URL,
			Message:   fmt.Sprintf("HTML size is %d bytes (%.0f KB), maximum recommended is %d bytes", size, float64(size)/1024, maxHTMLSize),
		}}
	}
	return nil
}

func (c *PerformanceChecker) checkCompression(page *model.Page) []model.Issue {
	enc := page.Headers.Get("Content-Encoding")
	enc = strings.ToLower(enc)
	if strings.Contains(enc, "gzip") || strings.Contains(enc, "br") {
		return nil
	}
	return []model.Issue{{
		CheckName: "performance/no-compression",
		Severity:  model.SeverityWarning,
		URL:       page.URL,
		Message:   "HTML response is not compressed (missing gzip or br Content-Encoding)",
	}}
}

func (c *PerformanceChecker) checkRenderBlockingScripts(doc *html.Node, url string) []model.Issue {
	heads := findNodes(doc, "head")
	if len(heads) == 0 {
		return nil
	}

	var issues []model.Issue
	for _, head := range heads {
		scripts := findNodes(head, "script")
		for _, s := range scripts {
			src, hasSrc := getAttr(s, "src")
			if !hasSrc {
				continue
			}
			_, hasAsync := getAttr(s, "async")
			_, hasDefer := getAttr(s, "defer")
			if !hasAsync && !hasDefer {
				issues = append(issues, model.Issue{
					CheckName: "performance/render-blocking-script",
					Severity:  model.SeverityWarning,
					URL:       url,
					Message:   fmt.Sprintf("render-blocking script in <head>: %s", src),
				})
			}
		}
	}
	return issues
}

func (c *PerformanceChecker) checkExcessiveCSS(doc *html.Node, url string) []model.Issue {
	heads := findNodes(doc, "head")
	if len(heads) == 0 {
		return nil
	}

	count := 0
	for _, head := range heads {
		links := findNodes(head, "link")
		for _, l := range links {
			rel, _ := getAttr(l, "rel")
			if strings.EqualFold(rel, "stylesheet") {
				count++
			}
		}
	}

	if count > maxCSSInHead {
		return []model.Issue{{
			CheckName: "performance/excessive-css",
			Severity:  model.SeverityInfo,
			URL:       url,
			Message:   fmt.Sprintf("page has %d stylesheets in <head>, maximum recommended is %d", count, maxCSSInHead),
		}}
	}
	return nil
}

// checkRenderBlockingCSS flags <link rel="stylesheet"> in <head> without a
// non-blocking media attribute (e.g. media="print").
func (c *PerformanceChecker) checkRenderBlockingCSS(doc *html.Node, pageURL string) []model.Issue {
	heads := findNodes(doc, "head")
	if len(heads) == 0 {
		return nil
	}

	var issues []model.Issue
	for _, head := range heads {
		links := findNodes(head, "link")
		for _, l := range links {
			rel, _ := getAttr(l, "rel")
			if !strings.EqualFold(rel, "stylesheet") {
				continue
			}
			media, hasMedia := getAttr(l, "media")
			if hasMedia && !strings.EqualFold(strings.TrimSpace(media), "all") && strings.TrimSpace(media) != "" {
				continue // non-blocking media query like "print"
			}
			href, _ := getAttr(l, "href")
			issues = append(issues, model.Issue{
				CheckName: "performance/render-blocking-css",
				Severity:  model.SeverityWarning,
				URL:       pageURL,
				Message:   fmt.Sprintf("render-blocking stylesheet in <head>: %s", href),
			})
		}
	}
	return issues
}

// checkMissingResourceHints flags pages with no <link rel="preconnect"> or
// <link rel="dns-prefetch"> in <head>.
func (c *PerformanceChecker) checkMissingResourceHints(doc *html.Node, pageURL string) []model.Issue {
	heads := findNodes(doc, "head")
	if len(heads) == 0 {
		return nil
	}

	for _, head := range heads {
		links := findNodes(head, "link")
		for _, l := range links {
			rel, _ := getAttr(l, "rel")
			rel = strings.ToLower(rel)
			if rel == "preconnect" || rel == "dns-prefetch" {
				return nil
			}
		}
	}

	return []model.Issue{{
		CheckName: "performance/missing-resource-hints",
		Severity:  model.SeverityInfo,
		URL:       pageURL,
		Message:   "page has no <link rel=\"preconnect\"> or <link rel=\"dns-prefetch\"> resource hints",
	}}
}

// checkFontLoading flags inline <style> blocks containing @font-face without
// font-display, or with font-display: block.
func (c *PerformanceChecker) checkFontLoading(doc *html.Node, pageURL string) []model.Issue {
	styles := findNodes(doc, "style")
	var issues []model.Issue
	for _, s := range styles {
		content := textContent(s)
		if !strings.Contains(content, "@font-face") {
			continue
		}
		lower := strings.ToLower(content)
		if !strings.Contains(lower, "font-display") {
			issues = append(issues, model.Issue{
				CheckName: "performance/font-loading",
				Severity:  model.SeverityInfo,
				URL:       pageURL,
				Message:   "inline @font-face declaration missing font-display property",
			})
		} else if strings.Contains(lower, "font-display:") {
			// extract the value after font-display:
			idx := strings.Index(lower, "font-display:")
			rest := strings.TrimSpace(lower[idx+len("font-display:"):])
			semi := strings.IndexByte(rest, ';')
			if semi > 0 {
				rest = rest[:semi]
			}
			rest = strings.TrimSpace(rest)
			if rest == "block" {
				issues = append(issues, model.Issue{
					CheckName: "performance/font-loading",
					Severity:  model.SeverityInfo,
					URL:       pageURL,
					Message:   "inline @font-face uses font-display: block which delays text rendering",
				})
			}
		}
	}
	return issues
}

// checkExcessiveThirdParty flags pages with >5 external script origins.
func (c *PerformanceChecker) checkExcessiveThirdParty(doc *html.Node, page *model.Page) []model.Issue {
	pageHost := ""
	if parsed, err := url.Parse(page.URL); err == nil {
		pageHost = strings.ToLower(parsed.Host)
	}

	origins := make(map[string]struct{})
	scripts := findNodes(doc, "script")
	for _, s := range scripts {
		src, hasSrc := getAttr(s, "src")
		if !hasSrc || src == "" {
			continue
		}
		parsed, err := url.Parse(src)
		if err != nil || parsed.Host == "" {
			continue // relative URL, same origin
		}
		host := strings.ToLower(parsed.Host)
		if host != pageHost {
			origins[host] = struct{}{}
		}
	}

	if len(origins) > maxThirdPartyScripts {
		return []model.Issue{{
			CheckName: "performance/excessive-third-party",
			Severity:  model.SeverityWarning,
			URL:       page.URL,
			Message:   fmt.Sprintf("page loads scripts from %d external origins, maximum recommended is %d", len(origins), maxThirdPartyScripts),
		}}
	}
	return nil
}

// countElementNodes recursively counts all element nodes in the subtree.
func countElementNodes(n *html.Node) int {
	count := 0
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			count++
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return count
}

// checkExcessiveDOMSize flags pages with more than maxDOMElements element nodes.
func (c *PerformanceChecker) checkExcessiveDOMSize(doc *html.Node, pageURL string) []model.Issue {
	count := countElementNodes(doc)
	if count > maxDOMElements {
		return []model.Issue{{
			CheckName: "performance/excessive-dom-size",
			Severity:  model.SeverityWarning,
			URL:       pageURL,
			Message:   fmt.Sprintf("DOM has %d element nodes, maximum recommended is %d", count, maxDOMElements),
		}}
	}
	return nil
}

// checkInlineBloat flags pages where total inline <script> and <style> text
// exceeds maxInlineBloatBytes.
func (c *PerformanceChecker) checkInlineBloat(doc *html.Node, pageURL string) []model.Issue {
	total := 0
	for _, tag := range []string{"script", "style"} {
		nodes := findNodes(doc, tag)
		for _, n := range nodes {
			// Only count inline content (no src attribute for scripts)
			if tag == "script" {
				if _, hasSrc := getAttr(n, "src"); hasSrc {
					continue
				}
			}
			total += len(textContent(n))
		}
	}

	if total > maxInlineBloatBytes {
		return []model.Issue{{
			CheckName: "performance/inline-bloat",
			Severity:  model.SeverityInfo,
			URL:       pageURL,
			Message:   fmt.Sprintf("total inline <script> and <style> content is %d bytes (%.0f KB), maximum recommended is %d bytes", total, float64(total)/1024, maxInlineBloatBytes),
		}}
	}
	return nil
}

// checkMissingCacheHeaders flags pages without a Cache-Control header.
func (c *PerformanceChecker) checkMissingCacheHeaders(page *model.Page) []model.Issue {
	if page.Headers.Get("Cache-Control") != "" {
		return nil
	}
	return []model.Issue{{
		CheckName: "performance/missing-cache-headers",
		Severity:  model.SeverityWarning,
		URL:       page.URL,
		Message:   "response has no Cache-Control header",
	}}
}

// checkExcessiveWebfonts flags pages with more than maxWebfonts @font-face
// declarations in inline <style> blocks.
func (c *PerformanceChecker) checkExcessiveWebfonts(doc *html.Node, pageURL string) []model.Issue {
	count := 0
	styles := findNodes(doc, "style")
	for _, s := range styles {
		content := strings.ToLower(textContent(s))
		count += strings.Count(content, "@font-face")
	}

	if count > maxWebfonts {
		return []model.Issue{{
			CheckName: "performance/excessive-webfonts",
			Severity:  model.SeverityWarning,
			URL:       pageURL,
			Message:   fmt.Sprintf("page has %d @font-face declarations in inline styles, maximum recommended is %d", count, maxWebfonts),
		}}
	}
	return nil
}

// checkUnminifiedResources flags inline <script> blocks that appear unminified,
// detected by multi-line comments or excessive consecutive whitespace.
func (c *PerformanceChecker) checkUnminifiedResources(doc *html.Node, pageURL string) []model.Issue {
	scripts := findNodes(doc, "script")
	var issues []model.Issue
	for _, s := range scripts {
		if _, hasSrc := getAttr(s, "src"); hasSrc {
			continue
		}
		content := textContent(s)
		if len(strings.TrimSpace(content)) == 0 {
			continue
		}
		if multiLineCommentRe.MatchString(content) || excessiveWhitespaceRe.MatchString(content) {
			issues = append(issues, model.Issue{
				CheckName: "performance/unminified-resources",
				Severity:  model.SeverityInfo,
				URL:       pageURL,
				Message:   "inline <script> appears to contain unminified code",
			})
			break // one issue per page is enough
		}
	}
	return issues
}

// checkNoHTTP2 heuristically detects the absence of HTTP/2 or HTTP/3 support
// by looking for Alt-Svc headers indicating h2/h3.
func (c *PerformanceChecker) checkNoHTTP2(page *model.Page) []model.Issue {
	altSvc := strings.ToLower(page.Headers.Get("Alt-Svc"))
	if strings.Contains(altSvc, "h2") || strings.Contains(altSvc, "h3") {
		return nil
	}
	return []model.Issue{{
		CheckName: "performance/no-http2",
		Severity:  model.SeverityInfo,
		URL:       page.URL,
		Message:   "no Alt-Svc header indicating HTTP/2 or HTTP/3 support detected",
	}}
}
