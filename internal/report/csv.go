package report

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"

	"github.com/meysam81/scry/internal/model"
)

// sanitizeCSVCell prevents CSV formula injection by prefixing cells that start
// with characters interpreted as formulas by spreadsheet applications.
func sanitizeCSVCell(s string) string {
	if len(s) == 0 {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "\t" + s
	}
	return s
}

// CSVReporter writes the CrawlResult issues as a CSV document.
type CSVReporter struct{}

// Name returns "csv".
func (r *CSVReporter) Name() string { return "csv" }

// Write renders the issues from result as CSV rows and writes them to w.
// The header row is: url,severity,check,message,detail.
// Issues are written in their existing order (assumed pre-sorted by severity).
func (r *CSVReporter) Write(_ context.Context, result *model.CrawlResult, w io.Writer) error {
	if result == nil {
		return nil
	}
	cw := csv.NewWriter(w)

	header := []string{"url", "severity", "check", "message", "detail"}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	for i, iss := range result.Issues {
		row := []string{
			sanitizeCSVCell(iss.URL),
			sanitizeCSVCell(string(iss.Severity)),
			sanitizeCSVCell(iss.CheckName),
			sanitizeCSVCell(iss.Message),
			sanitizeCSVCell(iss.Detail),
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
