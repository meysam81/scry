// Package report provides formatters that render a CrawlResult for human or
// machine consumption.
package report

import (
	"context"
	"io"

	"github.com/meysam81/scry/internal/model"
)

// Reporter formats and writes a CrawlResult to an output destination.
type Reporter interface {
	// Name returns the short identifier of this reporter (e.g. "terminal").
	Name() string
	// Write renders result and writes the formatted output to w.
	Write(ctx context.Context, result *model.CrawlResult, w io.Writer) error
}

// AllReporters returns a map of all available reporters keyed by name.
func AllReporters() map[string]Reporter {
	return map[string]Reporter{
		"terminal": &TerminalReporter{},
		"json":     &JSONReporter{},
		"csv":      &CSVReporter{},
		"markdown": &MarkdownReporter{},
		"html":     &HTMLReporter{},
	}
}
