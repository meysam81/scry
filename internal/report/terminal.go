package report

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/meysam81/scry/core/model"
)

// TerminalReporter renders a human-readable audit report to a terminal,
// using lipgloss for color and go-pretty for tables.
type TerminalReporter struct{}

// Name returns "terminal".
func (r *TerminalReporter) Name() string { return "terminal" }

// Write formats result as a colourful terminal report and writes it to w.
func (r *TerminalReporter) Write(_ context.Context, result *model.CrawlResult, w io.Writer) error {
	if result == nil {
		return nil
	}
	noColor := os.Getenv("NO_COLOR") != ""

	// Define styles — disabled when NO_COLOR is set.
	var (
		titleStyle    lipgloss.Style
		criticalStyle lipgloss.Style
		warningStyle  lipgloss.Style
		infoStyle     lipgloss.Style
	)

	if noColor {
		titleStyle = lipgloss.NewStyle()
		criticalStyle = lipgloss.NewStyle()
		warningStyle = lipgloss.NewStyle()
		infoStyle = lipgloss.NewStyle()
	} else {
		titleStyle = lipgloss.NewStyle().Bold(true)
		criticalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // red
		warningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
		infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))    // blue
	}

	// Group issues by severity.
	groups := map[model.Severity][]model.Issue{
		model.SeverityCritical: {},
		model.SeverityWarning:  {},
		model.SeverityInfo:     {},
	}
	for _, iss := range result.Issues {
		groups[iss.Severity] = append(groups[iss.Severity], iss)
	}

	// Header.
	if _, err := fmt.Fprintln(w, titleStyle.Render("=== Scry Audit Report ===")); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	// Summary metadata.
	durationSec := result.Duration.Seconds()
	lines := []string{
		fmt.Sprintf("URL:      %s", result.SeedURL),
		fmt.Sprintf("Pages:    %d", len(result.Pages)),
		fmt.Sprintf("Duration: %.1fs", durationSec),
	}
	for _, l := range lines {
		if _, err := fmt.Fprintln(w, l); err != nil {
			return fmt.Errorf("writing summary: %w", err)
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}

	// Severity summary table.
	sevTable := table.NewWriter()
	sevTable.SetStyle(table.StyleLight)
	sevTable.AppendHeader(table.Row{"Severity", "Count"})
	sevTable.AppendRow(table.Row{"Critical", len(groups[model.SeverityCritical])})
	sevTable.AppendRow(table.Row{"Warning", len(groups[model.SeverityWarning])})
	sevTable.AppendRow(table.Row{"Info", len(groups[model.SeverityInfo])})
	if _, err := fmt.Fprintln(w, sevTable.Render()); err != nil {
		return fmt.Errorf("writing severity table: %w", err)
	}

	// By Category table.
	summary := ComputeSummary(result)
	if len(summary.ByCategory) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return fmt.Errorf("writing newline: %w", err)
		}
		if _, err := fmt.Fprintln(w, titleStyle.Render("-- By Category --")); err != nil {
			return fmt.Errorf("writing category heading: %w", err)
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return fmt.Errorf("writing newline: %w", err)
		}

		// Sort categories for deterministic output.
		categories := make([]string, 0, len(summary.ByCategory))
		for cat := range summary.ByCategory {
			categories = append(categories, cat)
		}
		sort.Strings(categories)

		catTable := table.NewWriter()
		catTable.SetStyle(table.StyleLight)
		catTable.AppendHeader(table.Row{"Category", "Count"})
		for _, cat := range categories {
			catTable.AppendRow(table.Row{cat, summary.ByCategory[cat]})
		}
		if _, err := fmt.Fprintln(w, catTable.Render()); err != nil {
			return fmt.Errorf("writing category table: %w", err)
		}
	}

	// Issue sections in severity order.
	order := []struct {
		sev   model.Severity
		label string
		style lipgloss.Style
	}{
		{model.SeverityCritical, "Critical", criticalStyle},
		{model.SeverityWarning, "Warning", warningStyle},
		{model.SeverityInfo, "Info", infoStyle},
	}

	for _, o := range order {
		issues := groups[o.sev]
		if len(issues) == 0 {
			continue
		}

		// Sort issues by URL then check name for deterministic output.
		sort.Slice(issues, func(i, j int) bool {
			if issues[i].URL == issues[j].URL {
				return issues[i].CheckName < issues[j].CheckName
			}
			return issues[i].URL < issues[j].URL
		})

		if _, err := fmt.Fprintln(w); err != nil {
			return fmt.Errorf("writing newline: %w", err)
		}
		heading := fmt.Sprintf("-- %s Issues (%d) --", o.label, len(issues))
		if _, err := fmt.Fprintln(w, o.style.Render(heading)); err != nil {
			return fmt.Errorf("writing section heading: %w", err)
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return fmt.Errorf("writing newline: %w", err)
		}

		for _, iss := range issues {
			line := fmt.Sprintf("  [%s] %s", iss.CheckName, iss.URL)
			if _, err := fmt.Fprintln(w, o.style.Render(line)); err != nil {
				return fmt.Errorf("writing issue line: %w", err)
			}
			msg := fmt.Sprintf("  -> %s", iss.Message)
			if _, err := fmt.Fprintln(w, msg); err != nil {
				return fmt.Errorf("writing issue message: %w", err)
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return fmt.Errorf("writing newline: %w", err)
			}
		}
	}

	// Lighthouse scores table.
	if len(result.Lighthouse) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return fmt.Errorf("writing newline: %w", err)
		}
		if _, err := fmt.Fprintln(w, titleStyle.Render("-- Lighthouse Scores --")); err != nil {
			return fmt.Errorf("writing lighthouse heading: %w", err)
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return fmt.Errorf("writing newline: %w", err)
		}

		lhTable := table.NewWriter()
		lhTable.SetStyle(table.StyleLight)
		lhTable.AppendHeader(table.Row{"URL", "Performance", "Accessibility", "Best Practices", "SEO"})
		for _, lh := range result.Lighthouse {
			lhTable.AppendRow(table.Row{
				lh.URL,
				fmt.Sprintf("%.0f", lh.PerformanceScore),
				fmt.Sprintf("%.0f", lh.AccessibilityScore),
				fmt.Sprintf("%.0f", lh.BestPracticesScore),
				fmt.Sprintf("%.0f", lh.SEOScore),
			})
		}
		if _, err := fmt.Fprintln(w, lhTable.Render()); err != nil {
			return fmt.Errorf("writing lighthouse table: %w", err)
		}
	}

	return nil
}
