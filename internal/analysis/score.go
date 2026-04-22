// Package analysis provides advanced content analysis engines for crawled pages.
package analysis

import (
	"math"
	"sort"
	"strings"

	"github.com/meysam81/scry/core/model"
)

// ScoreResult holds the computed health score for a crawled site.
type ScoreResult struct {
	Overall     int            `json:"overall"`    // 0-100
	Categories  map[string]int `json:"categories"` // category -> score
	Breakdown   map[string]int `json:"breakdown"`  // severity -> count
	TotalPages  int            `json:"total_pages"`
	TotalIssues int            `json:"total_issues"`
}

// categoryNames enumerates the recognised score categories.
var categoryNames = []string{
	"seo",
	"health",
	"performance",
	"security",
	"accessibility",
	"links",
	"images",
	"structured-data",
	"content",
}

// checkPrefixToCategory maps a check name prefix to a score category.
// Prefixes not listed here are ignored during category scoring.
var checkPrefixToCategory = map[string]string{
	"seo":             "seo",
	"health":          "health",
	"performance":     "performance",
	"security":        "security",
	"accessibility":   "accessibility",
	"links":           "links",
	"external-links":  "links",
	"images":          "images",
	"structured-data": "structured-data",
	"content":         "content",
	"hreflang":        "seo",
	"tls":             "security",
}

// deduction returns the point deduction for a given severity.
// Weights match the Prometheus metrics push (critical=10, warning=3, info=1).
func deduction(s model.Severity) float64 {
	switch s {
	case model.SeverityCritical:
		return 10
	case model.SeverityWarning:
		return 3
	case model.SeverityInfo:
		return 1
	default:
		return 0
	}
}

// categoryFromCheckName extracts the category for an issue based on its
// CheckName prefix (the part before the first '/').
func categoryFromCheckName(checkName string) string {
	prefix := checkName
	if idx := strings.Index(checkName, "/"); idx >= 0 {
		prefix = checkName[:idx]
	}
	if cat, ok := checkPrefixToCategory[prefix]; ok {
		return cat
	}
	return ""
}

// ComputeScore calculates a 0-100 aggregate health score from the issues in a
// CrawlResult. The overall score starts at 100 and is decremented per issue
// (critical=-10, warning=-3, info=-1), floored at 0. Category scores are
// computed independently using the same algorithm.
func ComputeScore(result *model.CrawlResult) *ScoreResult {
	sr := &ScoreResult{
		Overall:     100,
		Categories:  make(map[string]int, len(categoryNames)),
		Breakdown:   make(map[string]int),
		TotalPages:  len(result.Pages),
		TotalIssues: len(result.Issues),
	}

	// Initialise all categories to 100.
	for _, name := range categoryNames {
		sr.Categories[name] = 100
	}

	// Accumulate deductions.
	overallDeduction := 0.0
	catDeductions := make(map[string]float64, len(categoryNames))

	for _, issue := range result.Issues {
		d := deduction(issue.Severity)
		overallDeduction += d

		// Breakdown by severity.
		sr.Breakdown[string(issue.Severity)]++

		// Category deduction.
		cat := categoryFromCheckName(issue.CheckName)
		if cat != "" {
			catDeductions[cat] += d
		}
	}

	sr.Overall = clampScore(100.0 - overallDeduction)

	for _, name := range categoryNames {
		sr.Categories[name] = clampScore(100.0 - catDeductions[name])
	}

	return sr
}

// clampScore rounds a floating-point score to an integer in [0, 100].
func clampScore(v float64) int {
	rounded := int(math.Round(v))
	if rounded < 0 {
		return 0
	}
	if rounded > 100 {
		return 100
	}
	return rounded
}

// SortedCategories returns the category names sorted alphabetically.
// This is useful for deterministic output ordering.
func SortedCategories(sr *ScoreResult) []string {
	names := make([]string, 0, len(sr.Categories))
	for name := range sr.Categories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
