package analysis

import (
	"context"
	"fmt"
	"strings"

	"github.com/meysam81/scry/internal/model"
)

// CrawlBudgetAnalyzer detects crawl budget waste across a site.
type CrawlBudgetAnalyzer struct{}

// NewCrawlBudgetAnalyzer returns a new CrawlBudgetAnalyzer.
func NewCrawlBudgetAnalyzer() *CrawlBudgetAnalyzer {
	return &CrawlBudgetAnalyzer{}
}

// Name returns the checker name.
func (c *CrawlBudgetAnalyzer) Name() string { return "crawl-budget" }

// Check returns nil; all analysis is site-wide.
func (c *CrawlBudgetAnalyzer) Check(_ context.Context, _ *model.Page) []model.Issue {
	return nil
}

// CheckSite runs crawl budget analysis across all pages.
func (c *CrawlBudgetAnalyzer) CheckSite(_ context.Context, pages []*model.Page) []model.Issue {
	if len(pages) == 0 {
		return nil
	}

	var issues []model.Issue

	issues = append(issues, c.checkDeepPages(pages)...)
	issues = append(issues, c.checkParameterURLs(pages)...)
	issues = append(issues, c.checkLowInternalLinks(pages)...)
	issues = append(issues, c.checkSitemapGap(pages)...)

	return issues
}

// checkDeepPages flags when more than 10% of pages are at depth > 3.
func (c *CrawlBudgetAnalyzer) checkDeepPages(pages []*model.Page) []model.Issue {
	total := len(pages)
	deep := 0
	for _, p := range pages {
		if p.Depth > 3 {
			deep++
		}
	}

	if total > 0 && float64(deep)/float64(total) > 0.10 {
		return []model.Issue{{
			CheckName: "crawl-budget/deep-pages",
			Severity:  model.SeverityWarning,
			Message:   fmt.Sprintf("%d of %d pages (%.0f%%) are at depth > 3", deep, total, 100*float64(deep)/float64(total)),
		}}
	}
	return nil
}

// checkParameterURLs flags URLs containing query parameters.
func (c *CrawlBudgetAnalyzer) checkParameterURLs(pages []*model.Page) []model.Issue {
	count := 0
	for _, p := range pages {
		if strings.Contains(p.URL, "?") {
			count++
		}
	}

	if count > 0 {
		return []model.Issue{{
			CheckName: "crawl-budget/parameter-urls",
			Severity:  model.SeverityInfo,
			Message:   fmt.Sprintf("%d URLs contain query parameters that could waste crawl budget", count),
		}}
	}
	return nil
}

// checkLowInternalLinks flags when the average internal links per page is < 3.
func (c *CrawlBudgetAnalyzer) checkLowInternalLinks(pages []*model.Page) []model.Issue {
	if len(pages) == 0 {
		return nil
	}

	totalLinks := 0
	for _, p := range pages {
		totalLinks += len(p.Links)
	}

	avg := float64(totalLinks) / float64(len(pages))
	if avg < 3.0 {
		return []model.Issue{{
			CheckName: "crawl-budget/low-internal-links",
			Severity:  model.SeverityInfo,
			Message:   fmt.Sprintf("average internal links per page is %.1f, recommended minimum is 3", avg),
		}}
	}
	return nil
}

// checkSitemapGap flags when crawled vs sitemap pages diverge by more than 20%.
func (c *CrawlBudgetAnalyzer) checkSitemapGap(pages []*model.Page) []model.Issue {
	crawled := len(pages)
	crawledAndInSitemap := 0

	for _, p := range pages {
		if p.InSitemap {
			crawledAndInSitemap++
		}
	}

	crawledNotInSitemap := crawled - crawledAndInSitemap

	if crawled == 0 {
		return nil
	}

	// More than 20% of crawled pages not in sitemap.
	if float64(crawledNotInSitemap)/float64(crawled) > 0.20 {
		return []model.Issue{{
			CheckName: "crawl-budget/large-sitemap-gap",
			Severity:  model.SeverityWarning,
			Message:   fmt.Sprintf("%d of %d crawled pages (%.0f%%) are not in the sitemap", crawledNotInSitemap, crawled, 100*float64(crawledNotInSitemap)/float64(crawled)),
		}}
	}

	return nil
}
