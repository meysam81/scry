package crawler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadCheckpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "checkpoint.json")

	frontier := NewFrontier("example.com", 100, nil, nil)
	frontier.Add("https://example.com/page1", 0)
	frontier.Add("https://example.com/page2", 1)

	pageURLs := []string{"https://example.com/page1"}

	if err := SaveCheckpoint(path, "https://example.com", frontier, pageURLs); err != nil {
		t.Fatalf("SaveCheckpoint failed: %v", err)
	}

	cp, err := LoadCheckpoint(path)
	if err != nil {
		t.Fatalf("LoadCheckpoint failed: %v", err)
	}

	if cp.SeedURL != "https://example.com" {
		t.Errorf("SeedURL = %q, want %q", cp.SeedURL, "https://example.com")
	}
	if len(cp.Seen) != 2 {
		t.Errorf("Seen has %d entries, want 2", len(cp.Seen))
	}
	if len(cp.PageURLs) != 1 {
		t.Errorf("PageURLs has %d entries, want 1", len(cp.PageURLs))
	}
}

func TestRestoreFrontier(t *testing.T) {
	cp := &Checkpoint{
		SeedURL: "https://example.com",
		Seen: map[string]bool{
			"https://example.com/a": true,
			"https://example.com/b": true,
		},
		Queue: []FrontierTask{
			{URL: "https://example.com/c", Depth: 2},
		},
	}

	f := RestoreFrontier(cp, "example.com", 100, nil, nil)

	if f.Seen() != 2 {
		t.Errorf("Seen() = %d, want 2", f.Seen())
	}
	if f.Len() != 1 {
		t.Errorf("Len() = %d, want 1", f.Len())
	}

	task, ok := f.Dequeue()
	if !ok {
		t.Fatal("Dequeue returned false")
	}
	if task.URL != "https://example.com/c" {
		t.Errorf("task.URL = %q, want %q", task.URL, "https://example.com/c")
	}
}

func TestDeleteCheckpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "checkpoint.json")

	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := DeleteCheckpoint(path); err != nil {
		t.Fatalf("DeleteCheckpoint failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestDeleteCheckpoint_NotExist(t *testing.T) {
	if err := DeleteCheckpoint("/tmp/nonexistent-checkpoint-12345.json"); err != nil {
		t.Fatalf("DeleteCheckpoint should not error on missing file: %v", err)
	}
}

func TestLoadCheckpoint_NotExist(t *testing.T) {
	_, err := LoadCheckpoint("/tmp/nonexistent-checkpoint-12345.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
