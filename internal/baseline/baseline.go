// Package baseline provides snapshot-based comparison for crawl results.
// A baseline captures the set of issues found during a crawl so that
// subsequent crawls can report only new or resolved issues.
package baseline

import (
	"encoding/json"
	"os"
	"time"

	"github.com/meysam81/scry/internal/model"
)

// version is the schema version written into every baseline file.
const version = "1"

// Baseline is a snapshot of issues from a single crawl session.
type Baseline struct {
	SeedURL   string        `json:"seed_url"`
	Issues    []model.Issue `json:"issues"`
	CreatedAt time.Time     `json:"created_at"`
	Version   string        `json:"version"`
}

// DiffResult categorises issues by comparing a baseline to a current crawl.
type DiffResult struct {
	New      []model.Issue `json:"new"`      // issues in current but not baseline
	Resolved []model.Issue `json:"resolved"` // issues in baseline but not current
	Existing []model.Issue `json:"existing"` // issues in both
}

// issueKey returns the identity key for an issue.
// Two issues are considered the same if they share the same check name and URL.
func issueKey(i model.Issue) string {
	return i.CheckName + "|" + i.URL
}

// Save serialises the issues from a CrawlResult into a pretty-printed JSON
// baseline file at the given path.
func Save(path string, result *model.CrawlResult) error {
	b := Baseline{
		SeedURL:   result.SeedURL,
		Issues:    result.Issues,
		CreatedAt: result.CrawledAt,
		Version:   version,
	}

	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// Load reads a baseline file from disk and returns the parsed Baseline.
func Load(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var b Baseline
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, err
	}

	return &b, nil
}

// Diff compares a baseline snapshot against a current crawl result and
// classifies every issue as new, resolved, or existing.
func Diff(baseline *Baseline, current *model.CrawlResult) *DiffResult {
	result := &DiffResult{}

	// Index baseline issues by key. When multiple issues share the same key
	// (shouldn't normally happen), we track a count so that duplicates are
	// handled correctly.
	baselineIndex := make(map[string]int, len(baseline.Issues))
	for _, issue := range baseline.Issues {
		baselineIndex[issueKey(issue)]++
	}

	// Walk current issues: if the key exists in the baseline, it is existing;
	// otherwise it is new.
	currentIndex := make(map[string]int, len(current.Issues))
	for _, issue := range current.Issues {
		key := issueKey(issue)
		currentIndex[key]++

		if baselineIndex[key] > 0 {
			result.Existing = append(result.Existing, issue)
		} else {
			result.New = append(result.New, issue)
		}
	}

	// Walk baseline issues: anything not present in the current set is resolved.
	for _, issue := range baseline.Issues {
		if currentIndex[issueKey(issue)] == 0 {
			result.Resolved = append(result.Resolved, issue)
		}
	}

	return result
}
