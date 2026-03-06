package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPartialYAML(t *testing.T) {
	clearScryEnv(t)

	dir := t.TempDir()
	yamlContent := `
crawl:
  max_depth: 3
  concurrency: 4
`
	path := filepath.Join(dir, "scry.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := LoadWithFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Overridden by YAML.
	if cfg.MaxDepth != 3 {
		t.Errorf("MaxDepth = %d, want 3", cfg.MaxDepth)
	}
	if cfg.Concurrency != 4 {
		t.Errorf("Concurrency = %d, want 4", cfg.Concurrency)
	}

	// Kept defaults.
	if cfg.MaxPages != 500 {
		t.Errorf("MaxPages = %d, want 500 (default)", cfg.MaxPages)
	}
	if cfg.RateLimit != 50 {
		t.Errorf("RateLimit = %d, want 50 (default)", cfg.RateLimit)
	}
	if cfg.UserAgent != "scry/1.0" {
		t.Errorf("UserAgent = %q, want %q (default)", cfg.UserAgent, "scry/1.0")
	}
}

func TestMergePrecedenceEnvWinsOverYAML(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_MAX_DEPTH", "20")

	dir := t.TempDir()
	yamlContent := `
crawl:
  max_depth: 3
`
	path := filepath.Join(dir, "scry.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := LoadWithFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Env var SCRY_MAX_DEPTH=20 takes precedence over YAML max_depth=3.
	if cfg.MaxDepth != 20 {
		t.Errorf("MaxDepth = %d, want 20 (env wins over YAML)", cfg.MaxDepth)
	}
}

func TestMergePrecedenceYAMLOverridesDefault(t *testing.T) {
	clearScryEnv(t)

	dir := t.TempDir()
	yamlContent := `
crawl:
  max_depth: 3
`
	path := filepath.Join(dir, "scry.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := LoadWithFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No env var set, so YAML max_depth=3 overrides the default of 5.
	if cfg.MaxDepth != 3 {
		t.Errorf("MaxDepth = %d, want 3 (YAML overrides default)", cfg.MaxDepth)
	}
}

func TestMissingExplicitFile(t *testing.T) {
	clearScryEnv(t)

	_, err := LoadWithFile("/nonexistent/path/scry.yml")
	if err == nil {
		t.Fatal("expected error for missing explicit config file, got nil")
	}
}

func TestNoConfigFile(t *testing.T) {
	clearScryEnv(t)

	// Use an empty directory as CWD so no scry.yml is found.
	dir := t.TempDir()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore cwd: %v", err)
		}
	})

	cfg, err := LoadWithFile("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use defaults.
	if cfg.MaxDepth != 5 {
		t.Errorf("MaxDepth = %d, want 5 (default)", cfg.MaxDepth)
	}
}

func TestMalformedYAML(t *testing.T) {
	clearScryEnv(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "scry.yml")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	_, err := LoadWithFile(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
	if !strings.Contains(err.Error(), "parse config file") {
		t.Errorf("error = %q, want mention of parse config file", err.Error())
	}
}

func TestFullYAML(t *testing.T) {
	clearScryEnv(t)

	dir := t.TempDir()
	yamlContent := `
crawl:
  max_depth: 10
  max_pages: 100
  concurrency: 2
  respect_robots: false
  exclude:
    - "/admin/*"
    - "/api/*"
  include:
    - "/blog/*"
  rate_limit: 5
  timeout: "15s"
  user_agent: "test-agent/2.0"

output:
  formats:
    - json
    - csv
  file: ./report
  fail_on: warning

lighthouse:
  enabled: true
  mode: browserless
  strategy: desktop

browser:
  enabled: true
  browserless_url: http://example.com:9222
`
	path := filepath.Join(dir, "scry.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := LoadWithFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MaxDepth != 10 {
		t.Errorf("MaxDepth = %d, want 10", cfg.MaxDepth)
	}
	if cfg.MaxPages != 100 {
		t.Errorf("MaxPages = %d, want 100", cfg.MaxPages)
	}
	if cfg.Concurrency != 2 {
		t.Errorf("Concurrency = %d, want 2", cfg.Concurrency)
	}
	if cfg.RespectRobots {
		t.Error("RespectRobots = true, want false")
	}
	if cfg.RateLimit != 5 {
		t.Errorf("RateLimit = %d, want 5", cfg.RateLimit)
	}
	if cfg.RequestTimeout != 15*time.Second {
		t.Errorf("RequestTimeout = %v, want 15s", cfg.RequestTimeout)
	}
	if cfg.UserAgent != "test-agent/2.0" {
		t.Errorf("UserAgent = %q, want %q", cfg.UserAgent, "test-agent/2.0")
	}

	wantExclude := []string{"/admin/*", "/api/*"}
	if len(cfg.ExcludePatterns) != len(wantExclude) {
		t.Fatalf("ExcludePatterns = %v, want %v", cfg.ExcludePatterns, wantExclude)
	}
	for i, p := range cfg.ExcludePatterns {
		if p != wantExclude[i] {
			t.Errorf("ExcludePatterns[%d] = %q, want %q", i, p, wantExclude[i])
		}
	}

	wantInclude := []string{"/blog/*"}
	if len(cfg.IncludePatterns) != len(wantInclude) {
		t.Fatalf("IncludePatterns = %v, want %v", cfg.IncludePatterns, wantInclude)
	}
	if cfg.IncludePatterns[0] != "/blog/*" {
		t.Errorf("IncludePatterns[0] = %q, want %q", cfg.IncludePatterns[0], "/blog/*")
	}

	if cfg.OutputFormat != "json,csv" {
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, "json,csv")
	}
	if cfg.OutputFile != "./report" {
		t.Errorf("OutputFile = %q, want %q", cfg.OutputFile, "./report")
	}
	if cfg.FailOn != "warning" {
		t.Errorf("FailOn = %q, want %q", cfg.FailOn, "warning")
	}

	if !cfg.LighthouseEnabled {
		t.Error("LighthouseEnabled = false, want true")
	}
	if cfg.LighthouseMode != "browserless" {
		t.Errorf("LighthouseMode = %q, want %q", cfg.LighthouseMode, "browserless")
	}
	if cfg.PSIStrategy != "desktop" {
		t.Errorf("PSIStrategy = %q, want %q", cfg.PSIStrategy, "desktop")
	}

	if !cfg.BrowserMode {
		t.Error("BrowserMode = false, want true")
	}
	if cfg.BrowserlessURL != "http://example.com:9222" {
		t.Errorf("BrowserlessURL = %q, want %q", cfg.BrowserlessURL, "http://example.com:9222")
	}
}

func TestOutputFormatsMerge(t *testing.T) {
	clearScryEnv(t)

	dir := t.TempDir()
	yamlContent := `
output:
  formats:
    - terminal
    - json
    - markdown
`
	path := filepath.Join(dir, "scry.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := LoadWithFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.OutputFormat != "terminal,json,markdown" {
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, "terminal,json,markdown")
	}

	formats := cfg.OutputFormats()
	want := []string{"terminal", "json", "markdown"}
	if len(formats) != len(want) {
		t.Fatalf("OutputFormats() = %v, want %v", formats, want)
	}
	for i, f := range formats {
		if f != want[i] {
			t.Errorf("OutputFormats()[%d] = %q, want %q", i, f, want[i])
		}
	}
}

func TestDurationParsing(t *testing.T) {
	clearScryEnv(t)

	dir := t.TempDir()
	yamlContent := `
crawl:
  timeout: "30s"
`
	path := filepath.Join(dir, "scry.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := LoadWithFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("RequestTimeout = %v, want 30s", cfg.RequestTimeout)
	}
}

func TestInvalidDuration(t *testing.T) {
	clearScryEnv(t)

	dir := t.TempDir()
	yamlContent := `
crawl:
  timeout: "not-a-duration"
`
	path := filepath.Join(dir, "scry.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	_, err := LoadWithFile(path)
	if err == nil {
		t.Fatal("expected error for invalid duration, got nil")
	}
	if !strings.Contains(err.Error(), "parse crawl timeout") {
		t.Errorf("error = %q, want mention of parse crawl timeout", err.Error())
	}
}

func TestFindConfigFileCWD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, configFileName)
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore cwd: %v", err)
		}
	})

	found := findConfigFile("")
	if found != path {
		t.Errorf("findConfigFile() = %q, want %q", found, path)
	}
}

func TestFindConfigFileExplicitPath(t *testing.T) {
	found := findConfigFile("/some/explicit/path.yml")
	if found != "/some/explicit/path.yml" {
		t.Errorf("findConfigFile() = %q, want %q", found, "/some/explicit/path.yml")
	}
}

func TestFindConfigFileNotFound(t *testing.T) {
	dir := t.TempDir()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore cwd: %v", err)
		}
	})

	found := findConfigFile("")
	// No scry.yml in temp dir or home, but home might have one.
	// We just verify it doesn't crash. If it returns empty, that's expected.
	if found != "" {
		// Only fail if the found file doesn't actually exist.
		if _, err := os.Stat(found); err != nil {
			t.Errorf("findConfigFile() = %q, but file does not exist", found)
		}
	}
}

func TestLoadYAMLDirectly(t *testing.T) {
	dir := t.TempDir()
	yamlContent := `
crawl:
  max_depth: 7
`
	path := filepath.Join(dir, "test.yml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	ycfg, err := loadYAML(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ycfg.Crawl.MaxDepth == nil {
		t.Fatal("MaxDepth is nil, want non-nil")
	}
	if *ycfg.Crawl.MaxDepth != 7 {
		t.Errorf("MaxDepth = %d, want 7", *ycfg.Crawl.MaxDepth)
	}

	// Unset fields should remain nil.
	if ycfg.Crawl.MaxPages != nil {
		t.Errorf("MaxPages = %v, want nil", ycfg.Crawl.MaxPages)
	}
}
