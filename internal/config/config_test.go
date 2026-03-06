package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// clearScryEnv unsets all SCRY_ environment variables via t.Setenv so they
// are automatically restored after the test.
func clearScryEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(key, "SCRY_") {
			t.Setenv(key, "")
			_ = os.Unsetenv(key)
		}
	}
}

func TestLoadDefaults(t *testing.T) {
	clearScryEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.MaxDepth != 5 {
		t.Errorf("MaxDepth = %d, want 5", cfg.MaxDepth)
	}
	if cfg.MaxPages != 500 {
		t.Errorf("MaxPages = %d, want 500", cfg.MaxPages)
	}
	if cfg.Concurrency != 10 {
		t.Errorf("Concurrency = %d, want 10", cfg.Concurrency)
	}
	if cfg.RequestTimeout != 10*time.Second {
		t.Errorf("RequestTimeout = %v, want 10s", cfg.RequestTimeout)
	}
	if cfg.RateLimit != 50 {
		t.Errorf("RateLimit = %d, want 50", cfg.RateLimit)
	}
	if cfg.UserAgent != "scry/1.0" {
		t.Errorf("UserAgent = %q, want %q", cfg.UserAgent, "scry/1.0")
	}
	if !cfg.RespectRobots {
		t.Error("RespectRobots = false, want true")
	}
	if cfg.OutputFormat != "terminal" {
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, "terminal")
	}
	if cfg.OutputFile != "" {
		t.Errorf("OutputFile = %q, want empty", cfg.OutputFile)
	}
	if cfg.FailOn != "" {
		t.Errorf("FailOn = %q, want empty", cfg.FailOn)
	}
	if cfg.BrowserMode {
		t.Error("BrowserMode = true, want false")
	}
	if cfg.BrowserlessURL != "http://localhost:3000" {
		t.Errorf("BrowserlessURL = %q, want %q", cfg.BrowserlessURL, "http://localhost:3000")
	}
	if cfg.LighthouseEnabled {
		t.Error("LighthouseEnabled = true, want false")
	}
	if cfg.LighthouseMode != "psi" {
		t.Errorf("LighthouseMode = %q, want %q", cfg.LighthouseMode, "psi")
	}
	if cfg.PSIApiKey != "" {
		t.Errorf("PSIApiKey = %q, want empty", cfg.PSIApiKey)
	}
	if cfg.PSIStrategy != "mobile" {
		t.Errorf("PSIStrategy = %q, want %q", cfg.PSIStrategy, "mobile")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.LogFormat != "pretty" {
		t.Errorf("LogFormat = %q, want %q", cfg.LogFormat, "pretty")
	}
}

func TestLoadEnvOverride(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_MAX_DEPTH", "10")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MaxDepth != 10 {
		t.Errorf("MaxDepth = %d, want 10", cfg.MaxDepth)
	}
}

func TestLoadDurationParsing(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_REQUEST_TIMEOUT", "30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("RequestTimeout = %v, want 30s", cfg.RequestTimeout)
	}
}

func TestValidateMaxDepthZero(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_MAX_DEPTH", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for MaxDepth=0, got nil")
	}
	if !strings.Contains(err.Error(), "max depth") {
		t.Errorf("error = %q, want mention of max depth", err.Error())
	}
}

func TestValidateInvalidOutputFormat(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_OUTPUT", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid output format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid output format") {
		t.Errorf("error = %q, want mention of invalid output format", err.Error())
	}
}

func TestValidateInvalidFailOn(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_FAIL_ON", "none")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid fail-on, got nil")
	}
	if !strings.Contains(err.Error(), "invalid fail-on") {
		t.Errorf("error = %q, want mention of invalid fail-on", err.Error())
	}
}

func TestValidateInvalidLogLevel(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_LOG_LEVEL", "trace")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid log level, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log level") {
		t.Errorf("error = %q, want mention of invalid log level", err.Error())
	}
}

func TestValidateInvalidLogFormat(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_LOG_FORMAT", "xml")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid log format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid log format") {
		t.Errorf("error = %q, want mention of invalid log format", err.Error())
	}
}

func TestValidateInvalidLighthouseMode(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_LIGHTHOUSE_MODE", "local")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid lighthouse mode, got nil")
	}
	if !strings.Contains(err.Error(), "invalid lighthouse mode") {
		t.Errorf("error = %q, want mention of invalid lighthouse mode", err.Error())
	}
}

func TestValidateInvalidPSIStrategy(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_PSI_STRATEGY", "tablet")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid psi strategy, got nil")
	}
	if !strings.Contains(err.Error(), "invalid psi strategy") {
		t.Errorf("error = %q, want mention of invalid psi strategy", err.Error())
	}
}

func TestOutputFormatsSplitsComma(t *testing.T) {
	cfg := &Config{OutputFormat: "terminal, json, csv"}
	formats := cfg.OutputFormats()

	want := []string{"terminal", "json", "csv"}
	if len(formats) != len(want) {
		t.Fatalf("OutputFormats() returned %d items, want %d", len(formats), len(want))
	}
	for i, f := range formats {
		if f != want[i] {
			t.Errorf("OutputFormats()[%d] = %q, want %q", i, f, want[i])
		}
	}
}

func TestValidateMultipleOutputFormatsValid(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_OUTPUT", "terminal,json")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	formats := cfg.OutputFormats()
	if len(formats) != 2 {
		t.Fatalf("OutputFormats() = %v, want 2 elements", formats)
	}
	if formats[0] != "terminal" || formats[1] != "json" {
		t.Errorf("OutputFormats() = %v, want [terminal json]", formats)
	}
}

func TestValidateMultipleOutputFormatsInvalid(t *testing.T) {
	clearScryEnv(t)
	t.Setenv("SCRY_OUTPUT", "terminal,invalid")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for terminal,invalid output format, got nil")
	}
	if !strings.Contains(err.Error(), `invalid output format "invalid"`) {
		t.Errorf("error = %q, want mention of invalid output format", err.Error())
	}
}
