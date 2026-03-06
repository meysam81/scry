package audit

import (
	"context"
	"fmt"
	"strings"

	"github.com/meysam81/scry/internal/model"
	"golang.org/x/net/html"
)

const (
	maxHTMLSize  = 100 * 1024
	maxCSSInHead = 3
)

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

	doc, err := parseHTMLDoc(page.Body)
	if err != nil {
		return issues
	}

	issues = append(issues, c.checkRenderBlockingScripts(doc, page.URL)...)
	issues = append(issues, c.checkExcessiveCSS(doc, page.URL)...)

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
