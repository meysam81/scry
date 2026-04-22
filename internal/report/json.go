package report

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/meysam81/scry/core/model"
)

// jsonOutput wraps CrawlResult with summary statistics for the JSON reporter.
type jsonOutput struct {
	*model.CrawlResult
	Summary SummaryStats `json:"summary"`
}

// JSONReporter writes the CrawlResult as pretty-printed JSON.
type JSONReporter struct{}

// Name returns "json".
func (r *JSONReporter) Name() string { return "json" }

// Write serialises result as indented JSON and writes it to w.
func (r *JSONReporter) Write(_ context.Context, result *model.CrawlResult, w io.Writer) error {
	if result == nil {
		return nil
	}

	output := jsonOutput{
		CrawlResult: result,
		Summary:     ComputeSummary(result),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling crawl result to JSON: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing JSON output: %w", err)
	}

	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("writing trailing newline: %w", err)
	}

	return nil
}
