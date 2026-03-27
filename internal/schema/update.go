package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/meysam81/scry/internal/logger"
)

const (
	// DefaultUpdateURL is the default URL for fetching schema updates.
	DefaultUpdateURL = "https://raw.githubusercontent.com/meysam81/scry/main/internal/schema/data/schemas.json"

	fetchTimeout  = 30 * time.Second
	maxSchemaSize = 2 << 20 // 2 MB
)

// fetchClient is an HTTP client with explicit timeout, used instead of
// http.DefaultClient which has no timeout and follows redirects without bound.
var fetchClient = &http.Client{
	Timeout: fetchTimeout,
}

// FetchLatest downloads schema definitions from url, validates them, and
// writes to destPath. Returns the version string on success.
// The local file is not modified on any error.
//
// Note: url is typically the default GitHub raw URL or a user-provided override.
// This function trusts the caller to supply a safe URL.
func FetchLatest(l logger.Logger, url, destPath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := fetchClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch schemas: %w", err)
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			l.Error().Err(err).Msg("failed closing response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch schemas: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSchemaSize))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	// Validate before writing.
	var reg Registry
	if err := json.Unmarshal(body, &reg); err != nil {
		return "", fmt.Errorf("invalid schema data: %w", err)
	}

	// Create parent directories.
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", fmt.Errorf("create directories: %w", err)
	}

	if err := os.WriteFile(destPath, body, 0o644); err != nil {
		return "", fmt.Errorf("write schema file: %w", err)
	}

	return reg.Version, nil
}

// LocalSchemaPath returns the default local schema file path.
// Uses $XDG_DATA_HOME/scry/schemas.json, falling back to ~/.local/share/scry/schemas.json.
func LocalSchemaPath() string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "scry", "schemas.json")
}
