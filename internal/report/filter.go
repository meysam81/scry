package report

import (
	"strings"

	"github.com/meysam81/scry/core/model"
)

// FilterIssues returns only issues matching the given criteria.
// severities: comma-separated list of severities (e.g. "critical,warning")
// categories: comma-separated list of category prefixes (e.g. "seo,performance")
// If a filter is empty string, all issues pass that filter.
func FilterIssues(issues []model.Issue, severities, categories string) []model.Issue {
	sevSet := parseCSV(severities)
	catSet := parseCSV(categories)

	filtered := make([]model.Issue, 0, len(issues))
	for _, issue := range issues {
		if !matchSeverity(issue, sevSet) {
			continue
		}
		if !matchCategory(issue, catSet) {
			continue
		}
		filtered = append(filtered, issue)
	}
	return filtered
}

// GroupIssues groups issues by the given key.
// key: "severity", "category", or "url"
// Returns a map[string][]model.Issue.
func GroupIssues(issues []model.Issue, key string) map[string][]model.Issue {
	groups := make(map[string][]model.Issue)
	for _, issue := range issues {
		k := groupKey(issue, key)
		groups[k] = append(groups[k], issue)
	}
	return groups
}

// categoryOf extracts the category prefix from a check name.
// e.g. "seo/title-missing" -> "seo"
func categoryOf(checkName string) string {
	parts := strings.SplitN(checkName, "/", 2)
	return strings.ToLower(parts[0])
}

// parseCSV splits a comma-separated string into a set of lowercase, trimmed values.
// Returns nil for an empty input.
func parseCSV(s string) map[string]bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	set := make(map[string]bool, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(strings.ToLower(p))
		if trimmed != "" {
			set[trimmed] = true
		}
	}
	return set
}

// matchSeverity returns true if the issue's severity is in the set,
// or if the set is nil (no filter).
func matchSeverity(issue model.Issue, sevSet map[string]bool) bool {
	if sevSet == nil {
		return true
	}
	return sevSet[string(issue.Severity)]
}

// matchCategory returns true if the issue's category is in the set,
// or if the set is nil (no filter).
func matchCategory(issue model.Issue, catSet map[string]bool) bool {
	if catSet == nil {
		return true
	}
	return catSet[categoryOf(issue.CheckName)]
}

// groupKey extracts the grouping key from an issue for the given key type.
func groupKey(issue model.Issue, key string) string {
	switch key {
	case "severity":
		return string(issue.Severity)
	case "category":
		return categoryOf(issue.CheckName)
	case "url":
		return issue.URL
	default:
		return string(issue.Severity)
	}
}
