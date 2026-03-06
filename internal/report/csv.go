package report

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/meysam81/scry/internal/model"
)

// CSVReporter writes the CrawlResult issues as a CSV document.
type CSVReporter struct{}

// Name returns "csv".
func (r *CSVReporter) Name() string { return "csv" }

// Write renders the issues from result as CSV rows and writes them to w.
// The header row is: url,severity,check,message,detail.
// Issues are written in their existing order (assumed pre-sorted by severity).
func (r *CSVReporter) Write(_ context.Context, result *model.CrawlResult, w io.Writer) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{"url", "severity", "check", "message", "detail"}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	for i, iss := range result.Issues {
		row := []string{
			iss.URL,
			string(iss.Severity),
			iss.CheckName,
			iss.Message,
			iss.Detail,
		}
		if err := cw.Write(row); err != nil {
			return fmt.Errorf("writing CSV row %d: %w", i, err)
		}
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return fmt.Errorf("flushing CSV writer: %w", err)
	}

	return nil
}
