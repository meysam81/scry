package report

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/meysam81/scry/internal/model"
)

// JSONReporter writes the CrawlResult as pretty-printed JSON.
type JSONReporter struct{}

// Name returns "json".
func (r *JSONReporter) Name() string { return "json" }

// Write serialises result as indented JSON and writes it to w.
func (r *JSONReporter) Write(_ context.Context, result *model.CrawlResult, w io.Writer) error {
	if result == nil {
		return nil
	}
	data, err := json.MarshalIndent(result, "", "  ")
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
