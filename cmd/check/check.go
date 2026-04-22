// Package check implements the check subcommand for scry.
package check

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/core/checks"
	"github.com/meysam81/scry/core/model"
	"github.com/meysam81/scry/core/rules"
	"github.com/meysam81/scry/internal/cmdutil"
	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/crawler"
	"github.com/meysam81/scry/internal/logger"
)

var (
	flagBrowser        bool
	flagBrowserlessURL string
	flagLighthouse     bool
	flagLighthouseMode string
	flagPSIKey         string
	flagPSIStrategy    string
	flagTimeout        time.Duration
	flagUserAgent      string
	flagFilterSeverity string
	flagFilterCategory string
	flagWatch          bool
	flagWatchInterval  time.Duration
	flagRulesFile      string
	flagSchemaPath     string
)

// Command returns the cli.Command for the check subcommand.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "check",
		Usage: "Run audit checks on a single URL.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "browser",
				Value:       false,
				Usage:       "enable rod headless mode",
				Destination: &flagBrowser,
			},
			&cli.StringFlag{
				Name:        "browserless-url",
				Value:       "http://localhost:3000",
				Usage:       "browserless endpoint URL",
				Destination: &flagBrowserlessURL,
			},
			&cli.BoolFlag{
				Name:        "lighthouse",
				Value:       false,
				Usage:       "enable lighthouse scoring",
				Destination: &flagLighthouse,
			},
			&cli.StringFlag{
				Name:        "lighthouse-mode",
				Value:       "psi",
				Usage:       "lighthouse mode: psi|browserless",
				Destination: &flagLighthouseMode,
			},
			&cli.StringFlag{
				Name:        "psi-key",
				Value:       "",
				Usage:       "PageSpeed Insights API key",
				Destination: &flagPSIKey,
			},
			&cli.StringFlag{
				Name:        "psi-strategy",
				Value:       "mobile",
				Usage:       "PSI strategy: mobile|desktop",
				Destination: &flagPSIStrategy,
			},
			&cli.DurationFlag{
				Name:        "timeout",
				Value:       10 * time.Second,
				Usage:       "per-request timeout",
				Destination: &flagTimeout,
			},
			&cli.StringFlag{
				Name:        "user-agent",
				Value:       "scry/1.0",
				Usage:       "HTTP user-agent string",
				Destination: &flagUserAgent,
			},
			&cli.StringFlag{
				Name:        "filter-severity",
				Value:       "",
				Usage:       "comma-separated severity filter (e.g. critical,warning)",
				Destination: &flagFilterSeverity,
			},
			&cli.StringFlag{
				Name:        "filter-category",
				Value:       "",
				Usage:       "comma-separated category filter (e.g. seo,performance)",
				Destination: &flagFilterCategory,
			},
			&cli.BoolFlag{
				Name:        "watch",
				Value:       false,
				Usage:       "re-run check on interval",
				Destination: &flagWatch,
			},
			&cli.DurationFlag{
				Name:        "watch-interval",
				Value:       30 * time.Second,
				Usage:       "watch interval",
				Destination: &flagWatchInterval,
			},
			&cli.StringFlag{
				Name:        "rules",
				Value:       "",
				Usage:       "path to CEL custom rules YAML file",
				Destination: &flagRulesFile,
			},
			&cli.StringFlag{
				Name:        "schema-path",
				Value:       "",
				Usage:       "path to custom Schema.org definitions JSON file",
				Sources:     cli.EnvVars("SCRY_SCHEMA_PATH"),
				Destination: &flagSchemaPath,
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
	if flagWatch {
		interval := flagWatchInterval
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
	registry := checks.DefaultRegistry(l, cfg.SchemaPath)

	// Load and register custom CEL rules if configured.
	if cfg.RulesFile != "" {
		rf, err := rules.LoadRuleFile(cfg.RulesFile)
		if err != nil {
			return fmt.Errorf("load rules file: %w", err)
		}
		engine, err := rules.NewEngine(rf.Rules, l)
		if err != nil {
			return fmt.Errorf("compile rules: %w", err)
		}
		l.Info().Int("count", engine.RuleCount()).Msg("loaded custom CEL rules")
		registry.Register(rules.NewRuleChecker(engine))
	}

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
		cfg.BrowserMode = flagBrowser
	}
	if cmd.IsSet("browserless-url") {
		cfg.BrowserlessURL = flagBrowserlessURL
	}
	if cmd.IsSet("lighthouse") {
		cfg.LighthouseEnabled = flagLighthouse
	}
	if cmd.IsSet("lighthouse-mode") {
		cfg.LighthouseMode = flagLighthouseMode
	}
	if cmd.IsSet("psi-key") {
		cfg.PSIAPIKey = flagPSIKey
	}
	if cmd.IsSet("psi-strategy") {
		cfg.PSIStrategy = flagPSIStrategy
	}
	if cmd.IsSet("timeout") {
		cfg.RequestTimeout = flagTimeout
	}
	if cmd.IsSet("user-agent") {
		cfg.UserAgent = flagUserAgent
	}
	if cmd.IsSet("filter-severity") {
		cfg.FilterSeverity = flagFilterSeverity
	}
	if cmd.IsSet("filter-category") {
		cfg.FilterCategory = flagFilterCategory
	}
	if cmd.IsSet("rules") {
		cfg.RulesFile = flagRulesFile
	}
	if cmd.IsSet("schema-path") {
		cfg.SchemaPath = flagSchemaPath
	}

	cmdutil.ApplyGlobalOverrides(cmd, cfg)
}
