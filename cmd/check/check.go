// Package check implements the check subcommand for scry.
package check

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/internal/audit"
	"github.com/meysam81/scry/internal/cmdutil"
	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/crawler"
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
		},
		Action: runCheck,
	}
}

func runCheck(ctx context.Context, cmd *cli.Command) error {
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
	fetcher, cleanup, err := crawler.NewFetcher(cfg)
	if err != nil {
		return fmt.Errorf("create fetcher: %w", err)
	}
	defer cleanup()

	// Fetch the single URL.
	log.Info().Str("url", targetURL).Msg("fetching URL")
	start := time.Now()
	page, err := fetcher.Fetch(ctx, targetURL)
	if err != nil {
		return cli.Exit(fmt.Sprintf("fetch failed: %v", err), 2)
	}
	duration := time.Since(start)
	log.Info().Int("status", page.StatusCode).Dur("duration", page.FetchDuration).Msg("fetch complete")

	// Run audit checks.
	log.Info().Msg("running audit checks")
	registry := audit.DefaultRegistry()
	issues := registry.RunAll(ctx, []*model.Page{page})
	log.Info().Int("issues", len(issues)).Msg("audit complete")

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
		cfg.PSIApiKey = cmd.String("psi-key")
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

	// Apply global flags from parent command.
	if cmd.IsSet("output") {
		cfg.OutputFormat = cmd.String("output")
	}
	if cmd.IsSet("output-file") {
		cfg.OutputFile = cmd.String("output-file")
	}
	if cmd.IsSet("fail-on") {
		cfg.FailOn = cmd.String("fail-on")
	}
}
