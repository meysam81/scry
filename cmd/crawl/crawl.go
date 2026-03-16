// Package crawl implements the crawl subcommand for scry.
package crawl

import (
	"context"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/meysam81/scry/internal/audit"
	"github.com/meysam81/scry/internal/baseline"
	"github.com/meysam81/scry/internal/cmdutil"
	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/crawler"
	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/internal/metrics"
	"github.com/meysam81/scry/internal/rules"
)

var (
	flagDepth           int
	flagMaxPages        int
	flagConcurrency     int
	flagBrowser         bool
	flagBrowserlessURL  string
	flagLighthouse      bool
	flagLighthouseMode  string
	flagPSIKey          string
	flagPSIStrategy     string
	flagIgnoreRobots    bool
	flagInclude         []string
	flagExclude         []string
	flagRateLimit       int
	flagTimeout         time.Duration
	flagUserAgent       string
	flagFilterSeverity  string
	flagFilterCategory  string
	flagURLsFile        string
	flagParallelDomains int
	flagMetricsPush     string
	flagCheckpoint      string
	flagResume          string
	flagIncremental     string
	flagRulesFile       string
	flagSaveBaseline    string
	flagCompareBaseline string
)

// Command returns the cli.Command for the crawl subcommand.
func Command() *cli.Command {
	return &cli.Command{
		Name:  "crawl",
		Usage: "Crawl a website and run audit checks.",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "depth",
				Aliases:     []string{"d"},
				Value:       5,
				Usage:       "max crawl depth",
				Destination: &flagDepth,
			},
			&cli.IntFlag{
				Name:        "max-pages",
				Value:       500,
				Usage:       "page cap",
				Destination: &flagMaxPages,
			},
			&cli.IntFlag{
				Name:        "concurrency",
				Aliases:     []string{"c"},
				Value:       10,
				Usage:       "parallel fetchers",
				Destination: &flagConcurrency,
			},
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
			&cli.BoolFlag{
				Name:        "ignore-robots",
				Value:       false,
				Usage:       "bypass robots.txt",
				Destination: &flagIgnoreRobots,
			},
			&cli.StringSliceFlag{
				Name:        "include",
				Usage:       "glob patterns for include",
				Destination: &flagInclude,
			},
			&cli.StringSliceFlag{
				Name:        "exclude",
				Usage:       "glob patterns for exclude",
				Destination: &flagExclude,
			},
			&cli.IntFlag{
				Name:        "rate-limit",
				Value:       50,
				Usage:       "requests per second",
				Destination: &flagRateLimit,
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
			&cli.StringFlag{
				Name:        "urls-file",
				Value:       "",
				Usage:       "file with URLs (one per line) for multi-domain crawl",
				Destination: &flagURLsFile,
			},
			&cli.IntFlag{
				Name:        "parallel-domains",
				Value:       3,
				Usage:       "number of domains to crawl in parallel",
				Destination: &flagParallelDomains,
			},
			&cli.StringFlag{
				Name:        "metrics-push",
				Value:       "",
				Usage:       "Prometheus Pushgateway URL for metrics",
				Destination: &flagMetricsPush,
			},
			&cli.StringFlag{
				Name:        "checkpoint",
				Value:       "",
				Usage:       "save crawl checkpoint to file (for resume)",
				Destination: &flagCheckpoint,
			},
			&cli.StringFlag{
				Name:        "resume",
				Value:       "",
				Usage:       "resume crawl from checkpoint file",
				Destination: &flagResume,
			},
			&cli.StringFlag{
				Name:        "incremental",
				Value:       "",
				Usage:       "incremental crawl cache file",
				Destination: &flagIncremental,
			},
			&cli.StringFlag{
				Name:        "rules",
				Value:       "",
				Usage:       "path to CEL custom rules YAML file",
				Destination: &flagRulesFile,
			},
			&cli.StringFlag{
				Name:        "save-baseline",
				Value:       "",
				Usage:       "save issues to baseline file for future comparison",
				Destination: &flagSaveBaseline,
			},
			&cli.StringFlag{
				Name:        "compare-baseline",
				Value:       "",
				Usage:       "compare current issues against a saved baseline",
				Destination: &flagCompareBaseline,
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
	l.Info().Int("pages", len(result.Pages)).Str("duration", result.Duration.String()).Msg("crawl complete")

	// Run audit checks.
	l.Info().Msg("running audit checks")
	registry := audit.DefaultRegistry(l)

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

	result.Issues = registry.RunAll(ctx, result.Pages)
	l.Info().Int("issues", len(result.Issues)).Msg("audit complete")

	// Save baseline if configured (before compare, so the full issue set is captured).
	if cfg.SaveBaselineFile != "" {
		if err := baseline.Save(cfg.SaveBaselineFile, result); err != nil {
			return fmt.Errorf("save baseline: %w", err)
		}
		l.Info().Str("path", cfg.SaveBaselineFile).Msg("baseline saved")
	}

	// Compare against baseline if configured.
	if cfg.CompareBaselineFile != "" {
		bl, err := baseline.Load(cfg.CompareBaselineFile)
		if err != nil {
			return fmt.Errorf("load baseline: %w", err)
		}
		diff := baseline.Diff(bl, result)
		l.Info().
			Int("new", len(diff.New)).
			Int("resolved", len(diff.Resolved)).
			Int("existing", len(diff.Existing)).
			Msg("baseline comparison")
		result.Issues = diff.New
	}

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
		cfg.MaxDepth = flagDepth
	}
	if cmd.IsSet("max-pages") {
		cfg.MaxPages = flagMaxPages
	}
	if cmd.IsSet("concurrency") {
		cfg.Concurrency = flagConcurrency
	}
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
	if cmd.IsSet("ignore-robots") {
		cfg.RespectRobots = !flagIgnoreRobots
	}
	if cmd.IsSet("include") {
		cfg.IncludePatterns = flagInclude
	}
	if cmd.IsSet("exclude") {
		cfg.ExcludePatterns = flagExclude
	}
	if cmd.IsSet("rate-limit") {
		cfg.RateLimit = flagRateLimit
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

	if cmd.IsSet("parallel-domains") {
		cfg.ParallelDomains = flagParallelDomains
	}
	if cmd.IsSet("metrics-push") {
		cfg.MetricsPushURL = flagMetricsPush
	}

	if cmd.IsSet("checkpoint") {
		cfg.CheckpointFile = flagCheckpoint
	}
	if cmd.IsSet("resume") {
		cfg.ResumeFile = flagResume
	}
	if cmd.IsSet("incremental") {
		cfg.IncrementalFile = flagIncremental
	}
	if cmd.IsSet("rules") {
		cfg.RulesFile = flagRulesFile
	}
	if cmd.IsSet("save-baseline") {
		cfg.SaveBaselineFile = flagSaveBaseline
	}
	if cmd.IsSet("compare-baseline") {
		cfg.CompareBaselineFile = flagCompareBaseline
	}

	cmdutil.ApplyGlobalOverrides(cmd, cfg)
}
