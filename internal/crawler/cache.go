package crawler

import (
	"encoding/json"
	"fmt"
	"os"
)

// CrawlCacheEntry stores metadata for a previously crawled URL.
type CrawlCacheEntry struct {
	LastModified string `json:"last_modified,omitempty"`
	ETag         string `json:"etag,omitempty"`
	ContentHash  string `json:"content_hash,omitempty"`
}

// CrawlCache maps URLs to their cached metadata for incremental crawling.
type CrawlCache struct {
	Entries map[string]CrawlCacheEntry `json:"entries"`
}

// NewCrawlCache creates an empty CrawlCache.
func NewCrawlCache() *CrawlCache {
	return &CrawlCache{Entries: make(map[string]CrawlCacheEntry)}
}

// LoadCrawlCache reads a cache file from disk.
func LoadCrawlCache(path string) (*CrawlCache, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewCrawlCache(), nil
		}
		return nil, fmt.Errorf("read crawl cache: %w", err)
	}

	var cc CrawlCache
	if err := json.Unmarshal(data, &cc); err != nil {
		return nil, fmt.Errorf("unmarshal crawl cache: %w", err)
	}

	if cc.Entries == nil {
		cc.Entries = make(map[string]CrawlCacheEntry)
	}

	return &cc, nil
}

// Save writes the cache to disk.
func (cc *CrawlCache) Save(path string) error {
	data, err := json.MarshalIndent(cc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal crawl cache: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write crawl cache: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename crawl cache: %w", err)
	}

	return nil
}

// Get returns the cache entry for a URL, or empty if not cached.
func (cc *CrawlCache) Get(url string) (CrawlCacheEntry, bool) {
	e, ok := cc.Entries[url]
	return e, ok
}

// Set stores a cache entry for a URL.
func (cc *CrawlCache) Set(url string, entry CrawlCacheEntry) {
	cc.Entries[url] = entry
}
