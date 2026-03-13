package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadURLsFromFile(t *testing.T) {
	content := "https://example.com\n# comment\nhttps://test.org\n\nhttps://foo.bar\n"
	path := filepath.Join(t.TempDir(), "urls.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	urls, err := ReadURLsFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) != 3 {
		t.Fatalf("expected 3 urls, got %d", len(urls))
	}
}

func TestReadURLsFromFile_NotExist(t *testing.T) {
	_, err := ReadURLsFromFile("/tmp/nonexistent-url-file-12345.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadURLsFromFile_MaxCap(t *testing.T) {
	var lines []string
	for i := 0; i < 10010; i++ {
		lines = append(lines, "https://example.com/page-"+fmt.Sprintf("%d", i))
	}
	path := filepath.Join(t.TempDir(), "many.txt")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	urls, err := ReadURLsFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(urls) > maxURLsFromFile {
		t.Fatalf("expected at most %d urls, got %d", maxURLsFromFile, len(urls))
	}
}
