package analysis

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/meysam81/scry/internal/model"
)

// RedirectEntry represents a single redirect, including its full chain,
// hop count, and the set of pages that link to the redirecting URL.
type RedirectEntry struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Chain    []string `json:"chain"`
	Hops     int      `json:"hops"`
	LinkedBy []string `json:"linked_by"`
}

// RedirectMap holds all redirect entries discovered during analysis.
type RedirectMap struct {
	Entries []RedirectEntry `json:"entries"`
}

// RedirectChecker implements SiteChecker for redirect-related issues.
type RedirectChecker struct{}

// NewRedirectChecker returns a new RedirectChecker.
func NewRedirectChecker() *RedirectChecker {
	return &RedirectChecker{}
}

// Name returns the checker name.
func (c *RedirectChecker) Name() string { return "redirects" }

// Check returns nil; all redirect analysis is site-wide.
func (c *RedirectChecker) Check(_ context.Context, _ *model.Page) []model.Issue {
	return nil
}

// CheckSite runs site-wide redirect analysis and returns any issues found.
func (c *RedirectChecker) CheckSite(_ context.Context, pages []*model.Page) []model.Issue {
	rm := AnalyzeRedirects(pages)
	return checkRedirectIssues(rm, pages)
}

// AnalyzeRedirects scans all pages for non-empty RedirectChain, builds
// redirect entries with full chain info, cross-references pages that link
// to the redirecting URL, and returns the result sorted by hop count descending.
func AnalyzeRedirects(pages []*model.Page) *RedirectMap {
	// Build a map of which pages link to which URLs.
	linkedBy := make(map[string][]string)
	for _, p := range pages {
		for _, link := range p.Links {
			linkedBy[link] = append(linkedBy[link], p.URL)
		}
	}

	// Build the set of pages that themselves redirect, keyed by final URL.
	redirectTargets := make(map[string]bool)
	for _, p := range pages {
		if len(p.RedirectChain) > 0 {
			redirectTargets[p.URL] = true
		}
	}

	var entries []RedirectEntry
	for _, p := range pages {
		if len(p.RedirectChain) == 0 {
			continue
		}

		from := p.RedirectChain[0]
		to := p.URL
		chain := make([]string, len(p.RedirectChain)+1)
		copy(chain, p.RedirectChain)
		chain[len(p.RedirectChain)] = to

		entry := RedirectEntry{
			From:     from,
			To:       to,
			Chain:    chain,
			Hops:     len(p.RedirectChain),
			LinkedBy: linkedBy[from],
		}
		entries = append(entries, entry)
	}

	// Sort by hop count descending, then by From URL for stability.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Hops != entries[j].Hops {
			return entries[i].Hops > entries[j].Hops
		}
		return entries[i].From < entries[j].From
	})

	return &RedirectMap{Entries: entries}
}

// ExportCSV writes the redirect map as CSV to the given writer.
// The header row is: from,to,chain,hops,linked_by.
func (m *RedirectMap) ExportCSV(w io.Writer) error {
	cw := csv.NewWriter(w)

	header := []string{"from", "to", "chain", "hops", "linked_by"}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	for i, e := range m.Entries {
		row := []string{
			e.From,
			e.To,
			strings.Join(e.Chain, " -> "),
			fmt.Sprintf("%d", e.Hops),
			strings.Join(e.LinkedBy, " "),
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("writing CSV row %d: %w", i, err)
		}
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("flushing CSV writer: %w", err)
	}

	return nil
}

// checkRedirectIssues examines the redirect map and pages for known redirect
// problems and returns the corresponding issues.
func checkRedirectIssues(rm *RedirectMap, pages []*model.Page) []model.Issue {
	var issues []model.Issue

	// Build a set of URLs that are redirect targets (i.e., pages with
	// non-empty RedirectChain) so we can detect redirect-to-redirect.
	redirectSources := make(map[string]bool)
	for _, e := range rm.Entries {
		redirectSources[e.From] = true
	}

	// Build a set of all internal page URLs for the internal-redirect check.
	pageURLs := make(map[string]bool, len(pages))
	for _, p := range pages {
		pageURLs[p.URL] = true
	}

	for _, e := range rm.Entries {
		// Long chain: redirect chain > 2 hops.
		if e.Hops > 2 {
			issues = append(issues, model.Issue{
				CheckName: "redirects/long-chain",
				Severity:  model.SeverityWarning,
				URL:       e.From,
				Message:   fmt.Sprintf("redirect chain has %d hops (%s)", e.Hops, strings.Join(e.Chain, " -> ")),
			})
		}

		// Internal redirect: an internal page links to a URL that redirects.
		for _, src := range e.LinkedBy {
			if pageURLs[src] {
				issues = append(issues, model.Issue{
					CheckName: "redirects/internal-redirect",
					Severity:  model.SeverityInfo,
					URL:       src,
					Message:   fmt.Sprintf("links to %s which redirects to %s (update link to final URL)", e.From, e.To),
				})
			}
		}

		// Redirect-to-redirect: the target of this redirect is itself a
		// redirect source.
		if redirectSources[e.To] {
			issues = append(issues, model.Issue{
				CheckName: "redirects/redirect-to-redirect",
				Severity:  model.SeverityWarning,
				URL:       e.From,
				Message:   fmt.Sprintf("redirects to %s which is itself a redirect", e.To),
			})
		}
	}

	return issues
}
