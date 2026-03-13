package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/meysam81/scry/internal/model"
)

const (
	defaultPageRankIterations = 20
	defaultDampingFactor      = 0.85
	highOutgoingLinksLimit    = 100
	lowPageRankPercentile     = 0.10
)

// LinkGraph represents the internal link structure of a website.
type LinkGraph struct {
	// nodes maps URL to outgoing internal URLs.
	nodes map[string][]string
	// allURLs is the sorted list of all node URLs (for deterministic output).
	allURLs []string
}

// BuildGraph constructs a LinkGraph from a set of crawled pages.
// Only internal links (same host) are included in the graph.
func BuildGraph(pages []*model.Page) *LinkGraph {
	pageURLs := make(map[string]bool, len(pages))
	for _, p := range pages {
		pageURLs[p.URL] = true
	}

	nodes := make(map[string][]string, len(pages))
	for _, p := range pages {
		// Ensure every page is a node even if it has no outgoing links.
		if _, ok := nodes[p.URL]; !ok {
			nodes[p.URL] = nil
		}
		for _, link := range p.Links {
			if !pageURLs[link] {
				continue
			}
			if !sameHost(p.URL, link) {
				continue
			}
			if link == p.URL {
				continue // skip self-links
			}
			nodes[p.URL] = append(nodes[p.URL], link)
		}
	}

	allURLs := make([]string, 0, len(nodes))
	for u := range nodes {
		allURLs = append(allURLs, u)
	}
	sort.Strings(allURLs)

	return &LinkGraph{
		nodes:   nodes,
		allURLs: allURLs,
	}
}

// Nodes returns the number of nodes in the graph.
func (g *LinkGraph) Nodes() int {
	return len(g.allURLs)
}

// Edges returns the total number of directed edges in the graph.
func (g *LinkGraph) Edges() int {
	total := 0
	for _, targets := range g.nodes {
		total += len(targets)
	}
	return total
}

// OutgoingLinks returns the outgoing internal links for a URL.
func (g *LinkGraph) OutgoingLinks(url string) []string {
	return g.nodes[url]
}

// ComputePageRank runs the iterative PageRank algorithm.
//
// Parameters:
//   - iterations: number of iterations to run (use 0 for default of 20)
//   - dampingFactor: the damping factor d (use 0 for default of 0.85)
//
// Returns a map of URL to PageRank score.
func (g *LinkGraph) ComputePageRank(iterations int, dampingFactor float64) map[string]float64 {
	if iterations <= 0 {
		iterations = defaultPageRankIterations
	}
	if dampingFactor <= 0 || dampingFactor >= 1 {
		dampingFactor = defaultDampingFactor
	}

	n := len(g.allURLs)
	if n == 0 {
		return nil
	}

	// Index mapping for fast lookup.
	urlIndex := make(map[string]int, n)
	for i, u := range g.allURLs {
		urlIndex[u] = i
	}

	// Build adjacency: outgoing[i] = list of indices that page i links to.
	outgoing := make([][]int, n)
	for i, u := range g.allURLs {
		for _, target := range g.nodes[u] {
			if idx, ok := urlIndex[target]; ok {
				outgoing[i] = append(outgoing[i], idx)
			}
		}
	}

	// Initialise ranks.
	rank := make([]float64, n)
	initial := 1.0 / float64(n)
	for i := range rank {
		rank[i] = initial
	}

	newRank := make([]float64, n)
	base := (1.0 - dampingFactor) / float64(n)

	for range iterations {
		// Reset.
		for i := range newRank {
			newRank[i] = base
		}

		// Distribute rank.
		for i := range n {
			if len(outgoing[i]) == 0 {
				// Dangling node: distribute rank evenly.
				share := dampingFactor * rank[i] / float64(n)
				for j := range n {
					newRank[j] += share
				}
			} else {
				share := dampingFactor * rank[i] / float64(len(outgoing[i]))
				for _, j := range outgoing[i] {
					newRank[j] += share
				}
			}
		}

		rank, newRank = newRank, rank
	}

	result := make(map[string]float64, n)
	for i, u := range g.allURLs {
		result[u] = rank[i]
	}
	return result
}

// graphJSON is the serialisation format for ExportJSON.
type graphJSON struct {
	Nodes []graphNodeJSON `json:"nodes"`
	Edges []graphEdgeJSON `json:"edges"`
}

type graphNodeJSON struct {
	URL string `json:"url"`
}

type graphEdgeJSON struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ExportDOT writes the link graph in Graphviz DOT format to w.
func (g *LinkGraph) ExportDOT(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "digraph linkgraph {"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "  rankdir=LR;"); err != nil {
		return err
	}

	for _, u := range g.allURLs {
		label := shortLabel(u)
		if _, err := fmt.Fprintf(w, "  %q [label=%q];\n", u, label); err != nil {
			return err
		}
	}

	for _, u := range g.allURLs {
		for _, target := range g.nodes[u] {
			if _, err := fmt.Fprintf(w, "  %q -> %q;\n", u, target); err != nil {
				return err
			}
		}
	}

	_, err := fmt.Fprintln(w, "}")
	return err
}

// ExportJSON writes the link graph as a JSON adjacency list to w.
func (g *LinkGraph) ExportJSON(w io.Writer) error {
	var data graphJSON
	for _, u := range g.allURLs {
		data.Nodes = append(data.Nodes, graphNodeJSON{URL: u})
		for _, target := range g.nodes[u] {
			data.Edges = append(data.Edges, graphEdgeJSON{From: u, To: target})
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// LinkGraphChecker wraps LinkGraph analysis as a SiteChecker for the audit registry.
type LinkGraphChecker struct{}

// NewLinkGraphChecker returns a new LinkGraphChecker.
func NewLinkGraphChecker() *LinkGraphChecker {
	return &LinkGraphChecker{}
}

// Name returns the checker name.
func (c *LinkGraphChecker) Name() string { return "link-graph" }

// Check returns nil; all analysis is site-wide.
func (c *LinkGraphChecker) Check(_ context.Context, _ *model.Page) []model.Issue {
	return nil
}

// CheckSite builds the link graph, computes PageRank, and generates issues.
func (c *LinkGraphChecker) CheckSite(_ context.Context, pages []*model.Page) []model.Issue {
	if len(pages) == 0 {
		return nil
	}

	graph := BuildGraph(pages)
	ranks := graph.ComputePageRank(0, 0) // use defaults

	var issues []model.Issue

	// Find low PageRank pages (bottom 10%).
	if len(ranks) > 1 {
		scores := make([]float64, 0, len(ranks))
		for _, score := range ranks {
			scores = append(scores, score)
		}
		sort.Float64s(scores)

		threshold := scores[int(float64(len(scores))*lowPageRankPercentile)]
		for u, score := range ranks {
			if score <= threshold {
				issues = append(issues, model.Issue{
					CheckName: "links/low-internal-pagerank",
					Severity:  model.SeverityInfo,
					URL:       u,
					Message:   fmt.Sprintf("page has low internal PageRank score (%.6f)", score),
				})
			}
		}
	}

	// Find pages with high outgoing internal links.
	for _, p := range pages {
		outgoing := graph.OutgoingLinks(p.URL)
		if len(outgoing) > highOutgoingLinksLimit {
			issues = append(issues, model.Issue{
				CheckName: "links/high-outgoing-links",
				Severity:  model.SeverityInfo,
				URL:       p.URL,
				Message:   fmt.Sprintf("page has %d outgoing internal links, maximum recommended is %d", len(outgoing), highOutgoingLinksLimit),
			})
		}
	}

	return issues
}

// sameHost returns true if both URLs have the same hostname.
func sameHost(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	return strings.EqualFold(ua.Host, ub.Host)
}

// shortLabel extracts the path from a URL for use as a DOT node label.
func shortLabel(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	path := u.Path
	if path == "" || path == "/" {
		return u.Host + "/"
	}
	return path
}
