package cmdutil

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const maxURLsFromFile = 10000

// ReadURLsFromFile reads one URL per line from a file, trimming whitespace
// and skipping blank lines and lines starting with #.
func ReadURLsFromFile(path string) (_ []string, readErr error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open urls file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && readErr == nil {
			readErr = fmt.Errorf("close urls file: %w", cerr)
		}
	}()

	var urls []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, NormalizeURL(line))
		if len(urls) >= maxURLsFromFile {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read urls file: %w", err)
	}

	return urls, nil
}
