package cmdutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/model"
	"github.com/meysam81/scry/internal/report"
)

// ReportAndExit writes reports to the configured outputs and returns an exit
// code error if issues exceed the fail-on threshold.
func ReportAndExit(ctx context.Context, cfg *config.Config, result *model.CrawlResult) error {
	reporters := report.AllReporters()
	formats := cfg.OutputFormats()

	for _, format := range formats {
		r, ok := reporters[format]
		if !ok {
			log.Warn().Str("format", format).Msg("unknown output format, skipping")
			continue
		}

		if err := writeReport(ctx, r, result, cfg, format, formats); err != nil {
			return err
		}
	}

	exitCode := DetermineExitCode(result, cfg.FailOn)
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
		f, err := os.Create(path)
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
func DetermineExitCode(result *model.CrawlResult, failOn string) int {
	if failOn == "" {
		return 0
	}

	threshold := model.SeverityFromString(failOn)
	if threshold == "" && failOn == "any" {
		if len(result.Issues) > 0 {
			return 1
		}
		return 0
	}

	for _, issue := range result.Issues {
		if issue.Severity.AtLeast(threshold) {
			return 1
		}
	}

	return 0
}
