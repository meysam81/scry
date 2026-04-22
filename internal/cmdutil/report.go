package cmdutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/core/model"
	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/internal/report"
)

// ReportAndExit writes reports to the configured outputs and returns an exit
// code error if issues exceed the fail-on threshold.
func ReportAndExit(ctx context.Context, cfg *config.Config, result *model.CrawlResult) error {
	l := logger.FromContext(ctx)

	if cfg.FilterSeverity != "" || cfg.FilterCategory != "" {
		result.Issues = report.FilterIssues(result.Issues, cfg.FilterSeverity, cfg.FilterCategory)
	}

	reporters := report.AllReporters()
	formats := cfg.OutputFormats()

	for _, format := range formats {
		r, ok := reporters[format]
		if !ok {
			l.Warn().Str("format", format).Msg("unknown output format, skipping")
			continue
		}

		if err := writeReport(ctx, r, result, cfg, format, formats); err != nil {
			return err
		}
	}

	exitCode := DetermineExitCode(ctx, result, cfg.FailOn)
	if exitCode != 0 {
		return cli.Exit(fmt.Sprintf("issues found at or above %q severity threshold", cfg.FailOn), exitCode)
	}

	return nil
}

// writeReport writes a single report format, properly scoping file open/close.
func writeReport(ctx context.Context, r report.Reporter, result *model.CrawlResult, cfg *config.Config, format string, formats []string) (writeErr error) {
	w := os.Stdout

	if format != "terminal" && cfg.OutputFile != "" {
		ext := "." + format
		path := cfg.OutputFile
		if len(formats) > 1 {
			path = cfg.OutputFile + ext
		}
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create output directory %s: %w", dir, err)
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return fmt.Errorf("create output file %s: %w", path, err)
		}
		defer func() {
			if cerr := f.Close(); cerr != nil && writeErr == nil {
				writeErr = cerr
			}
		}()
		w = f
	}

	if err := r.Write(ctx, result, w); err != nil {
		return fmt.Errorf("write %s report: %w", format, err)
	}

	return nil
}

// DetermineExitCode checks if any issues meet the fail-on threshold.
// An unrecognised failOn value is logged as a warning and treated as no threshold (exit 0).
func DetermineExitCode(ctx context.Context, result *model.CrawlResult, failOn string) int {
	if failOn == "" {
		return 0
	}

	if failOn == "any" {
		if len(result.Issues) > 0 {
			return 1
		}
		return 0
	}

	threshold := model.SeverityFromString(failOn)
	if threshold == "" {
		logger.FromContext(ctx).Warn().Str("fail_on", failOn).Msg("unknown fail-on value, ignoring")
		return 0
	}

	for _, issue := range result.Issues {
		if issue.Severity.AtLeast(threshold) {
			return 1
		}
	}

	return 0
}
