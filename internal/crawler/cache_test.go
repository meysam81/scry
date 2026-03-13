package crawler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewCrawlCache(t *testing.T) {
	cc := NewCrawlCache()
	if cc.Entries == nil {
		t.Fatal("expected non-nil Entries")
	}
	if len(cc.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(cc.Entries))
	}
}

func TestCrawlCache_GetSet(t *testing.T) {
	cc := NewCrawlCache()

	_, ok := cc.Get("https://example.com")
	if ok {
		t.Error("expected false for missing URL")
	}

	cc.Set("https://example.com", CrawlCacheEntry{ETag: "abc123"})

	e, ok := cc.Get("https://example.com")
	if !ok {
		t.Fatal("expected true for existing URL")
	}
	if e.ETag != "abc123" {
		t.Errorf("ETag = %q, want %q", e.ETag, "abc123")
	}
}

func TestCrawlCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	cc := NewCrawlCache()
	cc.Set("https://example.com", CrawlCacheEntry{
		LastModified: "Mon, 01 Jan 2024 00:00:00 GMT",
		ETag:         "etag-1",
		ContentHash:  "sha256-abc",
	})

	if err := cc.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadCrawlCache(path)
	if err != nil {
		t.Fatalf("LoadCrawlCache failed: %v", err)
	}

	e, ok := loaded.Get("https://example.com")
	if !ok {
		t.Fatal("expected entry after reload")
	}
	if e.ETag != "etag-1" {
		t.Errorf("ETag = %q, want %q", e.ETag, "etag-1")
	}
}

func TestLoadCrawlCache_NotExist(t *testing.T) {
	cc, err := LoadCrawlCache("/tmp/nonexistent-cache-12345.json")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if len(cc.Entries) != 0 {
		t.Errorf("expected empty cache, got %d entries", len(cc.Entries))
	}
}

func TestLoadCrawlCache_Invalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadCrawlCache(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
