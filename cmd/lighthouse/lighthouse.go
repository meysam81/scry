// Package lighthouse implements the lighthouse subcommand for scry.
package lighthouse

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/internal/cmdutil"
	"github.com/meysam81/scry/internal/config"
	lh "github.com/meysam81/scry/internal/lighthouse"
	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/internal/model"
)

var (
	flagMode     string
	flagPSIKey   string
	flagStrategy string
	flagURLsFile string
)

// Command returns the cli.Command for the lighthouse subcommand.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "lighthouse",
		Usage: "Run Lighthouse audits on one or more URLs.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "mode",
				Value:       "psi",
				Usage:       "lighthouse mode: psi|browserless",
				Destination: &flagMode,
			},
			&cli.StringFlag{
				Name:        "psi-key",
				Value:       "",
				Usage:       "PageSpeed Insights API key",
				Destination: &flagPSIKey,
			},
			&cli.StringFlag{
				Name:        "strategy",
				Value:       "mobile",
				Usage:       "PSI strategy: mobile|desktop",
				Destination: &flagStrategy,
			},
			&cli.StringFlag{
				Name:        "urls-file",
				Value:       "",
				Usage:       "path to file with URLs (one per line)",
				Destination: &flagURLsFile,
			},
		},
		Action: runLighthouse,
	}
}

func runLighthouse(ctx context.Context, cmd *cli.Command) error {
	l := logger.FromContext(ctx)

	cfg, err := config.LoadWithFile(cmd.String("config"))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	applyFlagOverrides(cmd, cfg)

	// Collect target URLs from args and/or --urls-file.
	urls, err := collectURLs(cmd)
	if err != nil {
		return err
	}
	if len(urls) == 0 {
		return fmt.Errorf("no URLs provided; pass a URL argument or use --urls-file")
	}

	// Create the runner.
	runner, err := lh.NewRunner(cfg, l)
	if err != nil {
		return fmt.Errorf("create lighthouse runner: %w", err)
	}

	// Run audits.
	start := time.Now()
	var results []model.LighthouseResult
	for _, u := range urls {
		l.Info().Str("url", u).Msg("running lighthouse audit")
		result, err := runner.Run(ctx, u)
		if err != nil {
			l.Warn().Err(err).Str("url", u).Msg("lighthouse audit failed, skipping")
			continue
		}
		results = append(results, *result)
		l.Info().
			Str("url", u).
			Float64("perf", result.PerformanceScore).
			Float64("a11y", result.AccessibilityScore).
			Float64("seo", result.SEOScore).
			Msg("audit complete")
	}

	// Convert scores to issues.
	var issues []model.Issue
	for i := range results {
		issues = append(issues, lh.ScoreToIssues(&results[i])...)
	}

	// Build the CrawlResult.
	seedURL := urls[0]
	crawlResult := &model.CrawlResult{
		SeedURL:    seedURL,
		Issues:     issues,
		Lighthouse: results,
		CrawledAt:  start,
		Duration:   time.Since(start),
	}

	return cmdutil.ReportAndExit(ctx, cfg, crawlResult)
}

// collectURLs gathers URLs from the positional args and the --urls-file flag.
func collectURLs(cmd *cli.Command) ([]string, error) {
	var urls []string

	// Add positional args.
	for i := range cmd.Args().Len() {
		u := strings.TrimSpace(cmd.Args().Get(i))
		if u != "" {
			urls = append(urls, cmdutil.NormalizeURL(u))
		}
	}

	// Read from --urls-file if provided.
	urlsFile := flagURLsFile
	if urlsFile != "" {
		fileURLs, err := readURLsFile(urlsFile)
		if err != nil {
			return nil, fmt.Errorf("read urls file: %w", err)
		}
		urls = append(urls, fileURLs...)
	}

	return urls, nil
}

// maxURLFileLines is the maximum number of lines allowed in a URL file.
const maxURLFileLines = 10_000

// readURLsFile reads a file of URLs, one per line, skipping empty lines and comments.
func readURLsFile(path string) (_ []string, readErr error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && readErr == nil {
			readErr = cerr
		}
	}()

	var urls []string
	lineCount := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineCount++
		if lineCount > maxURLFileLines {
			return nil, fmt.Errorf("urls file %s exceeds %d lines", path, maxURLFileLines)
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, cmdutil.NormalizeURL(line))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}

// applyFlagOverrides applies CLI flag values to the config.
func applyFlagOverrides(cmd *cli.Command, cfg *config.Config) {
	cmdutil.ApplyGlobalOverrides(cmd, cfg)
	if cmd.IsSet("mode") {
		cfg.LighthouseMode = flagMode
	}
	if cmd.IsSet("psi-key") {
		cfg.PSIAPIKey = flagPSIKey
	}
	if cmd.IsSet("strategy") {
		cfg.PSIStrategy = flagStrategy
	}
}
