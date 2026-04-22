package report

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/meysam81/scry/core/model"
)

// JSONLReporter writes the CrawlResult as JSON Lines — one JSON object per line.
type JSONLReporter struct{}

// Name returns "jsonl".
func (r *JSONLReporter) Name() string { return "jsonl" }

// jsonlIssueLine represents a single issue line in the JSONL output.
type jsonlIssueLine struct {
	CheckName string         `json:"check_name"`
	Severity  model.Severity `json:"severity"`
	Message   string         `json:"message"`
	URL       string         `json:"url"`
	Detail    string         `json:"detail,omitempty"`
}

// jsonlSummaryLine is the trailing summary line in the JSONL output.
type jsonlSummaryLine struct {
	Type     string `json:"type"`
	SeedURL  string `json:"seed_url"`
	Pages    int    `json:"pages"`
	Issues   int    `json:"issues"`
	Critical int    `json:"critical"`
	Warning  int    `json:"warning"`
	Info     int    `json:"info"`
	Duration string `json:"duration"`
}

// Write renders result as JSONL (one JSON object per line) and writes it to w.
func (r *JSONLReporter) Write(_ context.Context, result *model.CrawlResult, w io.Writer) error {
	if result == nil {
		return nil
	}

	var critCount, warnCount, infoCount int

	for i, iss := range result.Issues {
		line := jsonlIssueLine{
			CheckName: iss.CheckName,
			Severity:  iss.Severity,
			Message:   iss.Message,
			URL:       iss.URL,
			Detail:    iss.Detail,
		}

		data, err := json.Marshal(line)
		if err != nil {
			return fmt.Errorf("marshalling JSONL issue line %d: %w", i, err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("writing JSONL issue line %d: %w", i, err)
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			return fmt.Errorf("writing JSONL newline for line %d: %w", i, err)
		}

		switch iss.Severity {
		case model.SeverityCritical:
			critCount++
		case model.SeverityWarning:
			warnCount++
		case model.SeverityInfo:
			infoCount++
		}
	}

	// Write trailing summary line.
	summary := jsonlSummaryLine{
		Type:     "summary",
		SeedURL:  result.SeedURL,
		Pages:    len(result.Pages),
		Issues:   len(result.Issues),
		Critical: critCount,
		Warning:  warnCount,
		Info:     infoCount,
		Duration: result.Duration.String(),
	}

	data, err := json.Marshal(summary)
	if err != nil {
		return fmt.Errorf("marshalling JSONL summary line: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing JSONL summary line: %w", err)
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("writing JSONL trailing newline: %w", err)
	}

	return nil
}
