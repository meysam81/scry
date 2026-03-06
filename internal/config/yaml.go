package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// envSet reports whether the given environment variable is explicitly set.
func envSet(name string) bool {
	_, ok := os.LookupEnv(name)
	return ok
}

// YAMLConfig represents the configuration file structure.
// Pointer fields distinguish unset from zero values.
type YAMLConfig struct {
	Crawl struct {
		MaxDepth      *int     `yaml:"max_depth"`
		MaxPages      *int     `yaml:"max_pages"`
		Concurrency   *int     `yaml:"concurrency"`
		RespectRobots *bool    `yaml:"respect_robots"`
		Exclude       []string `yaml:"exclude"`
		Include       []string `yaml:"include"`
		RateLimit     *int     `yaml:"rate_limit"`
		Timeout       *string  `yaml:"timeout"`
		UserAgent     *string  `yaml:"user_agent"`
	} `yaml:"crawl"`

	Output struct {
		Formats []string `yaml:"formats"`
		File    *string  `yaml:"file"`
		FailOn  *string  `yaml:"fail_on"`
	} `yaml:"output"`

	Lighthouse struct {
		Enabled  *bool   `yaml:"enabled"`
		Mode     *string `yaml:"mode"`
		Strategy *string `yaml:"strategy"`
	} `yaml:"lighthouse"`

	Browser struct {
		Enabled        *bool   `yaml:"enabled"`
		BrowserlessURL *string `yaml:"browserless_url"`
	} `yaml:"browser"`
}

// configFileName is the name of the YAML configuration file.
const configFileName = "scry.yml"

// loadYAML reads and parses the YAML configuration file at the given path.
func loadYAML(path string) (*YAMLConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	ycfg := &YAMLConfig{}
	if err := yaml.Unmarshal(data, ycfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}

	return ycfg, nil
}

// findConfigFile discovers the YAML config file using the following order:
//  1. If explicitPath is non-empty, return it directly.
//  2. Look for scry.yml in the current working directory.
//  3. Look for scry.yml in $HOME.
//  4. Return "" if none found.
func findConfigFile(explicitPath string) string {
	if explicitPath != "" {
		return explicitPath
	}

	cwd, err := os.Getwd()
	if err == nil {
		candidate := filepath.Join(cwd, configFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		candidate := filepath.Join(home, configFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// mergeYAML applies YAML configuration values to Config only when the YAML
// field is non-nil AND the corresponding env var is not explicitly set.
// This preserves the precedence: CLI > env > YAML > defaults.
func mergeYAML(cfg *Config, ycfg *YAMLConfig) error {
	// Crawl settings.
	if ycfg.Crawl.MaxDepth != nil && !envSet("SCRY_MAX_DEPTH") {
		cfg.MaxDepth = *ycfg.Crawl.MaxDepth
	}
	if ycfg.Crawl.MaxPages != nil && !envSet("SCRY_MAX_PAGES") {
		cfg.MaxPages = *ycfg.Crawl.MaxPages
	}
	if ycfg.Crawl.Concurrency != nil && !envSet("SCRY_CONCURRENCY") {
		cfg.Concurrency = *ycfg.Crawl.Concurrency
	}
	if ycfg.Crawl.RespectRobots != nil && !envSet("SCRY_RESPECT_ROBOTS") {
		cfg.RespectRobots = *ycfg.Crawl.RespectRobots
	}
	if ycfg.Crawl.RateLimit != nil && !envSet("SCRY_RATE_LIMIT") {
		cfg.RateLimit = *ycfg.Crawl.RateLimit
	}
	if ycfg.Crawl.UserAgent != nil && !envSet("SCRY_USER_AGENT") {
		cfg.UserAgent = *ycfg.Crawl.UserAgent
	}
	if ycfg.Crawl.Timeout != nil && !envSet("SCRY_REQUEST_TIMEOUT") {
		d, err := time.ParseDuration(*ycfg.Crawl.Timeout)
		if err != nil {
			return fmt.Errorf("parse crawl timeout %q: %w", *ycfg.Crawl.Timeout, err)
		}
		cfg.RequestTimeout = d
	}
	if len(ycfg.Crawl.Include) > 0 {
		cfg.IncludePatterns = ycfg.Crawl.Include
	}
	if len(ycfg.Crawl.Exclude) > 0 {
		cfg.ExcludePatterns = ycfg.Crawl.Exclude
	}

	// Output settings.
	if len(ycfg.Output.Formats) > 0 && !envSet("SCRY_OUTPUT") {
		cfg.OutputFormat = strings.Join(ycfg.Output.Formats, ",")
	}
	if ycfg.Output.File != nil && !envSet("SCRY_OUTPUT_FILE") {
		cfg.OutputFile = *ycfg.Output.File
	}
	if ycfg.Output.FailOn != nil && !envSet("SCRY_FAIL_ON") {
		cfg.FailOn = *ycfg.Output.FailOn
	}

	// Lighthouse settings.
	if ycfg.Lighthouse.Enabled != nil && !envSet("SCRY_LIGHTHOUSE") {
		cfg.LighthouseEnabled = *ycfg.Lighthouse.Enabled
	}
	if ycfg.Lighthouse.Mode != nil && !envSet("SCRY_LIGHTHOUSE_MODE") {
		cfg.LighthouseMode = *ycfg.Lighthouse.Mode
	}
	if ycfg.Lighthouse.Strategy != nil && !envSet("SCRY_PSI_STRATEGY") {
		cfg.PSIStrategy = *ycfg.Lighthouse.Strategy
	}

	// Browser settings.
	if ycfg.Browser.Enabled != nil && !envSet("SCRY_BROWSER_MODE") {
		cfg.BrowserMode = *ycfg.Browser.Enabled
	}
	if ycfg.Browser.BrowserlessURL != nil && !envSet("SCRY_BROWSERLESS_URL") {
		cfg.BrowserlessURL = *ycfg.Browser.BrowserlessURL
	}

	return nil
}

// LoadWithFile performs the full configuration load sequence:
//  1. Load environment variables and defaults via Load().
//  2. Discover the YAML config file.
//  3. If found, parse and merge YAML values (only non-nil fields).
//  4. Re-validate the merged configuration.
func LoadWithFile(configPath string) (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	path := findConfigFile(configPath)
	if path == "" {
		return cfg, nil
	}

	// When an explicit path was requested but does not exist, treat it as an error.
	if configPath != "" {
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("config file not found: %w", err)
		}
	}

	ycfg, err := loadYAML(path)
	if err != nil {
		return nil, err
	}

	if err := mergeYAML(cfg, ycfg); err != nil {
		return nil, fmt.Errorf("merge yaml config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
