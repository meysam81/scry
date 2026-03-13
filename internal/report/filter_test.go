package report

import (
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func sampleIssues() []model.Issue {
	return []model.Issue{
		{CheckName: "seo/title-missing", Severity: model.SeverityCritical, URL: "https://a.com/1"},
		{CheckName: "seo/meta-description", Severity: model.SeverityWarning, URL: "https://a.com/2"},
		{CheckName: "performance/slow-ttfb", Severity: model.SeverityWarning, URL: "https://a.com/1"},
		{CheckName: "security/no-https", Severity: model.SeverityCritical, URL: "https://a.com/3"},
		{CheckName: "links/broken", Severity: model.SeverityInfo, URL: "https://a.com/2"},
	}
}

func TestFilterIssues_NoFilters(t *testing.T) {
	issues := sampleIssues()
	got := FilterIssues(issues, "", "")
	if len(got) != len(issues) {
		t.Errorf("FilterIssues with no filters: got %d, want %d", len(got), len(issues))
	}
}

func TestFilterIssues_BySeverity(t *testing.T) {
	issues := sampleIssues()

	got := FilterIssues(issues, "critical", "")
	if len(got) != 2 {
		t.Errorf("FilterIssues severity=critical: got %d, want 2", len(got))
	}
	for _, issue := range got {
		if issue.Severity != model.SeverityCritical {
			t.Errorf("unexpected severity %q in critical filter", issue.Severity)
		}
	}
}

func TestFilterIssues_BySeverityMultiple(t *testing.T) {
	issues := sampleIssues()

	got := FilterIssues(issues, "critical,info", "")
	if len(got) != 3 {
		t.Errorf("FilterIssues severity=critical,info: got %d, want 3", len(got))
	}
}

func TestFilterIssues_ByCategory(t *testing.T) {
	issues := sampleIssues()

	got := FilterIssues(issues, "", "seo")
	if len(got) != 2 {
		t.Errorf("FilterIssues category=seo: got %d, want 2", len(got))
	}
	for _, issue := range got {
		if categoryOf(issue.CheckName) != "seo" {
			t.Errorf("unexpected category %q in seo filter", categoryOf(issue.CheckName))
		}
	}
}

func TestFilterIssues_ByCategoryMultiple(t *testing.T) {
	issues := sampleIssues()

	got := FilterIssues(issues, "", "seo,performance")
	if len(got) != 3 {
		t.Errorf("FilterIssues category=seo,performance: got %d, want 3", len(got))
	}
}

func TestFilterIssues_Combined(t *testing.T) {
	issues := sampleIssues()

	// Only critical issues in seo category
	got := FilterIssues(issues, "critical", "seo")
	if len(got) != 1 {
		t.Errorf("FilterIssues severity=critical,category=seo: got %d, want 1", len(got))
	}
	if got[0].CheckName != "seo/title-missing" {
		t.Errorf("expected seo/title-missing, got %q", got[0].CheckName)
	}
}

func TestFilterIssues_NoMatch(t *testing.T) {
	issues := sampleIssues()

	got := FilterIssues(issues, "critical", "links")
	if len(got) != 0 {
		t.Errorf("FilterIssues with no matches: got %d, want 0", len(got))
	}
}

func TestFilterIssues_EmptyInput(t *testing.T) {
	got := FilterIssues(nil, "critical", "seo")
	if len(got) != 0 {
		t.Errorf("FilterIssues with nil input: got %d, want 0", len(got))
	}

	got = FilterIssues([]model.Issue{}, "critical", "seo")
	if len(got) != 0 {
		t.Errorf("FilterIssues with empty input: got %d, want 0", len(got))
	}
}

func TestFilterIssues_CaseInsensitive(t *testing.T) {
	issues := sampleIssues()

	got := FilterIssues(issues, "Critical", "SEO")
	if len(got) != 1 {
		t.Errorf("FilterIssues case-insensitive: got %d, want 1", len(got))
	}
}

func TestFilterIssues_WhitespaceInFilter(t *testing.T) {
	issues := sampleIssues()

	got := FilterIssues(issues, " critical , warning ", "")
	if len(got) != 4 {
		t.Errorf("FilterIssues with whitespace: got %d, want 4", len(got))
	}
}

func TestGroupIssues_BySeverity(t *testing.T) {
	issues := sampleIssues()

	groups := GroupIssues(issues, "severity")
	if len(groups) != 3 {
		t.Errorf("GroupIssues by severity: got %d groups, want 3", len(groups))
	}
	if len(groups["critical"]) != 2 {
		t.Errorf("critical group: got %d, want 2", len(groups["critical"]))
	}
	if len(groups["warning"]) != 2 {
		t.Errorf("warning group: got %d, want 2", len(groups["warning"]))
	}
	if len(groups["info"]) != 1 {
		t.Errorf("info group: got %d, want 1", len(groups["info"]))
	}
}

func TestGroupIssues_ByCategory(t *testing.T) {
	issues := sampleIssues()

	groups := GroupIssues(issues, "category")
	if len(groups) != 4 {
		t.Errorf("GroupIssues by category: got %d groups, want 4", len(groups))
	}
	if len(groups["seo"]) != 2 {
		t.Errorf("seo group: got %d, want 2", len(groups["seo"]))
	}
	if len(groups["performance"]) != 1 {
		t.Errorf("performance group: got %d, want 1", len(groups["performance"]))
	}
}

func TestGroupIssues_ByURL(t *testing.T) {
	issues := sampleIssues()

	groups := GroupIssues(issues, "url")
	if len(groups) != 3 {
		t.Errorf("GroupIssues by url: got %d groups, want 3", len(groups))
	}
	if len(groups["https://a.com/1"]) != 2 {
		t.Errorf("url /1 group: got %d, want 2", len(groups["https://a.com/1"]))
	}
}

func TestGroupIssues_Empty(t *testing.T) {
	groups := GroupIssues(nil, "severity")
	if len(groups) != 0 {
		t.Errorf("GroupIssues empty: got %d groups, want 0", len(groups))
	}
}

func TestGroupIssues_UnknownKey(t *testing.T) {
	issues := sampleIssues()

	// Unknown key defaults to severity grouping.
	groups := GroupIssues(issues, "unknown")
	if len(groups) != 3 {
		t.Errorf("GroupIssues unknown key: got %d groups, want 3 (default severity)", len(groups))
	}
}

func TestCategoryOf(t *testing.T) {
	tests := []struct {
		checkName string
		want      string
	}{
		{"seo/title-missing", "seo"},
		{"performance/slow-ttfb", "performance"},
		{"links/broken", "links"},
		{"security/no-https", "security"},
		{"single-segment", "single-segment"},
		{"SEO/Upper", "seo"},
		{"", ""},
	}
	for _, tt := range tests {
		got := categoryOf(tt.checkName)
		if got != tt.want {
			t.Errorf("categoryOf(%q) = %q, want %q", tt.checkName, got, tt.want)
		}
	}
}

func TestParseCSV(t *testing.T) {
	tests := []struct {
		input string
		want  int // number of entries
	}{
		{"", 0},
		{"  ", 0},
		{"a", 1},
		{"a,b", 2},
		{"a, b, c", 3},
		{" A , B ", 2},
		{",,,", 0},
	}
	for _, tt := range tests {
		got := parseCSV(tt.input)
		gotLen := len(got)
		if tt.want == 0 && got != nil && gotLen != 0 {
			t.Errorf("parseCSV(%q) = %v, want nil or empty", tt.input, got)
		}
		if tt.want > 0 && gotLen != tt.want {
			t.Errorf("parseCSV(%q) has %d entries, want %d", tt.input, gotLen, tt.want)
		}
	}
}
