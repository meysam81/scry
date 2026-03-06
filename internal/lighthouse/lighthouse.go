// Package lighthouse provides Lighthouse audit runners and score-to-issue conversion.
package lighthouse

import (
	"context"
	"fmt"

	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/model"
)

// Score thresholds for converting Lighthouse scores to issues.
const (
	// PerfCriticalThreshold is the performance score below which an issue is critical.
	PerfCriticalThreshold = 50
	// PerfWarningThreshold is the performance score below which an issue is a warning.
	PerfWarningThreshold = 90
	// AccessibilityWarningThreshold is the accessibility score below which an issue is a warning.
	AccessibilityWarningThreshold = 90
	// SEOWarningThreshold is the SEO score below which an issue is a warning.
	SEOWarningThreshold = 90
)

// Check name constants.
const (
	CheckPerformance   = "lighthouse/performance"
	CheckAccessibility = "lighthouse/accessibility"
	CheckSEO           = "lighthouse/seo"
)

// Runner runs Lighthouse audits against a URL.
type Runner interface {
	Run(ctx context.Context, url string) (*model.LighthouseResult, error)
}

// NewRunner creates the appropriate lighthouse Runner based on config.
func NewRunner(cfg *config.Config) (Runner, error) {
	switch cfg.LighthouseMode {
	case "browserless":
		return NewBrowserlessClient(cfg.BrowserlessURL), nil
	case "psi":
		return NewPSIClient(cfg.PSIApiKey, cfg.PSIStrategy), nil
	default:
		return nil, fmt.Errorf("unknown lighthouse mode: %q", cfg.LighthouseMode)
	}
}

// ScoreToIssues converts a LighthouseResult into issues based on score thresholds.
func ScoreToIssues(result *model.LighthouseResult) []model.Issue {
	var issues []model.Issue

	// Performance scoring.
	switch {
	case result.PerformanceScore < PerfCriticalThreshold:
		issues = append(issues, model.Issue{
			CheckName: CheckPerformance,
			Severity:  model.SeverityCritical,
			Message:   fmt.Sprintf("performance score %0.f is below %d", result.PerformanceScore, PerfCriticalThreshold),
			URL:       result.URL,
		})
	case result.PerformanceScore < PerfWarningThreshold:
		issues = append(issues, model.Issue{
			CheckName: CheckPerformance,
			Severity:  model.SeverityWarning,
			Message:   fmt.Sprintf("performance score %0.f is below %d", result.PerformanceScore, PerfWarningThreshold),
			URL:       result.URL,
		})
	}

	// Accessibility scoring.
	if result.AccessibilityScore < AccessibilityWarningThreshold {
		issues = append(issues, model.Issue{
			CheckName: CheckAccessibility,
			Severity:  model.SeverityWarning,
			Message:   fmt.Sprintf("accessibility score %0.f is below %d", result.AccessibilityScore, AccessibilityWarningThreshold),
			URL:       result.URL,
		})
	}

	// SEO scoring.
	if result.SEOScore < SEOWarningThreshold {
		issues = append(issues, model.Issue{
			CheckName: CheckSEO,
			Severity:  model.SeverityWarning,
			Message:   fmt.Sprintf("seo score %0.f is below %d", result.SEOScore, SEOWarningThreshold),
			URL:       result.URL,
		})
	}

	return issues
}
