package schema

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/meysam81/scry/internal/logger"
)

var (
	l = logger.Nop()
)

func TestFetchLatest_Success(t *testing.T) {
	validJSON := `{"version":"test-latest","types":{"Article":{"name":"Article","required_fields":["headline"],"properties":{},"google_eligible":true}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(validJSON)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "schemas.json")

	version, err := FetchLatest(l, srv.URL, dest)
	if err != nil {
		t.Fatalf("FetchLatest() error: %v", err)
	}
	if version != "test-latest" {
		t.Errorf("expected version %q, got %q", "test-latest", version)
	}

	// Verify file was written.
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest file: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty file")
	}
}

func TestFetchLatest_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("{not valid json")); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "schemas.json")

	_, err := FetchLatest(l, srv.URL, dest)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	// Verify file was NOT written.
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Error("expected no file to be written on invalid response")
	}
}

func TestFetchLatest_NetworkError(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "schemas.json")

	_, err := FetchLatest(l, "http://localhost:1/nonexistent", dest)
	if err == nil {
		t.Error("expected error for network failure")
	}
}

func TestFetchLatest_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "schemas.json")

	_, err := FetchLatest(l, srv.URL, dest)
	if err == nil {
		t.Error("expected error for HTTP 404")
	}

	// Verify file was NOT written.
	if _, statErr := os.Stat(dest); statErr == nil {
		t.Error("expected no file to be written on HTTP error")
	}
}

func TestFetchLatest_CreatesParentDirs(t *testing.T) {
	validJSON := `{"version":"dirs-test","types":{}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(validJSON)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "nested", "deep", "schemas.json")

	_, err := FetchLatest(l, srv.URL, dest)
	if err != nil {
		t.Fatalf("FetchLatest() error: %v", err)
	}

	if _, statErr := os.Stat(dest); statErr != nil {
		t.Errorf("expected file to exist at %s", dest)
	}
}
