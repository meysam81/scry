// Package model defines the shared data types used across the scry CLI.
package model

import (
	"net/http"
	"strings"
	"time"
)

// Severity represents the severity level of an issue found during a scan.
type Severity string

const (
	// SeverityCritical indicates a critical issue that must be addressed.
	SeverityCritical Severity = "critical"
	// SeverityWarning indicates a potential problem worth investigating.
	SeverityWarning Severity = "warning"
	// SeverityInfo indicates an informational finding.
	SeverityInfo Severity = "info"
)

// Level returns the numeric level of the severity for comparison.
// Critical=3, warning=2, info=1, unknown=0.
func (s Severity) Level() int {
	switch s {
	case SeverityCritical:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

// AtLeast reports whether the severity is at least as severe as the given threshold.
func (s Severity) AtLeast(threshold Severity) bool {
	return s.Level() >= threshold.Level()
}

// SeverityFromString converts a string to a Severity, case-insensitively.
// It returns an empty Severity for unrecognised values.
func SeverityFromString(s string) Severity {
	switch Severity(strings.ToLower(s)) {
	case SeverityCritical:
		return SeverityCritical
	case SeverityWarning:
		return SeverityWarning
	case SeverityInfo:
		return SeverityInfo
	default:
		return ""
	}
}

// Issue represents a single problem detected by a check.
type Issue struct {
	CheckName string   `json:"check_name"`
	Severity  Severity `json:"severity"`
	Message   string   `json:"message"`
	URL       string   `json:"url"`
	Detail    string   `json:"detail,omitempty"`
}

// Page holds the fetched data and metadata for a single web page.
type Page struct {
	URL           string        `json:"url"`
	StatusCode    int           `json:"status_code"`
	ContentType   string        `json:"content_type"`
	RedirectChain []string      `json:"redirect_chain,omitempty"`
	Body          []byte        `json:"-"`
	Headers       http.Header   `json:"headers,omitempty"`
	Links         []string      `json:"links,omitempty"`
	Assets        []string      `json:"assets,omitempty"`
	Depth         int           `json:"depth"`
	FetchedAt     time.Time     `json:"fetched_at"`
	FetchDuration time.Duration `json:"fetch_duration"`
	InSitemap     bool          `json:"in_sitemap"`
}

// LighthouseResult stores the scores returned by a Lighthouse audit.
type LighthouseResult struct {
	URL                string    `json:"url"`
	PerformanceScore   float64   `json:"performance_score"`
	AccessibilityScore float64   `json:"accessibility_score"`
	BestPracticesScore float64   `json:"best_practices_score"`
	SEOScore           float64   `json:"seo_score"`
	FetchedAt          time.Time `json:"fetched_at"`
	Source             string    `json:"source"`
}

// CrawlResult is the top-level output of a crawl session.
type CrawlResult struct {
	SeedURL    string             `json:"seed_url"`
	Pages      []*Page            `json:"pages"`
	Issues     []Issue            `json:"issues"`
	Lighthouse []LighthouseResult `json:"lighthouse,omitempty"`
	CrawledAt  time.Time          `json:"crawled_at"`
	Duration   time.Duration      `json:"duration"`
}
