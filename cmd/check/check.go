// Package check implements the check subcommand for scry.
package check

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/internal/audit"
	"github.com/meysam81/scry/internal/cmdutil"
	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/crawler"
	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/internal/model"
)

// Command returns the cli.Command for the check subcommand.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "Run audit checks on a single URL.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "browser",
				Value: false,
				Usage: "enable rod headless mode",
			},
			&cli.StringFlag{
				Name:  "browserless-url",
				Value: "http://localhost:3000",
				Usage: "browserless endpoint URL",
			},
			&cli.BoolFlag{
				Name:  "lighthouse",
				Value: false,
				Usage: "enable lighthouse scoring",
			},
			&cli.StringFlag{
				Name:  "lighthouse-mode",
				Value: "psi",
				Usage: "lighthouse mode: psi|browserless",
			},
			&cli.StringFlag{
				Name:  "psi-key",
				Value: "",
				Usage: "PageSpeed Insights API key",
			},
			&cli.StringFlag{
				Name:  "psi-strategy",
				Value: "mobile",
				Usage: "PSI strategy: mobile|desktop",
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Value: 10 * time.Second,
				Usage: "per-request timeout",
			},
			&cli.StringFlag{
				Name:  "user-agent",
				Value: "scry/1.0",
				Usage: "HTTP user-agent string",
			},
			&cli.StringFlag{
				Name:  "filter-severity",
				Value: "",
				Usage: "comma-separated severity filter (e.g. critical,warning)",
			},
			&cli.StringFlag{
				Name:  "filter-category",
				Value: "",
				Usage: "comma-separated category filter (e.g. seo,performance)",
			},
			&cli.BoolFlag{
				Name:  "watch",
				Value: false,
				Usage: "re-run check on interval",
			},
			&cli.DurationFlag{
				Name:  "watch-interval",
				Value: 30 * time.Second,
				Usage: "watch interval",
			},
		},
		Action: runCheck,
	}
}

func runCheck(ctx context.Context, cmd *cli.Command) error {
	l := logger.FromContext(ctx)

	cfg, err := config.LoadWithFile(cmd.String("config"))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	applyFlagOverrides(cmd, cfg)

	targetURL := cmd.Args().First()
	if targetURL == "" {
		return fmt.Errorf("missing required URL argument")
	}
	targetURL = cmdutil.NormalizeURL(targetURL)

	// Create fetcher.
	fetcher, cleanup, err := crawler.NewFetcher(cfg, l)
	if err != nil {
		return fmt.Errorf("create fetcher: %w", err)
	}
	defer cleanup()

	// Run the check once.
	if err := runSingleCheck(ctx, cfg, fetcher, targetURL); err != nil {
		return err
	}

	// If watch mode is enabled, re-run on interval.
	if cmd.Bool("watch") {
		interval := cmd.Duration("watch-interval")
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				// Clear screen.
				if _, err := os.Stdout.WriteString("\033[2J\033[H"); err != nil {
					l.Warn().Err(err).Msg("failed to clear screen")
				}
				if err := runSingleCheck(ctx, cfg, fetcher, targetURL); err != nil {
					l.Error().Err(err).Msg("watch iteration failed")
				}
			}
		}
	}

	return nil
}

// runSingleCheck performs a single fetch + audit cycle and writes reports.
func runSingleCheck(ctx context.Context, cfg *config.Config, fetcher crawler.Fetcher, targetURL string) error {
	l := logger.FromContext(ctx)

	l.Info().Str("url", targetURL).Msg("fetching URL")
	start := time.Now()
	page, err := fetcher.Fetch(ctx, targetURL)
	if err != nil {
		return cli.Exit(fmt.Sprintf("fetch failed: %v", err), 2)
	}
	duration := time.Since(start)
	l.Info().Int("status", page.StatusCode).Dur("duration", page.FetchDuration).Msg("fetch complete")

	// Run audit checks.
	l.Info().Msg("running audit checks")
	registry := audit.DefaultRegistry(l)
	issues := registry.RunAll(ctx, []*model.Page{page})
	l.Info().Int("issues", len(issues)).Msg("audit complete")

	// Wrap in CrawlResult for reporter compatibility.
	result := &model.CrawlResult{
		SeedURL:   targetURL,
		Pages:     []*model.Page{page},
		Issues:    issues,
		CrawledAt: start,
		Duration:  duration,
	}

	return cmdutil.ReportAndExit(ctx, cfg, result)
}

// applyFlagOverrides applies CLI flag values to the config, only when explicitly set.
func applyFlagOverrides(cmd *cli.Command, cfg *config.Config) {
	if cmd.IsSet("browser") {
		cfg.BrowserMode = cmd.Bool("browser")
	}
	if cmd.IsSet("browserless-url") {
		cfg.BrowserlessURL = cmd.String("browserless-url")
	}
	if cmd.IsSet("lighthouse") {
		cfg.LighthouseEnabled = cmd.Bool("lighthouse")
	}
	if cmd.IsSet("lighthouse-mode") {
		cfg.LighthouseMode = cmd.String("lighthouse-mode")
	}
	if cmd.IsSet("psi-key") {
		cfg.PSIAPIKey = cmd.String("psi-key")
	}
	if cmd.IsSet("psi-strategy") {
		cfg.PSIStrategy = cmd.String("psi-strategy")
	}
	if cmd.IsSet("timeout") {
		cfg.RequestTimeout = cmd.Duration("timeout")
	}
	if cmd.IsSet("user-agent") {
		cfg.UserAgent = cmd.String("user-agent")
	}
	if cmd.IsSet("filter-severity") {
		cfg.FilterSeverity = cmd.String("filter-severity")
	}
	if cmd.IsSet("filter-category") {
		cfg.FilterCategory = cmd.String("filter-category")
	}

	cmdutil.ApplyGlobalOverrides(cmd, cfg)
}
