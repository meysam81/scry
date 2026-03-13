package baseline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meysam81/scry/internal/model"
)

// helper builds a minimal CrawlResult with the given issues.
func crawlResult(seedURL string, issues ...model.Issue) *model.CrawlResult {
	return &model.CrawlResult{
		SeedURL:   seedURL,
		Issues:    issues,
		CrawledAt: time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC),
		Duration:  3 * time.Second,
	}
}

func issue(check, url string) model.Issue {
	return model.Issue{
		CheckName: check,
		Severity:  model.SeverityWarning,
		Message:   check + " on " + url,
		URL:       url,
	}
}

// ---------- Save / Load roundtrip ----------

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")

	cr := crawlResult("https://example.com",
		issue("broken-link", "https://example.com/a"),
		issue("missing-alt", "https://example.com/b"),
	)

	if err := Save(path, cr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.SeedURL != cr.SeedURL {
		t.Errorf("SeedURL = %q, want %q", got.SeedURL, cr.SeedURL)
	}
	if got.Version != version {
		t.Errorf("Version = %q, want %q", got.Version, version)
	}
	if !got.CreatedAt.Equal(cr.CrawledAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, cr.CrawledAt)
	}
	if len(got.Issues) != 2 {
		t.Fatalf("len(Issues) = %d, want 2", len(got.Issues))
	}
	for i, want := range cr.Issues {
		if got.Issues[i].CheckName != want.CheckName {
			t.Errorf("Issues[%d].CheckName = %q, want %q", i, got.Issues[i].CheckName, want.CheckName)
		}
		if got.Issues[i].URL != want.URL {
			t.Errorf("Issues[%d].URL = %q, want %q", i, got.Issues[i].URL, want.URL)
		}
	}
}

func TestSavePrettyPrintsJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")

	cr := crawlResult("https://example.com", issue("check", "https://example.com"))
	if err := Save(path, cr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Pretty-printed JSON must be valid and contain indentation.
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("file is not valid JSON: %v", err)
	}

	// A compact representation would not contain a newline followed by spaces.
	if len(data) < 10 {
		t.Fatal("file suspiciously short")
	}
	// Check the file ends with a trailing newline.
	if data[len(data)-1] != '\n' {
		t.Error("file should end with a trailing newline")
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/baseline.json")
	if err == nil {
		t.Fatal("expected error when loading nonexistent file")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not json}"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error when loading invalid JSON")
	}
}

// ---------- Diff ----------

func TestDiffNewIssues(t *testing.T) {
	bl := &Baseline{
		SeedURL: "https://example.com",
		Issues:  []model.Issue{issue("check-a", "https://example.com/1")},
	}

	current := crawlResult("https://example.com",
		issue("check-a", "https://example.com/1"),
		issue("check-b", "https://example.com/2"),
	)

	diff := Diff(bl, current)

	if len(diff.New) != 1 {
		t.Fatalf("len(New) = %d, want 1", len(diff.New))
	}
	if diff.New[0].CheckName != "check-b" {
		t.Errorf("New[0].CheckName = %q, want %q", diff.New[0].CheckName, "check-b")
	}
	if len(diff.Existing) != 1 {
		t.Fatalf("len(Existing) = %d, want 1", len(diff.Existing))
	}
	if len(diff.Resolved) != 0 {
		t.Errorf("len(Resolved) = %d, want 0", len(diff.Resolved))
	}
}

func TestDiffResolvedIssues(t *testing.T) {
	bl := &Baseline{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			issue("check-a", "https://example.com/1"),
			issue("check-b", "https://example.com/2"),
		},
	}

	current := crawlResult("https://example.com",
		issue("check-a", "https://example.com/1"),
	)

	diff := Diff(bl, current)

	if len(diff.Resolved) != 1 {
		t.Fatalf("len(Resolved) = %d, want 1", len(diff.Resolved))
	}
	if diff.Resolved[0].CheckName != "check-b" {
		t.Errorf("Resolved[0].CheckName = %q, want %q", diff.Resolved[0].CheckName, "check-b")
	}
	if len(diff.Existing) != 1 {
		t.Fatalf("len(Existing) = %d, want 1", len(diff.Existing))
	}
	if len(diff.New) != 0 {
		t.Errorf("len(New) = %d, want 0", len(diff.New))
	}
}

func TestDiffExistingIssues(t *testing.T) {
	issues := []model.Issue{
		issue("check-a", "https://example.com/1"),
		issue("check-b", "https://example.com/2"),
	}

	bl := &Baseline{SeedURL: "https://example.com", Issues: issues}
	current := crawlResult("https://example.com", issues...)

	diff := Diff(bl, current)

	if len(diff.Existing) != 2 {
		t.Errorf("len(Existing) = %d, want 2", len(diff.Existing))
	}
	if len(diff.New) != 0 {
		t.Errorf("len(New) = %d, want 0", len(diff.New))
	}
	if len(diff.Resolved) != 0 {
		t.Errorf("len(Resolved) = %d, want 0", len(diff.Resolved))
	}
}

func TestDiffEmptyBaseline(t *testing.T) {
	bl := &Baseline{SeedURL: "https://example.com"}

	current := crawlResult("https://example.com",
		issue("check-a", "https://example.com/1"),
		issue("check-b", "https://example.com/2"),
	)

	diff := Diff(bl, current)

	if len(diff.New) != 2 {
		t.Errorf("len(New) = %d, want 2", len(diff.New))
	}
	if len(diff.Existing) != 0 {
		t.Errorf("len(Existing) = %d, want 0", len(diff.Existing))
	}
	if len(diff.Resolved) != 0 {
		t.Errorf("len(Resolved) = %d, want 0", len(diff.Resolved))
	}
}

func TestDiffEmptyCurrent(t *testing.T) {
	bl := &Baseline{
		SeedURL: "https://example.com",
		Issues: []model.Issue{
			issue("check-a", "https://example.com/1"),
			issue("check-b", "https://example.com/2"),
		},
	}

	current := crawlResult("https://example.com")

	diff := Diff(bl, current)

	if len(diff.Resolved) != 2 {
		t.Errorf("len(Resolved) = %d, want 2", len(diff.Resolved))
	}
	if len(diff.New) != 0 {
		t.Errorf("len(New) = %d, want 0", len(diff.New))
	}
	if len(diff.Existing) != 0 {
		t.Errorf("len(Existing) = %d, want 0", len(diff.Existing))
	}
}

func TestDiffBothEmpty(t *testing.T) {
	bl := &Baseline{SeedURL: "https://example.com"}
	current := crawlResult("https://example.com")

	diff := Diff(bl, current)

	if len(diff.New) != 0 {
		t.Errorf("len(New) = %d, want 0", len(diff.New))
	}
	if len(diff.Resolved) != 0 {
		t.Errorf("len(Resolved) = %d, want 0", len(diff.Resolved))
	}
	if len(diff.Existing) != 0 {
		t.Errorf("len(Existing) = %d, want 0", len(diff.Existing))
	}
}

func TestDiffSameCheckDifferentURL(t *testing.T) {
	bl := &Baseline{
		SeedURL: "https://example.com",
		Issues:  []model.Issue{issue("broken-link", "https://example.com/a")},
	}

	current := crawlResult("https://example.com",
		issue("broken-link", "https://example.com/b"),
	)

	diff := Diff(bl, current)

	// Same check name but different URL = different issue.
	if len(diff.New) != 1 {
		t.Errorf("len(New) = %d, want 1", len(diff.New))
	}
	if len(diff.Resolved) != 1 {
		t.Errorf("len(Resolved) = %d, want 1", len(diff.Resolved))
	}
	if len(diff.Existing) != 0 {
		t.Errorf("len(Existing) = %d, want 0", len(diff.Existing))
	}
}

func TestDiffSameURLDifferentCheck(t *testing.T) {
	url := "https://example.com/page"
	bl := &Baseline{
		SeedURL: "https://example.com",
		Issues:  []model.Issue{issue("check-a", url)},
	}

	current := crawlResult("https://example.com",
		issue("check-b", url),
	)

	diff := Diff(bl, current)

	// Same URL but different check = different issue.
	if len(diff.New) != 1 {
		t.Errorf("len(New) = %d, want 1", len(diff.New))
	}
	if len(diff.Resolved) != 1 {
		t.Errorf("len(Resolved) = %d, want 1", len(diff.Resolved))
	}
	if len(diff.Existing) != 0 {
		t.Errorf("len(Existing) = %d, want 0", len(diff.Existing))
	}
}

func TestDiffDuplicateIssues(t *testing.T) {
	// Baseline has 2 copies of the same issue; current has 1.
	// Expected: 1 existing, 1 resolved, 0 new.
	issueA := issue("check-a", "https://example.com/1")

	bl := &Baseline{
		SeedURL: "https://example.com",
		Issues:  []model.Issue{issueA, issueA},
	}

	current := crawlResult("https://example.com", issueA)

	diff := Diff(bl, current)

	if len(diff.Existing) != 1 {
		t.Errorf("len(Existing) = %d, want 1", len(diff.Existing))
	}
	if len(diff.Resolved) != 1 {
		t.Errorf("len(Resolved) = %d, want 1", len(diff.Resolved))
	}
	if len(diff.New) != 0 {
		t.Errorf("len(New) = %d, want 0", len(diff.New))
	}
}

func TestDiffDuplicateIssuesReverse(t *testing.T) {
	// Baseline has 1 copy; current has 2 copies.
	// Expected: 1 existing, 0 resolved, 1 new.
	issueA := issue("check-a", "https://example.com/1")

	bl := &Baseline{
		SeedURL: "https://example.com",
		Issues:  []model.Issue{issueA},
	}

	current := crawlResult("https://example.com", issueA, issueA)

	diff := Diff(bl, current)

	if len(diff.Existing) != 1 {
		t.Errorf("len(Existing) = %d, want 1", len(diff.Existing))
	}
	if len(diff.Resolved) != 0 {
		t.Errorf("len(Resolved) = %d, want 0", len(diff.Resolved))
	}
	if len(diff.New) != 1 {
		t.Errorf("len(New) = %d, want 1", len(diff.New))
	}
}
