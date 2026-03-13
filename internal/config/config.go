// Package config loads and validates scry configuration from environment variables.
package config

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config holds all configuration for scry.
type Config struct {
	// Crawler settings.
	MaxDepth       int           `env:"SCRY_MAX_DEPTH"       envDefault:"5"`
	MaxPages       int           `env:"SCRY_MAX_PAGES"       envDefault:"500"`
	Concurrency    int           `env:"SCRY_CONCURRENCY"     envDefault:"10"`
	RequestTimeout time.Duration `env:"SCRY_REQUEST_TIMEOUT" envDefault:"10s"`
	RateLimit      int           `env:"SCRY_RATE_LIMIT"      envDefault:"50"`
	UserAgent      string        `env:"SCRY_USER_AGENT"      envDefault:"scry/1.0"`
	RespectRobots  bool          `env:"SCRY_RESPECT_ROBOTS"  envDefault:"true"`

	// Output settings.
	OutputFormat string `env:"SCRY_OUTPUT"      envDefault:"terminal"`
	OutputFile   string `env:"SCRY_OUTPUT_FILE" envDefault:""`
	FailOn       string `env:"SCRY_FAIL_ON"     envDefault:""`

	// Browser mode.
	BrowserMode    bool   `env:"SCRY_BROWSER_MODE"    envDefault:"false"`
	BrowserlessURL string `env:"SCRY_BROWSERLESS_URL" envDefault:"http://localhost:3000"`

	// Lighthouse.
	LighthouseEnabled bool   `env:"SCRY_LIGHTHOUSE"      envDefault:"false"`
	LighthouseMode    string `env:"SCRY_LIGHTHOUSE_MODE"  envDefault:"psi"`
	PSIAPIKey         string `env:"SCRY_PSI_API_KEY"      envDefault:""`
	PSIStrategy       string `env:"SCRY_PSI_STRATEGY"     envDefault:"mobile"`

	// Logging.
	LogLevel  string `env:"SCRY_LOG_LEVEL"  envDefault:"info"`
	LogFormat string `env:"SCRY_LOG_FORMAT" envDefault:"pretty"`

	// Filtering.
	FilterSeverity string `env:"SCRY_FILTER_SEVERITY" envDefault:""`
	FilterCategory string `env:"SCRY_FILTER_CATEGORY" envDefault:""`

	// Parallel domains.
	ParallelDomains int `env:"SCRY_PARALLEL_DOMAINS" envDefault:"3"`

	// Metrics.
	MetricsPushURL string `env:"SCRY_METRICS_PUSH_URL" envDefault:""`

	// Checkpoint / Incremental.
	CheckpointFile  string `env:"SCRY_CHECKPOINT_FILE"  envDefault:""`
	ResumeFile      string `env:"SCRY_RESUME_FILE"      envDefault:""`
	IncrementalFile string `env:"SCRY_INCREMENTAL_FILE" envDefault:""`

	// Custom rules.
	RulesFile string `env:"SCRY_RULES_FILE" envDefault:""`

	// Baseline comparison.
	SaveBaselineFile    string `env:"SCRY_SAVE_BASELINE"    envDefault:""`
	CompareBaselineFile string `env:"SCRY_COMPARE_BASELINE" envDefault:""`

	// CLI-only fields (not from env).
	IncludePatterns []string `env:"-"`
	ExcludePatterns []string `env:"-"`
}

// validOutputFormats lists the accepted output format values.
var validOutputFormats = map[string]bool{
	"terminal": true,
	"json":     true,
	"csv":      true,
	"markdown": true,
	"html":     true,
	"sarif":    true,
	"junit":    true,
	"jsonl":    true,
	"pdf":      true,
}

// Load parses environment variables into a Config and validates it.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// OutputFormats splits OutputFormat by comma and trims whitespace from each element.
func (c *Config) OutputFormats() []string {
	parts := strings.Split(c.OutputFormat, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// validate checks that all config values are within acceptable ranges and sets.
func (c *Config) validate() error {
	if c.MaxDepth < 1 {
		return fmt.Errorf("validate config: max depth must be >= 1, got %d", c.MaxDepth)
	}
	if c.MaxPages < 1 {
		return fmt.Errorf("validate config: max pages must be >= 1, got %d", c.MaxPages)
	}
	if c.Concurrency < 1 {
		return fmt.Errorf("validate config: concurrency must be >= 1, got %d", c.Concurrency)
	}
	if c.RateLimit < 1 {
		return fmt.Errorf("validate config: rate limit must be >= 1, got %d", c.RateLimit)
	}
	if c.ParallelDomains < 1 {
		return fmt.Errorf("validate config: parallel domains must be >= 1, got %d", c.ParallelDomains)
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("validate config: request timeout must be > 0, got %s", c.RequestTimeout)
	}

	for _, f := range c.OutputFormats() {
		if !validOutputFormats[f] {
			return fmt.Errorf("validate config: invalid output format %q", f)
		}
	}

	switch c.FailOn {
	case "", "critical", "warning", "any":
		// valid
	default:
		return fmt.Errorf("validate config: invalid fail-on value %q", c.FailOn)
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
		// valid
	default:
		return fmt.Errorf("validate config: invalid log level %q", c.LogLevel)
	}

	switch c.LogFormat {
	case "pretty", "json":
		// valid
	default:
		return fmt.Errorf("validate config: invalid log format %q", c.LogFormat)
	}

	switch c.LighthouseMode {
	case "psi", "browserless":
		// valid
	default:
		return fmt.Errorf("validate config: invalid lighthouse mode %q", c.LighthouseMode)
	}

	switch c.PSIStrategy {
	case "mobile", "desktop":
		// valid
	default:
		return fmt.Errorf("validate config: invalid psi strategy %q", c.PSIStrategy)
	}

	for _, p := range c.IncludePatterns {
		if _, err := filepath.Match(p, ""); err != nil {
			return fmt.Errorf("validate config: invalid include pattern %q: %w", p, err)
		}
	}
	for _, p := range c.ExcludePatterns {
		if _, err := filepath.Match(p, ""); err != nil {
			return fmt.Errorf("validate config: invalid exclude pattern %q: %w", p, err)
		}
	}

	return nil
}
