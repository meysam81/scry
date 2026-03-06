package audit

import (
	"context"
	"fmt"

	"github.com/meysam81/scry/internal/model"
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

// Check returns nil — link checks are site-wide only.
func (c *LinkChecker) Check(_ context.Context, _ *model.Page) []model.Issue {
	return nil
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
