package report

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io"

	"github.com/meysam81/scry/core/model"
)

//go:embed templates/report.html.tmpl
var htmlTemplateSource string

var htmlTmpl = template.Must(template.New("html").Parse(htmlTemplateSource))

// lighthouseRow holds a single row of Lighthouse scores for template rendering.
type lighthouseRow struct {
	URL           string
	Performance   int
	Accessibility int
	BestPractices int
	SEO           int
}

// htmlData holds the pre-processed data passed to the HTML template.
type htmlData struct {
	SeedURL       string
	CrawledAt     string
	PageCount     int
	Duration      string
	CriticalCount int
	WarningCount  int
	InfoCount     int
	Issues        []model.Issue
	Lighthouse    []lighthouseRow
	HasLighthouse bool
}

// HTMLReporter writes the CrawlResult as a self-contained HTML document.
type HTMLReporter struct{}

// Name returns "html".
func (r *HTMLReporter) Name() string { return "html" }

// Write renders result as an HTML report and writes it to w.
func (r *HTMLReporter) Write(_ context.Context, result *model.CrawlResult, w io.Writer) error {
	if result == nil {
		return nil
	}
	data := buildHTMLData(result)

	if err := htmlTmpl.Execute(w, data); err != nil {
		return fmt.Errorf("executing HTML template: %w", err)
	}

	return nil
}

// buildHTMLData converts a CrawlResult into the template data struct.
func buildHTMLData(result *model.CrawlResult) htmlData {
	var criticalCount, warningCount, infoCount int
	for _, iss := range result.Issues {
		switch iss.Severity {
		case model.SeverityCritical:
			criticalCount++
		case model.SeverityWarning:
			warningCount++
		case model.SeverityInfo:
			infoCount++
		}
	}

	rows := buildLighthouseRows(result.Lighthouse)

	return htmlData{
		SeedURL:       result.SeedURL,
		CrawledAt:     result.CrawledAt.Format(markdownTimeFmt),
		PageCount:     len(result.Pages),
		Duration:      fmt.Sprintf("%.1fs", result.Duration.Seconds()),
		CriticalCount: criticalCount,
		WarningCount:  warningCount,
		InfoCount:     infoCount,
		Issues:        result.Issues,
		Lighthouse:    rows,
		HasLighthouse: len(result.Lighthouse) > 0,
	}
}

// scoreToInt converts a Lighthouse score to an integer percentage.
// Scores > 1 are assumed to already be on a 0-100 scale.
// Scores in [0, 1] are multiplied by 100.
const lighthouseScaleThreshold = 1.0

// buildLighthouseRows converts raw Lighthouse results into template-ready rows.
func buildLighthouseRows(results []model.LighthouseResult) []lighthouseRow {
	rows := make([]lighthouseRow, 0, len(results))
	for _, lh := range results {
		rows = append(rows, lighthouseRow{
			URL:           lh.URL,
			Performance:   scoreToInt(lh.PerformanceScore),
			Accessibility: scoreToInt(lh.AccessibilityScore),
			BestPractices: scoreToInt(lh.BestPracticesScore),
			SEO:           scoreToInt(lh.SEOScore),
		})
	}
	return rows
}

// scoreToInt converts a Lighthouse score to an integer percentage.
// Scores > 1 are assumed to already be on a 0-100 scale.
// Scores in [0, 1] are multiplied by 100.
func scoreToInt(score float64) int {
	if score > lighthouseScaleThreshold {
		return int(score)
	}
	return int(score * 100)
}
