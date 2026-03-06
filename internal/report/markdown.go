package report

import (
	"context"
	"fmt"
	"io"
	"text/template"

	"github.com/meysam81/scry/internal/model"
)

const markdownTimeFmt = "2006-01-02 15:04:05"

var mdTmpl = template.Must(template.New("markdown").Parse(markdownTemplate))

const markdownTemplate = `# Site Audit Report

**URL:** {{ .SeedURL }}
**Crawled:** {{ .CrawledAt }}
**Pages:** {{ .PageCount }}
**Duration:** {{ .Duration }}

## Summary

| Severity | Count |
| -------- | ----- |
| Critical | {{ .CriticalCount }} |
| Warning  | {{ .WarningCount }} |
| Info     | {{ .InfoCount }} |
{{ if .CriticalIssues }}
## Critical Issues
{{ range .CriticalIssues }}
- [{{ .CheckName }}] {{ .URL }}: {{ .Message }}
{{- end }}
{{ end }}{{ if .WarningIssues }}
## Warning Issues
{{ range .WarningIssues }}
- [{{ .CheckName }}] {{ .URL }}: {{ .Message }}
{{- end }}
{{ end }}{{ if .InfoIssues }}
## Info Issues
{{ range .InfoIssues }}
- [{{ .CheckName }}] {{ .URL }}: {{ .Message }}
{{- end }}
{{ end }}{{ if .HasLighthouse }}
## Lighthouse Scores

| URL | Performance | Accessibility | Best Practices | SEO |
| --- | ----------- | ------------- | -------------- | --- |
{{ range .LighthouseRows -}}
| {{ .URL }} | {{ .Performance }} | {{ .Accessibility }} | {{ .BestPractices }} | {{ .SEO }} |
{{ end }}{{ end }}`

// mdData holds the pre-processed data passed to the Markdown template.
type mdData struct {
	SeedURL        string
	CrawledAt      string
	PageCount      int
	Duration       string
	CriticalCount  int
	WarningCount   int
	InfoCount      int
	CriticalIssues []model.Issue
	WarningIssues  []model.Issue
	InfoIssues     []model.Issue
	HasLighthouse  bool
	LighthouseRows []lighthouseRow
}

// MarkdownReporter writes the CrawlResult as a Markdown document.
type MarkdownReporter struct{}

// Name returns "markdown".
func (r *MarkdownReporter) Name() string { return "markdown" }

// Write renders result as Markdown and writes it to w.
func (r *MarkdownReporter) Write(_ context.Context, result *model.CrawlResult, w io.Writer) error {
	if result == nil {
		return nil
	}
	data := buildMdData(result)

	if err := mdTmpl.Execute(w, data); err != nil {
		return fmt.Errorf("executing markdown template: %w", err)
	}

	return nil
}

// buildMdData converts a CrawlResult into the template data struct.
func buildMdData(result *model.CrawlResult) mdData {
	var critical, warning, info []model.Issue
	for _, iss := range result.Issues {
		switch iss.Severity {
		case model.SeverityCritical:
			critical = append(critical, iss)
		case model.SeverityWarning:
			warning = append(warning, iss)
		case model.SeverityInfo:
			info = append(info, iss)
		}
	}

	rows := buildLighthouseRows(result.Lighthouse)

	return mdData{
		SeedURL:        result.SeedURL,
		CrawledAt:      result.CrawledAt.Format(markdownTimeFmt),
		PageCount:      len(result.Pages),
		Duration:       fmt.Sprintf("%.1fs", result.Duration.Seconds()),
		CriticalCount:  len(critical),
		WarningCount:   len(warning),
		InfoCount:      len(info),
		CriticalIssues: critical,
		WarningIssues:  warning,
		InfoIssues:     info,
		HasLighthouse:  len(result.Lighthouse) > 0,
		LighthouseRows: rows,
	}
}
