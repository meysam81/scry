package crawler

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
)

// Checkpoint represents a saved crawl state that can be resumed.
type Checkpoint struct {
	SeedURL  string          `json:"seed_url"`
	Seen     map[string]bool `json:"seen"`
	Queue    []FrontierTask  `json:"queue"`
	PageURLs []string        `json:"page_urls"`
}

// SaveCheckpoint writes the current frontier state to a JSON file.
func SaveCheckpoint(path string, seedURL string, frontier *Frontier, pageURLs []string) error {
	frontier.mu.Lock()
	seen := make(map[string]bool, len(frontier.seen))
	maps.Copy(seen, frontier.seen)
	queue := make([]FrontierTask, len(frontier.queue))
	copy(queue, frontier.queue)
	frontier.mu.Unlock()

	cp := Checkpoint{
		SeedURL:  seedURL,
		Seen:     seen,
		Queue:    queue,
		PageURLs: pageURLs,
	}

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write checkpoint: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename checkpoint: %w", err)
	}

	return nil
}

// LoadCheckpoint reads a checkpoint file and returns the saved state.
func LoadCheckpoint(path string) (*Checkpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint: %w", err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("unmarshal checkpoint: %w", err)
	}

	return &cp, nil
}

// RestoreFrontier creates a Frontier pre-populated from a Checkpoint.
func RestoreFrontier(cp *Checkpoint, seedHost string, maxPages int, include, exclude []string) *Frontier {
	f := NewFrontier(seedHost, maxPages, include, exclude)
	f.mu.Lock()
	defer f.mu.Unlock()

	maps.Copy(f.seen, cp.Seen)
	f.queue = append(f.queue, cp.Queue...)

	return f
}

// DeleteCheckpoint removes a checkpoint file.
func DeleteCheckpoint(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete checkpoint: %w", err)
	}
	return nil
}
