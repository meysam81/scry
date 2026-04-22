// Package analysis provides advanced content analysis engines for crawled pages.
package analysis

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"math/bits"
	"strings"

	"github.com/meysam81/scry/core/model"
	"golang.org/x/net/html"
)

const (
	// defaultThinContentThreshold is the minimum word count for a page to
	// not be flagged as thin content.
	defaultThinContentThreshold = 100

	// simHashDistanceThreshold is the maximum Hamming distance between two
	// SimHash values for them to be considered near-duplicates.
	simHashDistanceThreshold = 3
)

// DuplicateDetector finds exact duplicates, near-duplicates, and thin content
// across a set of crawled pages.
type DuplicateDetector struct {
	// ThinContentThreshold is the minimum word count below which a page is
	// flagged as thin content. Defaults to defaultThinContentThreshold.
	ThinContentThreshold int
}

// NewDuplicateDetector returns a DuplicateDetector with default settings.
func NewDuplicateDetector() *DuplicateDetector {
	return &DuplicateDetector{
		ThinContentThreshold: defaultThinContentThreshold,
	}
}

// Name returns the checker name.
func (d *DuplicateDetector) Name() string { return "content-duplicates" }

// Check returns nil; all analysis is site-wide.
func (d *DuplicateDetector) Check(_ context.Context, _ *model.Page) []model.Issue {
	return nil
}

// CheckSite runs duplicate and thin content detection across all pages.
func (d *DuplicateDetector) CheckSite(_ context.Context, pages []*model.Page) []model.Issue {
	return d.Analyze(pages)
}

// Analyze examines all pages for exact duplicates, near-duplicates, and thin content.
func (d *DuplicateDetector) Analyze(pages []*model.Page) []model.Issue {
	threshold := d.ThinContentThreshold
	if threshold <= 0 {
		threshold = defaultThinContentThreshold
	}

	type pageInfo struct {
		url     string
		sha     string
		simhash uint64
		words   int
	}

	var infos []pageInfo
	for _, p := range pages {
		if !isHTMLContent(p) {
			continue
		}
		text := extractText(p.Body)
		wc := wordCount(text)
		sha := sha256Hex(p.Body)
		sh := simHash(text)
		infos = append(infos, pageInfo{
			url:     p.URL,
			sha:     sha,
			simhash: sh,
			words:   wc,
		})
	}

	var issues []model.Issue

	// Exact duplicates: group by SHA-256 of body.
	shaGroups := make(map[string][]string)
	for _, info := range infos {
		shaGroups[info.sha] = append(shaGroups[info.sha], info.url)
	}
	// Track URLs already reported as exact duplicates so we don't also flag
	// them as near-duplicates.
	exactPairs := make(map[[2]string]bool)
	for _, urls := range shaGroups {
		if len(urls) < 2 {
			continue
		}
		for i := 0; i < len(urls); i++ {
			for j := i + 1; j < len(urls); j++ {
				issues = append(issues, model.Issue{
					CheckName: "content/exact-duplicate",
					Severity:  model.SeverityWarning,
					URL:       urls[i],
					Message:   fmt.Sprintf("page is an exact duplicate of %s", urls[j]),
				})
				exactPairs[[2]string{urls[i], urls[j]}] = true
				exactPairs[[2]string{urls[j], urls[i]}] = true
			}
		}
	}

	// Near-duplicates: compare SimHash with Hamming distance.
	for i := 0; i < len(infos); i++ {
		for j := i + 1; j < len(infos); j++ {
			a, b := infos[i], infos[j]
			if exactPairs[[2]string{a.url, b.url}] {
				continue
			}
			dist := hammingDistance(a.simhash, b.simhash)
			if dist <= simHashDistanceThreshold {
				similarity := 100.0 * float64(64-dist) / 64.0
				issues = append(issues, model.Issue{
					CheckName: "content/near-duplicate",
					Severity:  model.SeverityInfo,
					URL:       a.url,
					Message:   fmt.Sprintf("page is near-duplicate of %s (similarity: %.0f%%)", b.url, similarity),
				})
			}
		}
	}

	// Thin content.
	for _, info := range infos {
		if info.words < threshold {
			issues = append(issues, model.Issue{
				CheckName: "content/thin-content",
				Severity:  model.SeverityInfo,
				URL:       info.url,
				Message:   fmt.Sprintf("page has only %d words of text content", info.words),
			})
		}
	}

	return issues
}

// simHash computes a 64-bit SimHash of the given text.
//
// Algorithm:
//  1. Tokenize text into words (split on whitespace).
//  2. Hash each word using FNV-64a.
//  3. For each bit position 0-63: if the bit is set, add +1; otherwise add -1.
//  4. Final hash: bit is 1 if sum > 0, else 0.
func simHash(text string) uint64 {
	words := strings.Fields(text)
	if len(words) == 0 {
		return 0
	}

	var v [64]int
	for _, word := range words {
		h := fnv.New64a()
		h.Write([]byte(word))
		hash := h.Sum64()
		for i := range 64 {
			if hash&(1<<uint(i)) != 0 {
				v[i]++
			} else {
				v[i]--
			}
		}
	}

	var result uint64
	for i := range 64 {
		if v[i] > 0 {
			result |= 1 << uint(i)
		}
	}
	return result
}

// hammingDistance returns the number of differing bits between two uint64 values.
func hammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// sha256Hex returns the lowercase hex-encoded SHA-256 digest of data.
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// extractText strips HTML tags from body and returns normalised plain text.
func extractText(body []byte) string {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return ""
	}
	var sb strings.Builder
	extractTextNode(doc, &sb)
	return normalizeWhitespace(sb.String())
}

// extractTextNode recursively appends text content from a node tree,
// skipping script and style elements.
func extractTextNode(n *html.Node, sb *strings.Builder) {
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
		return
	}
	if n.Type == html.TextNode {
		sb.WriteString(n.Data)
		sb.WriteByte(' ')
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractTextNode(c, sb)
	}
}

// normalizeWhitespace collapses runs of whitespace into a single space and trims.
func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// wordCount returns the number of whitespace-delimited tokens in text.
func wordCount(text string) int {
	return len(strings.Fields(text))
}

// isHTMLContent reports whether the page's ContentType indicates HTML.
func isHTMLContent(page *model.Page) bool {
	return strings.Contains(strings.ToLower(page.ContentType), "text/html")
}
