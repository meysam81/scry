// Package crawl implements the crawl subcommand for scry.
package crawl

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/internal/audit"
	"github.com/meysam81/scry/internal/cmdutil"
	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/crawler"
	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/internal/metrics"
)

// Command returns the cli.Command for the crawl subcommand.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "crawl",
		Usage: "Crawl a website and run audit checks.",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "depth",
				Aliases: []string{"d"},
				Value:   5,
				Usage:   "max crawl depth",
			},
			&cli.IntFlag{
				Name:  "max-pages",
				Value: 500,
				Usage: "page cap",
			},
			&cli.IntFlag{
				Name:    "concurrency",
				Aliases: []string{"c"},
				Value:   10,
				Usage:   "parallel fetchers",
			},
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
			&cli.BoolFlag{
				Name:  "ignore-robots",
				Value: false,
				Usage: "bypass robots.txt",
			},
			&cli.StringSliceFlag{
				Name:  "include",
				Usage: "glob patterns for include",
			},
			&cli.StringSliceFlag{
				Name:  "exclude",
				Usage: "glob patterns for exclude",
			},
			&cli.IntFlag{
				Name:  "rate-limit",
				Value: 50,
				Usage: "requests per second",
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
			&cli.StringFlag{
				Name:  "urls-file",
				Value: "",
				Usage: "file with URLs (one per line) for multi-domain crawl",
			},
			&cli.IntFlag{
				Name:  "parallel-domains",
				Value: 3,
				Usage: "number of domains to crawl in parallel",
			},
			&cli.StringFlag{
				Name:  "metrics-push",
				Value: "",
				Usage: "Prometheus Pushgateway URL for metrics",
			},
			&cli.StringFlag{
				Name:  "checkpoint",
				Value: "",
				Usage: "save crawl checkpoint to file (for resume)",
			},
			&cli.StringFlag{
				Name:  "resume",
				Value: "",
				Usage: "resume crawl from checkpoint file",
			},
			&cli.StringFlag{
				Name:  "incremental",
				Value: "",
				Usage: "incremental crawl cache file",
			},
		},
		Action: runCrawl,
	}
}

func runCrawl(ctx context.Context, cmd *cli.Command) error {
	l := logger.FromContext(ctx)

	cfg, err := config.LoadWithFile(cmd.String("config"))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	applyFlagOverrides(cmd, cfg)

	seedURL := cmd.Args().First()
	if seedURL == "" {
		return fmt.Errorf("missing required URL argument")
	}
	seedURL = cmdutil.NormalizeURL(seedURL)

	// Create fetcher.
	fetcher, cleanup, err := crawler.NewFetcher(cfg, l)
	if err != nil {
		return fmt.Errorf("create fetcher: %w", err)
	}
	defer cleanup()

	// Run crawler.
	l.Info().Str("url", seedURL).Msg("starting crawl")
	c := crawler.NewCrawler(cfg, fetcher, l)
	result, err := c.Run(ctx, seedURL)
	if err != nil {
		return cli.Exit(fmt.Sprintf("crawl failed: %v", err), 2)
	}
	l.Info().Int("pages", len(result.Pages)).Dur("duration", result.Duration).Msg("crawl complete")

	// Run audit checks.
	l.Info().Msg("running audit checks")
	registry := audit.DefaultRegistry(l)
	result.Issues = registry.RunAll(ctx, result.Pages)
	l.Info().Int("issues", len(result.Issues)).Msg("audit complete")

	// Push metrics if configured.
	if cfg.MetricsPushURL != "" {
		if err := metrics.PushMetrics(result, cfg.MetricsPushURL, "scry"); err != nil {
			l.Warn().Err(err).Msg("failed to push metrics")
		}
	}

	return cmdutil.ReportAndExit(ctx, cfg, result)
}

// applyFlagOverrides applies CLI flag values to the config, only when explicitly set.
func applyFlagOverrides(cmd *cli.Command, cfg *config.Config) {
	if cmd.IsSet("depth") {
		cfg.MaxDepth = cmd.Int("depth")
	}
	if cmd.IsSet("max-pages") {
		cfg.MaxPages = cmd.Int("max-pages")
	}
	if cmd.IsSet("concurrency") {
		cfg.Concurrency = cmd.Int("concurrency")
	}
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
	if cmd.IsSet("ignore-robots") {
		cfg.RespectRobots = !cmd.Bool("ignore-robots")
	}
	if cmd.IsSet("include") {
		cfg.IncludePatterns = cmd.StringSlice("include")
	}
	if cmd.IsSet("exclude") {
		cfg.ExcludePatterns = cmd.StringSlice("exclude")
	}
	if cmd.IsSet("rate-limit") {
		cfg.RateLimit = cmd.Int("rate-limit")
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

	if cmd.IsSet("parallel-domains") {
		cfg.ParallelDomains = cmd.Int("parallel-domains")
	}
	if cmd.IsSet("metrics-push") {
		cfg.MetricsPushURL = cmd.String("metrics-push")
	}

	if cmd.IsSet("checkpoint") {
		cfg.CheckpointFile = cmd.String("checkpoint")
	}
	if cmd.IsSet("resume") {
		cfg.ResumeFile = cmd.String("resume")
	}
	if cmd.IsSet("incremental") {
		cfg.IncrementalFile = cmd.String("incremental")
	}

	cmdutil.ApplyGlobalOverrides(cmd, cfg)
}
