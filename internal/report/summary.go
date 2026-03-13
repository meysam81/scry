package report

import (
	"sort"
	"strings"

	"github.com/meysam81/scry/internal/model"
)

// SummaryStats holds aggregated statistics about audit results.
type SummaryStats struct {
	TotalIssues  int                    `json:"total_issues"`
	BySeverity   map[model.Severity]int `json:"by_severity"`
	ByCategory   map[string]int         `json:"by_category"`
	TopURLs      []URLIssueCount        `json:"top_urls"`
	PagesScanned int                    `json:"pages_scanned"`
}

// URLIssueCount pairs a URL with its issue count.
type URLIssueCount struct {
	URL   string `json:"url"`
	Count int    `json:"count"`
}

// ComputeSummary calculates summary statistics from a CrawlResult.
func ComputeSummary(result *model.CrawlResult) SummaryStats {
	stats := SummaryStats{
		BySeverity: map[model.Severity]int{
			model.SeverityCritical: 0,
			model.SeverityWarning:  0,
			model.SeverityInfo:     0,
		},
		ByCategory: make(map[string]int),
	}

	if result == nil {
		return stats
	}

	stats.TotalIssues = len(result.Issues)
	stats.PagesScanned = len(result.Pages)

	urlCounts := make(map[string]int)

	for _, iss := range result.Issues {
		// Count by severity.
		stats.BySeverity[iss.Severity]++

		// Extract category from CheckName prefix (e.g. "seo/missing-title" -> "seo").
		category := iss.CheckName
		if idx := strings.Index(iss.CheckName, "/"); idx >= 0 {
			category = iss.CheckName[:idx]
		}
		stats.ByCategory[category]++

		// Count issues per URL.
		urlCounts[iss.URL]++
	}

	// Build sorted TopURLs (descending by count), capped at 10.
	topURLs := make([]URLIssueCount, 0, len(urlCounts))
	for u, c := range urlCounts {
		topURLs = append(topURLs, URLIssueCount{URL: u, Count: c})
	}
	sort.Slice(topURLs, func(i, j int) bool {
		if topURLs[i].Count != topURLs[j].Count {
			return topURLs[i].Count > topURLs[j].Count
		}
		// Stable tie-break by URL for deterministic output.
		return topURLs[i].URL < topURLs[j].URL
	})
	if len(topURLs) > 10 {
		topURLs = topURLs[:10]
	}
	stats.TopURLs = topURLs

	return stats
}
