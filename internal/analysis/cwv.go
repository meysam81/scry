package analysis

import (
	"fmt"
	"sort"

	"github.com/meysam81/scry/core/model"
)

const (
	// cwvBottomN is the number of worst-scoring pages to report.
	cwvBottomN = 5

	// cwvPerformancePassThreshold is the minimum performance score for a
	// page to be considered "passing" Core Web Vitals.
	cwvPerformancePassThreshold = 90.0

	// cwvPoorSitePerformance is the threshold below which the average
	// site performance is flagged.
	cwvPoorSitePerformance = 50.0

	// cwvPoorSiteAccessibility is the threshold below which the average
	// site accessibility is flagged.
	cwvPoorSiteAccessibility = 90.0

	// cwvLowPassRate is the threshold below which the pass rate is flagged.
	cwvLowPassRate = 50.0
)

// CWVSummary aggregates Core Web Vitals / Lighthouse scores across all
// audited pages.
type CWVSummary struct {
	PageCount          int        `json:"page_count"`
	AvgPerformance     float64    `json:"avg_performance"`
	AvgAccessibility   float64    `json:"avg_accessibility"`
	AvgSEO             float64    `json:"avg_seo"`
	AvgBestPractices   float64    `json:"avg_best_practices"`
	WorstPerformance   []URLScore `json:"worst_performance"`
	WorstAccessibility []URLScore `json:"worst_accessibility"`
	PassRate           float64    `json:"pass_rate"`
}

// URLScore pairs a URL with a numeric score.
type URLScore struct {
	URL   string  `json:"url"`
	Score float64 `json:"score"`
}

// AggregateCWV computes aggregate Lighthouse scores from the given results.
// It returns nil if results is empty.
func AggregateCWV(results []model.LighthouseResult) *CWVSummary {
	if len(results) == 0 {
		return nil
	}

	summary := &CWVSummary{
		PageCount: len(results),
	}

	var totalPerf, totalAccess, totalSEO, totalBP float64
	var passing int

	perfScores := make([]URLScore, 0, len(results))
	accessScores := make([]URLScore, 0, len(results))

	for _, r := range results {
		totalPerf += r.PerformanceScore
		totalAccess += r.AccessibilityScore
		totalSEO += r.SEOScore
		totalBP += r.BestPracticesScore

		if r.PerformanceScore >= cwvPerformancePassThreshold {
			passing++
		}

		perfScores = append(perfScores, URLScore{URL: r.URL, Score: r.PerformanceScore})
		accessScores = append(accessScores, URLScore{URL: r.URL, Score: r.AccessibilityScore})
	}

	n := float64(len(results))
	summary.AvgPerformance = totalPerf / n
	summary.AvgAccessibility = totalAccess / n
	summary.AvgSEO = totalSEO / n
	summary.AvgBestPractices = totalBP / n
	summary.PassRate = float64(passing) / n * 100.0

	summary.WorstPerformance = bottomN(perfScores, cwvBottomN)
	summary.WorstAccessibility = bottomN(accessScores, cwvBottomN)

	return summary
}

// CheckCWVIssues examines the aggregated Lighthouse results and returns
// issues for poor site-wide scores.
func CheckCWVIssues(results []model.LighthouseResult) []model.Issue {
	summary := AggregateCWV(results)
	if summary == nil {
		return nil
	}

	var issues []model.Issue

	if summary.AvgPerformance < cwvPoorSitePerformance {
		issues = append(issues, model.Issue{
			CheckName: "lighthouse/poor-site-performance",
			Severity:  model.SeverityWarning,
			Message:   fmt.Sprintf("average site performance score is %.1f (threshold: %.0f)", summary.AvgPerformance, cwvPoorSitePerformance),
		})
	}

	if summary.AvgAccessibility < cwvPoorSiteAccessibility {
		issues = append(issues, model.Issue{
			CheckName: "lighthouse/poor-site-accessibility",
			Severity:  model.SeverityWarning,
			Message:   fmt.Sprintf("average site accessibility score is %.1f (threshold: %.0f)", summary.AvgAccessibility, cwvPoorSiteAccessibility),
		})
	}

	if summary.PassRate < cwvLowPassRate {
		issues = append(issues, model.Issue{
			CheckName: "lighthouse/low-pass-rate",
			Severity:  model.SeverityInfo,
			Message:   fmt.Sprintf("only %.1f%% of pages pass the performance threshold (>= %.0f)", summary.PassRate, cwvPerformancePassThreshold),
		})
	}

	return issues
}

// bottomN returns the n lowest-scoring entries from scores, sorted ascending
// by Score. If len(scores) < n, all entries are returned.
func bottomN(scores []URLScore, n int) []URLScore {
	if len(scores) == 0 {
		return nil
	}

	// Copy to avoid mutating the caller's slice.
	sorted := make([]URLScore, len(scores))
	copy(sorted, scores)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Score != sorted[j].Score {
			return sorted[i].Score < sorted[j].Score
		}
		return sorted[i].URL < sorted[j].URL
	})

	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}
