package analysis

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestAnalyzeRedirects_Empty(t *testing.T) {
	rm := AnalyzeRedirects(nil)
	if len(rm.Entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(rm.Entries))
	}
}

func TestAnalyzeRedirects_NoRedirects(t *testing.T) {
	pages := []*model.Page{
		{URL: "https://example.com/", StatusCode: 200},
		{URL: "https://example.com/about", StatusCode: 200},
	}
	rm := AnalyzeRedirects(pages)
	if len(rm.Entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(rm.Entries))
	}
}

func TestAnalyzeRedirects_SingleHop(t *testing.T) {
	pages := []*model.Page{
		{
			URL:           "https://example.com/new",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/old"},
		},
		{
			URL:        "https://example.com/linker",
			StatusCode: 200,
			Links:      []string{"https://example.com/old"},
		},
	}

	rm := AnalyzeRedirects(pages)
	if len(rm.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(rm.Entries))
	}

	e := rm.Entries[0]
	if e.From != "https://example.com/old" {
		t.Errorf("expected From https://example.com/old, got %s", e.From)
	}
	if e.To != "https://example.com/new" {
		t.Errorf("expected To https://example.com/new, got %s", e.To)
	}
	if e.Hops != 1 {
		t.Errorf("expected 1 hop, got %d", e.Hops)
	}
	if len(e.Chain) != 2 {
		t.Errorf("expected chain length 2, got %d", len(e.Chain))
	}
	if len(e.LinkedBy) != 1 || e.LinkedBy[0] != "https://example.com/linker" {
		t.Errorf("expected LinkedBy [https://example.com/linker], got %v", e.LinkedBy)
	}
}

func TestAnalyzeRedirects_MultiHop(t *testing.T) {
	pages := []*model.Page{
		{
			URL:           "https://example.com/final",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"},
		},
	}

	rm := AnalyzeRedirects(pages)
	if len(rm.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(rm.Entries))
	}

	e := rm.Entries[0]
	if e.Hops != 3 {
		t.Errorf("expected 3 hops, got %d", e.Hops)
	}
	if len(e.Chain) != 4 {
		t.Errorf("expected chain length 4, got %d: %v", len(e.Chain), e.Chain)
	}
	if e.Chain[0] != "https://example.com/a" || e.Chain[3] != "https://example.com/final" {
		t.Errorf("unexpected chain: %v", e.Chain)
	}
}

func TestAnalyzeRedirects_SortedByHopsDescending(t *testing.T) {
	pages := []*model.Page{
		{
			URL:           "https://example.com/dest1",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/a"},
		},
		{
			URL:           "https://example.com/dest2",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/x", "https://example.com/y", "https://example.com/z"},
		},
	}

	rm := AnalyzeRedirects(pages)
	if len(rm.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(rm.Entries))
	}

	if rm.Entries[0].Hops < rm.Entries[1].Hops {
		t.Errorf("entries not sorted by hop count descending: %d < %d", rm.Entries[0].Hops, rm.Entries[1].Hops)
	}
}

func TestAnalyzeRedirects_LinkedByMultipleSources(t *testing.T) {
	pages := []*model.Page{
		{
			URL:           "https://example.com/new",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/old"},
		},
		{
			URL:        "https://example.com/page1",
			StatusCode: 200,
			Links:      []string{"https://example.com/old"},
		},
		{
			URL:        "https://example.com/page2",
			StatusCode: 200,
			Links:      []string{"https://example.com/old", "https://example.com/other"},
		},
	}

	rm := AnalyzeRedirects(pages)
	if len(rm.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(rm.Entries))
	}

	if len(rm.Entries[0].LinkedBy) != 2 {
		t.Errorf("expected 2 linked-by sources, got %d: %v", len(rm.Entries[0].LinkedBy), rm.Entries[0].LinkedBy)
	}
}

func TestRedirectMap_ExportCSV(t *testing.T) {
	rm := &RedirectMap{
		Entries: []RedirectEntry{
			{
				From:     "https://example.com/old",
				To:       "https://example.com/new",
				Chain:    []string{"https://example.com/old", "https://example.com/new"},
				Hops:     1,
				LinkedBy: []string{"https://example.com/page1"},
			},
		},
	}

	var buf bytes.Buffer
	if err := rm.ExportCSV(&buf); err != nil {
		t.Fatalf("ExportCSV error: %v", err)
	}

	csv := buf.String()
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines (header + 1 row), got %d: %q", len(lines), csv)
	}

	if !strings.Contains(lines[0], "from") || !strings.Contains(lines[0], "linked_by") {
		t.Errorf("header missing expected columns: %s", lines[0])
	}

	if !strings.Contains(lines[1], "https://example.com/old") {
		t.Errorf("row missing expected URL: %s", lines[1])
	}
}

func TestRedirectMap_ExportCSV_Empty(t *testing.T) {
	rm := &RedirectMap{}
	var buf bytes.Buffer
	if err := rm.ExportCSV(&buf); err != nil {
		t.Fatalf("ExportCSV error: %v", err)
	}

	csv := buf.String()
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line (header only), got %d", len(lines))
	}
}

// --- Issue checks ---

func TestRedirectChecker_Name(t *testing.T) {
	c := NewRedirectChecker()
	if c.Name() != "redirects" {
		t.Errorf("expected name 'redirects', got %q", c.Name())
	}
}

func TestRedirectChecker_CheckReturnsNil(t *testing.T) {
	c := NewRedirectChecker()
	issues := c.Check(context.Background(), &model.Page{URL: "https://example.com"})
	if issues != nil {
		t.Errorf("expected nil from per-page Check, got %v", issues)
	}
}

func TestCheckRedirectIssues_LongChain(t *testing.T) {
	pages := []*model.Page{
		{
			URL:           "https://example.com/final",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/a", "https://example.com/b", "https://example.com/c"},
		},
	}

	c := NewRedirectChecker()
	issues := c.CheckSite(context.Background(), pages)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "redirects/long-chain" {
			found = true
			if iss.Severity != model.SeverityWarning {
				t.Errorf("expected warning severity, got %s", iss.Severity)
			}
			if !strings.Contains(iss.Message, "3 hops") {
				t.Errorf("expected message to contain '3 hops', got %q", iss.Message)
			}
		}
	}
	if !found {
		t.Error("expected redirects/long-chain issue not found")
	}
}

func TestCheckRedirectIssues_NoLongChainAt2Hops(t *testing.T) {
	pages := []*model.Page{
		{
			URL:           "https://example.com/final",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/a", "https://example.com/b"},
		},
	}

	c := NewRedirectChecker()
	issues := c.CheckSite(context.Background(), pages)

	for _, iss := range issues {
		if iss.CheckName == "redirects/long-chain" {
			t.Errorf("did not expect long-chain issue for 2 hops, got %+v", iss)
		}
	}
}

func TestCheckRedirectIssues_InternalRedirect(t *testing.T) {
	pages := []*model.Page{
		{
			URL:           "https://example.com/new",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/old"},
		},
		{
			URL:        "https://example.com/page",
			StatusCode: 200,
			Links:      []string{"https://example.com/old"},
		},
	}

	c := NewRedirectChecker()
	issues := c.CheckSite(context.Background(), pages)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "redirects/internal-redirect" {
			found = true
			if iss.Severity != model.SeverityInfo {
				t.Errorf("expected info severity, got %s", iss.Severity)
			}
			if iss.URL != "https://example.com/page" {
				t.Errorf("expected issue URL to be the linking page, got %s", iss.URL)
			}
			if !strings.Contains(iss.Message, "https://example.com/old") {
				t.Errorf("expected message to mention redirecting URL, got %q", iss.Message)
			}
		}
	}
	if !found {
		t.Error("expected redirects/internal-redirect issue not found")
	}
}

func TestCheckRedirectIssues_RedirectToRedirect(t *testing.T) {
	// Page /c redirected from /b, and /b itself was the final URL of a
	// redirect from /a. So /a -> /b is a redirect, and /b -> /c is also
	// a redirect, making /a's target (/b) itself a redirect source.
	pages := []*model.Page{
		{
			URL:           "https://example.com/b",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/a"},
		},
		{
			URL:           "https://example.com/c",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/b"},
		},
	}

	c := NewRedirectChecker()
	issues := c.CheckSite(context.Background(), pages)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "redirects/redirect-to-redirect" {
			found = true
			if iss.Severity != model.SeverityWarning {
				t.Errorf("expected warning severity, got %s", iss.Severity)
			}
			if iss.URL != "https://example.com/a" {
				t.Errorf("expected issue URL to be https://example.com/a, got %s", iss.URL)
			}
		}
	}
	if !found {
		t.Error("expected redirects/redirect-to-redirect issue not found")
	}
}

func TestCheckRedirectIssues_NoIssuesForCleanRedirect(t *testing.T) {
	// Single-hop redirect with no pages linking to it — no issues expected.
	pages := []*model.Page{
		{
			URL:           "https://example.com/new",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/old"},
		},
	}

	c := NewRedirectChecker()
	issues := c.CheckSite(context.Background(), pages)

	if len(issues) != 0 {
		t.Errorf("expected no issues, got %d: %+v", len(issues), issues)
	}
}

func TestCheckRedirectIssues_CombinedScenario(t *testing.T) {
	// /a -> /b -> /c (long chain), page /home links to /a (internal redirect),
	// /b is itself a redirect target (redirect-to-redirect from /a's perspective).
	pages := []*model.Page{
		{
			URL:           "https://example.com/c",
			StatusCode:    200,
			RedirectChain: []string{"https://example.com/a", "https://example.com/b"},
		},
		{
			URL:        "https://example.com/home",
			StatusCode: 200,
			Links:      []string{"https://example.com/a"},
		},
	}

	c := NewRedirectChecker()
	issues := c.CheckSite(context.Background(), pages)

	checks := make(map[string]bool)
	for _, iss := range issues {
		checks[iss.CheckName] = true
	}

	// 2 hops, which is not > 2, so no long-chain.
	if checks["redirects/long-chain"] {
		t.Error("did not expect long-chain for 2-hop redirect")
	}
	// /home links to /a which redirects.
	if !checks["redirects/internal-redirect"] {
		t.Error("expected internal-redirect issue")
	}
}
