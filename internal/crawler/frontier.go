package crawler

import (
	"net/url"
	"path/filepath"
	"sync"
)

// FrontierTask represents a URL to be crawled along with its depth from the seed.
type FrontierTask struct {
	URL   string
	Depth int
}

// Frontier is a thread-safe BFS queue with deduplication and scope enforcement.
type Frontier struct {
	seedHost        string
	maxPages        int
	includePatterns []string
	excludePatterns []string

	mu    sync.Mutex
	queue []FrontierTask
	seen  map[string]bool
}

// NewFrontier creates a new Frontier scoped to the given seed host.
func NewFrontier(seedHost string, maxPages int, includePatterns, excludePatterns []string) *Frontier {
	return &Frontier{
		seedHost:        seedHost,
		maxPages:        maxPages,
		includePatterns: includePatterns,
		excludePatterns: excludePatterns,
		seen:            make(map[string]bool),
	}
}

// Add enqueues a URL at the given depth if it is in scope, not a duplicate, and under the cap.
func (f *Frontier) Add(rawURL string, depth int) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.seen[rawURL] {
		return false
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	if parsed.Hostname() != f.seedHost {
		return false
	}

	if f.maxPages > 0 && len(f.seen) >= f.maxPages {
		return false
	}

	if !f.matchesPatterns(parsed.Path) {
		return false
	}

	f.seen[rawURL] = true
	f.queue = append(f.queue, FrontierTask{URL: rawURL, Depth: depth})
	return true
}

// Dequeue removes and returns the next task from the front of the queue.
func (f *Frontier) Dequeue() (FrontierTask, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.queue) == 0 {
		return FrontierTask{}, false
	}

	task := f.queue[0]
	f.queue[0] = FrontierTask{} // zero to release references
	f.queue = f.queue[1:]

	// Compact when the queue is mostly empty to reclaim memory.
	if cap(f.queue) > 64 && len(f.queue) < cap(f.queue)/4 {
		compact := make([]FrontierTask, len(f.queue))
		copy(compact, f.queue)
		f.queue = compact
	}

	return task, true
}

// Len returns the current number of items in the queue.
func (f *Frontier) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.queue)
}

// Seen returns the total number of unique URLs that have been added.
func (f *Frontier) Seen() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.seen)
}

// matchesPatterns checks if the URL path passes include/exclude filters.
// Must be called with f.mu held.
func (f *Frontier) matchesPatterns(path string) bool {
	if len(f.includePatterns) > 0 {
		matched := false
		for _, p := range f.includePatterns {
			if ok, _ := filepath.Match(p, path); ok {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	for _, p := range f.excludePatterns {
		if ok, _ := filepath.Match(p, path); ok {
			return false
		}
	}

	return true
}
