package audit

import (
	"context"
	"fmt"
	"strings"

	"github.com/meysam81/scry/internal/model"
	"golang.org/x/net/html"
)

const maxLinkDepth = 4

// LinkChecker analyses site-wide link structure for issues.
type LinkChecker struct{}

// NewLinkChecker returns a new LinkChecker.
func NewLinkChecker() *LinkChecker {
	return &LinkChecker{}
}

// Name returns the checker name.
func (c *LinkChecker) Name() string { return "links" }

// Check runs per-page link analysis.
func (c *LinkChecker) Check(_ context.Context, page *model.Page) []model.Issue {
	if !isHTMLContent(page) {
		return nil
	}
	doc := parseHTMLDocLog(page.Body, page.URL)
	if doc == nil {
		return nil
	}

	var issues []model.Issue
	issues = append(issues, c.checkExcessiveLinks(doc, page.URL)...)
	issues = append(issues, c.checkGenericAnchorText(doc, page.URL)...)
	return issues
}

// checkExcessiveLinks reports pages that have more than 100 <a href> links.
func (c *LinkChecker) checkExcessiveLinks(doc *html.Node, url string) []model.Issue {
	const maxLinks = 100
	anchors := findNodes(doc, "a")
	count := 0
	for _, a := range anchors {
		if _, ok := getAttr(a, "href"); ok {
			count++
		}
	}
	if count > maxLinks {
		return []model.Issue{{
			CheckName: "links/excessive-links",
			Severity:  model.SeverityInfo,
			URL:       url,
			Message:   fmt.Sprintf("page has %d links, which exceeds the recommended maximum of %d", count, maxLinks),
		}}
	}
	return nil
}

// checkGenericAnchorText reports <a> elements whose visible text is a generic
// phrase that provides poor context for SEO and accessibility.
func (c *LinkChecker) checkGenericAnchorText(doc *html.Node, url string) []model.Issue {
	genericTexts := map[string]struct{}{
		"click here": {},
		"read more":  {},
		"learn more": {},
		"here":       {},
		"more":       {},
		"link":       {},
		"this":       {},
	}

	var issues []model.Issue
	for _, a := range findNodes(doc, "a") {
		text := strings.ToLower(strings.TrimSpace(textContent(a)))
		if _, ok := genericTexts[text]; ok {
			issues = append(issues, model.Issue{
				CheckName: "links/generic-anchor-text",
				Severity:  model.SeverityInfo,
				URL:       url,
				Message:   fmt.Sprintf("link has generic anchor text %q", text),
			})
		}
	}
	return issues
}

// CheckSite runs site-wide link analysis.
func (c *LinkChecker) CheckSite(_ context.Context, pages []*model.Page) []model.Issue {
	var issues []model.Issue

	// Build a set of all page URLs and a map of which pages link to which.
	pageByURL := make(map[string]*model.Page, len(pages))
	linkedFrom := make(map[string][]string, len(pages))

	for _, p := range pages {
		pageByURL[p.URL] = p
	}

	for _, p := range pages {
		for _, link := range p.Links {
			if _, exists := pageByURL[link]; exists {
				linkedFrom[link] = append(linkedFrom[link], p.URL)
			}
		}
	}

	// Check broken internal links.
	for _, p := range pages {
		if p.StatusCode >= 400 && p.StatusCode <= 599 {
			if sources, ok := linkedFrom[p.URL]; ok {
				for _, src := range sources {
					issues = append(issues, model.Issue{
						CheckName: "links/broken-internal",
						Severity:  model.SeverityCritical,
						URL:       p.URL,
						Message:   fmt.Sprintf("broken internal link (HTTP %d), linked from %s", p.StatusCode, src),
					})
				}
			}
		}
	}

	// Check orphan pages (no inbound links, excluding root at depth 0).
	for _, p := range pages {
		if p.Depth == 0 {
			continue
		}
		if _, ok := linkedFrom[p.URL]; !ok {
			issues = append(issues, model.Issue{
				CheckName: "links/orphan-page",
				Severity:  model.SeverityWarning,
				URL:       p.URL,
				Message:   "page has no internal links pointing to it",
			})
		}
	}

	// Check deep pages.
	for _, p := range pages {
		if p.Depth > maxLinkDepth {
			issues = append(issues, model.Issue{
				CheckName: "links/deep-page",
				Severity:  model.SeverityInfo,
				URL:       p.URL,
				Message:   fmt.Sprintf("page is at depth %d, maximum recommended is %d", p.Depth, maxLinkDepth),
			})
		}
	}

	return issues
}
