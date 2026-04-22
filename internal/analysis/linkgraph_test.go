package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/meysam81/scry/core/model"
)

func makePages(links map[string][]string) []*model.Page {
	pages := make([]*model.Page, 0, len(links))
	for u, outgoing := range links {
		pages = append(pages, &model.Page{
			URL:         u,
			StatusCode:  200,
			ContentType: "text/html",
			Links:       outgoing,
		})
	}
	return pages
}

func TestBuildGraph_Basic(t *testing.T) {
	pages := makePages(map[string][]string{
		"https://example.com/":      {"https://example.com/about", "https://example.com/blog"},
		"https://example.com/about": {"https://example.com/"},
		"https://example.com/blog":  {"https://example.com/"},
	})

	g := BuildGraph(pages)

	if g.Nodes() != 3 {
		t.Errorf("expected 3 nodes, got %d", g.Nodes())
	}
	if g.Edges() != 4 {
		t.Errorf("expected 4 edges, got %d", g.Edges())
	}
}

func TestBuildGraph_SkipsSelfLinks(t *testing.T) {
	pages := makePages(map[string][]string{
		"https://example.com/": {"https://example.com/"},
	})

	g := BuildGraph(pages)

	if g.Edges() != 0 {
		t.Errorf("expected 0 edges (self-links excluded), got %d", g.Edges())
	}
}

func TestBuildGraph_SkipsExternalLinks(t *testing.T) {
	pages := makePages(map[string][]string{
		"https://example.com/": {"https://external.com/page"},
	})
	// external.com is not in our page set, so it should be excluded.

	g := BuildGraph(pages)

	if g.Edges() != 0 {
		t.Errorf("expected 0 edges (external links excluded), got %d", g.Edges())
	}
}

func TestBuildGraph_NoLinks(t *testing.T) {
	pages := makePages(map[string][]string{
		"https://example.com/":      nil,
		"https://example.com/about": nil,
	})

	g := BuildGraph(pages)

	if g.Nodes() != 2 {
		t.Errorf("expected 2 nodes, got %d", g.Nodes())
	}
	if g.Edges() != 0 {
		t.Errorf("expected 0 edges, got %d", g.Edges())
	}
}

func TestBuildGraph_EmptyPages(t *testing.T) {
	g := BuildGraph(nil)

	if g.Nodes() != 0 {
		t.Errorf("expected 0 nodes, got %d", g.Nodes())
	}
}

func TestComputePageRank_ConvergesToOne(t *testing.T) {
	// All PageRank scores should sum to approximately 1.0.
	pages := makePages(map[string][]string{
		"https://example.com/":  {"https://example.com/a", "https://example.com/b"},
		"https://example.com/a": {"https://example.com/"},
		"https://example.com/b": {"https://example.com/", "https://example.com/a"},
	})

	g := BuildGraph(pages)
	ranks := g.ComputePageRank(20, 0.85)

	sum := 0.0
	for _, r := range ranks {
		sum += r
	}
	if math.Abs(sum-1.0) > 0.01 {
		t.Errorf("PageRank scores sum to %f, expected ~1.0", sum)
	}
}

func TestComputePageRank_HubHasHighestRank(t *testing.T) {
	// The root page receives links from all others, so it should rank highest.
	pages := makePages(map[string][]string{
		"https://example.com/":  {"https://example.com/a", "https://example.com/b", "https://example.com/c"},
		"https://example.com/a": {"https://example.com/"},
		"https://example.com/b": {"https://example.com/"},
		"https://example.com/c": {"https://example.com/"},
	})

	g := BuildGraph(pages)
	ranks := g.ComputePageRank(20, 0.85)

	rootRank := ranks["https://example.com/"]
	for u, r := range ranks {
		if u != "https://example.com/" && r >= rootRank {
			t.Errorf("expected root to have highest rank, but %s has %f >= %f", u, r, rootRank)
		}
	}
}

func TestComputePageRank_DefaultParams(t *testing.T) {
	pages := makePages(map[string][]string{
		"https://example.com/":  {"https://example.com/a"},
		"https://example.com/a": {"https://example.com/"},
	})

	g := BuildGraph(pages)
	ranks := g.ComputePageRank(0, 0) // should use defaults

	if len(ranks) != 2 {
		t.Fatalf("expected 2 ranks, got %d", len(ranks))
	}
	for _, r := range ranks {
		if r <= 0 {
			t.Errorf("rank should be positive, got %f", r)
		}
	}
}

func TestComputePageRank_DanglingNodes(t *testing.T) {
	// Page /dead has no outgoing links (dangling).
	pages := makePages(map[string][]string{
		"https://example.com/":     {"https://example.com/dead"},
		"https://example.com/dead": nil,
	})

	g := BuildGraph(pages)
	ranks := g.ComputePageRank(20, 0.85)

	sum := 0.0
	for _, r := range ranks {
		sum += r
	}
	if math.Abs(sum-1.0) > 0.01 {
		t.Errorf("PageRank scores sum to %f with dangling node, expected ~1.0", sum)
	}
}

func TestComputePageRank_EmptyGraph(t *testing.T) {
	g := BuildGraph(nil)
	ranks := g.ComputePageRank(20, 0.85)

	if ranks != nil {
		t.Errorf("expected nil ranks for empty graph, got %v", ranks)
	}
}

func TestExportDOT(t *testing.T) {
	pages := makePages(map[string][]string{
		"https://example.com/":  {"https://example.com/a"},
		"https://example.com/a": {"https://example.com/"},
	})

	g := BuildGraph(pages)
	var buf bytes.Buffer
	if err := g.ExportDOT(&buf); err != nil {
		t.Fatalf("ExportDOT failed: %v", err)
	}

	dot := buf.String()
	if !strings.Contains(dot, "digraph linkgraph") {
		t.Errorf("expected 'digraph linkgraph' in DOT output")
	}
	if !strings.Contains(dot, "->") {
		t.Errorf("expected '->' edges in DOT output")
	}
	if !strings.HasSuffix(strings.TrimSpace(dot), "}") {
		t.Errorf("expected DOT output to end with '}'")
	}
}

func TestExportJSON(t *testing.T) {
	pages := makePages(map[string][]string{
		"https://example.com/":  {"https://example.com/a"},
		"https://example.com/a": {},
	})

	g := BuildGraph(pages)
	var buf bytes.Buffer
	if err := g.ExportJSON(&buf); err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	var data graphJSON
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(data.Nodes) != 2 {
		t.Errorf("expected 2 nodes in JSON, got %d", len(data.Nodes))
	}
	if len(data.Edges) != 1 {
		t.Errorf("expected 1 edge in JSON, got %d", len(data.Edges))
	}
}

func TestLinkGraphChecker_SiteCheckerInterface(t *testing.T) {
	c := NewLinkGraphChecker()

	if c.Name() != "link-graph" {
		t.Errorf("unexpected name: %s", c.Name())
	}

	issues := c.Check(context.Background(), &model.Page{})
	if issues != nil {
		t.Fatalf("Check should return nil, got %+v", issues)
	}
}

func TestLinkGraphChecker_HighOutgoingLinks(t *testing.T) {
	// Build a page with > 100 outgoing links.
	outgoing := make([]string, 0, 110)
	pages := make([]*model.Page, 0, 111)

	rootURL := "https://example.com/"
	for i := range 110 {
		u := "https://example.com/" + strings.Repeat("a", i+1)
		outgoing = append(outgoing, u)
		pages = append(pages, &model.Page{
			URL:         u,
			StatusCode:  200,
			ContentType: "text/html",
			Links:       []string{rootURL},
		})
	}
	pages = append(pages, &model.Page{
		URL:         rootURL,
		StatusCode:  200,
		ContentType: "text/html",
		Links:       outgoing,
	})

	c := NewLinkGraphChecker()
	issues := c.CheckSite(context.Background(), pages)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "links/high-outgoing-links" && iss.URL == rootURL {
			found = true
			if iss.Severity != model.SeverityInfo {
				t.Errorf("expected severity info, got %s", iss.Severity)
			}
			if !strings.Contains(iss.Message, "110") {
				t.Errorf("expected '110' in message, got %q", iss.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected links/high-outgoing-links issue for root")
	}
}

func TestLinkGraphChecker_LowPageRank(t *testing.T) {
	// Create a star topology: root links to many, but they don't link back.
	// Leaf nodes with no inbound (except from root) should have low rank.
	pages := make([]*model.Page, 0, 12)

	rootURL := "https://example.com/"
	leafLinks := make([]string, 0, 11)
	for i := range 11 {
		u := "https://example.com/" + string(rune('a'+i))
		leafLinks = append(leafLinks, u)
		pages = append(pages, &model.Page{
			URL:         u,
			StatusCode:  200,
			ContentType: "text/html",
			Links:       nil, // no outgoing
		})
	}
	pages = append(pages, &model.Page{
		URL:         rootURL,
		StatusCode:  200,
		ContentType: "text/html",
		Links:       leafLinks,
	})

	c := NewLinkGraphChecker()
	issues := c.CheckSite(context.Background(), pages)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "links/low-internal-pagerank" {
			found = true
			if iss.Severity != model.SeverityInfo {
				t.Errorf("expected severity info, got %s", iss.Severity)
			}
		}
	}
	if !found {
		t.Errorf("expected links/low-internal-pagerank issues")
	}
}

func TestLinkGraphChecker_EmptyPages(t *testing.T) {
	c := NewLinkGraphChecker()
	issues := c.CheckSite(context.Background(), nil)
	if issues != nil {
		t.Fatalf("expected nil for empty pages, got %+v", issues)
	}
}

func TestSameHost(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"https://example.com/a", "https://example.com/b", true},
		{"https://example.com/a", "https://other.com/b", false},
		{"https://Example.Com/a", "https://example.com/b", true},
		{"https://example.com:443/a", "https://example.com:443/b", true},
	}
	for _, tt := range tests {
		got := sameHost(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("sameHost(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestShortLabel(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/", "example.com/"},
		{"https://example.com/about", "/about"},
		{"https://example.com", "example.com/"},
	}
	for _, tt := range tests {
		got := shortLabel(tt.url)
		if got != tt.want {
			t.Errorf("shortLabel(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
