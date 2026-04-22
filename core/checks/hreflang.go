package checks

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/meysam81/scry/core/model"
	"golang.org/x/net/html"
)

// bcp47Re validates a simplified BCP 47 language tag or x-default.
// Accepts 2-3 letter primary subtag, optional subtags of 2-4 letters each.
var bcp47Re = regexp.MustCompile(`^[a-z]{2,3}(-[a-zA-Z]{2,4})*$|^x-default$`)

// hreflangEntry represents a single hreflang annotation found on a page.
type hreflangEntry struct {
	lang string
	href string
}

// HreflangChecker validates hreflang annotations per-page and across the site.
type HreflangChecker struct{}

// NewHreflangChecker returns a new HreflangChecker.
func NewHreflangChecker() *HreflangChecker {
	return &HreflangChecker{}
}

// Name returns the checker name.
func (c *HreflangChecker) Name() string { return "hreflang" }

// Check runs per-page hreflang checks.
func (c *HreflangChecker) Check(_ context.Context, page *model.Page) []model.Issue {
	if !isHTMLContent(page) {
		return nil
	}
	doc := parseHTMLDocLog(page.Body, page.URL)
	if doc == nil {
		return nil
	}

	entries := extractHreflangEntries(doc)
	if len(entries) == 0 {
		return nil
	}

	var issues []model.Issue

	// Check 1: invalid language codes.
	for _, e := range entries {
		if !bcp47Re.MatchString(e.lang) {
			issues = append(issues, model.Issue{
				CheckName: "hreflang/invalid-language-code",
				Severity:  model.SeverityWarning,
				URL:       page.URL,
				Message:   fmt.Sprintf("hreflang value %q is not a valid BCP 47 language tag", e.lang),
			})
		}
	}

	// Check 2: missing x-default.
	hasXDefault := false
	for _, e := range entries {
		if e.lang == "x-default" {
			hasXDefault = true
			break
		}
	}
	if !hasXDefault {
		issues = append(issues, model.Issue{
			CheckName: "hreflang/missing-x-default",
			Severity:  model.SeverityInfo,
			URL:       page.URL,
			Message:   "page has hreflang annotations but no x-default",
		})
	}

	return issues
}

// CheckSite runs site-wide hreflang cross-reference checks.
func (c *HreflangChecker) CheckSite(_ context.Context, pages []*model.Page) []model.Issue {
	// Build a map: pageURL -> list of hreflang entries on that page.
	pageEntries := make(map[string][]hreflangEntry, len(pages))
	for _, p := range pages {
		if !isHTMLContent(p) {
			continue
		}
		doc := parseHTMLDocLog(p.Body, p.URL)
		if doc == nil {
			continue
		}
		entries := extractHreflangEntries(doc)
		if len(entries) > 0 {
			pageEntries[p.URL] = entries
		}
	}

	var issues []model.Issue

	for pageURL, entries := range pageEntries {
		hasSelfReference := false

		for _, e := range entries {
			href := normalizeHreflangHref(e.href)
			normalizedPage := normalizeHreflangHref(pageURL)

			// Self-reference detection.
			if href == normalizedPage {
				hasSelfReference = true
				continue
			}

			// Check 3: missing return link.
			targetEntries, targetExists := pageEntries[href]
			if !targetExists {
				continue
			}
			hasReturn := false
			for _, te := range targetEntries {
				if normalizeHreflangHref(te.href) == normalizedPage {
					hasReturn = true
					break
				}
			}
			if !hasReturn {
				issues = append(issues, model.Issue{
					CheckName: "hreflang/missing-return-link",
					Severity:  model.SeverityWarning,
					URL:       pageURL,
					Message:   fmt.Sprintf("page has hreflang pointing to %s, but that page does not link back", href),
				})
			}
		}

		// Check 4: self-reference missing.
		if !hasSelfReference {
			issues = append(issues, model.Issue{
				CheckName: "hreflang/self-reference-missing",
				Severity:  model.SeverityInfo,
				URL:       pageURL,
				Message:   "page has hreflang annotations but does not include itself",
			})
		}
	}

	return issues
}

// extractHreflangEntries finds all <link rel="alternate" hreflang="..." href="..."> tags.
func extractHreflangEntries(doc *html.Node) []hreflangEntry {
	links := findNodes(doc, "link")
	var entries []hreflangEntry
	for _, link := range links {
		rel, _ := getAttr(link, "rel")
		if !strings.EqualFold(rel, "alternate") {
			continue
		}
		hreflang, hasHL := getAttr(link, "hreflang")
		if !hasHL || hreflang == "" {
			continue
		}
		href, hasHref := getAttr(link, "href")
		if !hasHref {
			continue
		}
		entries = append(entries, hreflangEntry{
			lang: hreflang,
			href: href,
		})
	}
	return entries
}

// normalizeHreflangHref strips trailing slashes for comparison.
func normalizeHreflangHref(href string) string {
	return strings.TrimRight(href, "/")
}
