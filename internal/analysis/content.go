package analysis

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/meysam81/scry/internal/model"
	"golang.org/x/net/html"
)

const (
	// minTextToHTMLRatio is the minimum acceptable text-to-HTML ratio.
	minTextToHTMLRatio = 0.10

	// minArticleWordCount is the minimum word count for article-like pages.
	minArticleWordCount = 300
)

// ContentAnalyzer checks per-page content quality metrics.
type ContentAnalyzer struct{}

// NewContentAnalyzer returns a new ContentAnalyzer.
func NewContentAnalyzer() *ContentAnalyzer {
	return &ContentAnalyzer{}
}

// Name returns the checker name.
func (c *ContentAnalyzer) Name() string { return "content" }

// Check runs per-page content quality checks.
func (c *ContentAnalyzer) Check(_ context.Context, page *model.Page) []model.Issue {
	if !isHTMLContent(page) {
		return nil
	}
	if len(page.Body) == 0 {
		return nil
	}

	var issues []model.Issue

	issues = append(issues, checkHeadingHierarchy(page.Body, page.URL)...)
	issues = append(issues, c.checkTextRatio(page)...)
	issues = append(issues, c.checkParagraphs(page)...)
	issues = append(issues, c.checkArticleWordCount(page)...)

	return issues
}

// checkTextRatio flags pages with a low text-to-HTML ratio.
func (c *ContentAnalyzer) checkTextRatio(page *model.Page) []model.Issue {
	ratio := textToHTMLRatio(page.Body)
	if ratio < minTextToHTMLRatio {
		return []model.Issue{{
			CheckName: "content/low-text-ratio",
			Severity:  model.SeverityInfo,
			URL:       page.URL,
			Message:   fmt.Sprintf("text-to-HTML ratio is %.1f%%, minimum recommended is %.0f%%", ratio*100, minTextToHTMLRatio*100),
		}}
	}
	return nil
}

// checkParagraphs flags HTML pages with zero <p> tags.
func (c *ContentAnalyzer) checkParagraphs(page *model.Page) []model.Issue {
	doc, err := html.Parse(bytes.NewReader(page.Body))
	if err != nil {
		return nil
	}
	pCount := countElements(doc, "p")
	if pCount == 0 {
		return []model.Issue{{
			CheckName: "content/no-paragraphs",
			Severity:  model.SeverityInfo,
			URL:       page.URL,
			Message:   "page contains no <p> tags",
		}}
	}
	return nil
}

// checkArticleWordCount flags article-like pages with fewer than
// minArticleWordCount words. A page is considered article-like if it has an
// <article> tag or its URL contains "/blog/" or "/post/".
func (c *ContentAnalyzer) checkArticleWordCount(page *model.Page) []model.Issue {
	if !isArticlePage(page) {
		return nil
	}
	text := extractText(page.Body)
	wc := wordCount(text)
	if wc < minArticleWordCount {
		return []model.Issue{{
			CheckName: "content/word-count-low",
			Severity:  model.SeverityInfo,
			URL:       page.URL,
			Message:   fmt.Sprintf("article page has %d words, minimum recommended is %d", wc, minArticleWordCount),
		}}
	}
	return nil
}

// checkHeadingHierarchy verifies that heading levels (h1-h6) do not skip
// levels (e.g. h1 followed by h3 without h2 in between).
func checkHeadingHierarchy(body []byte, pageURL string) []model.Issue {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil
	}

	headings := collectHeadings(doc)
	if len(headings) == 0 {
		return nil
	}

	var issues []model.Issue
	prevLevel := 0

	for _, h := range headings {
		level := headingLevel(h)
		if level == 0 {
			continue
		}
		if prevLevel > 0 && level > prevLevel+1 {
			issues = append(issues, model.Issue{
				CheckName: "content/heading-hierarchy-skip",
				Severity:  model.SeverityWarning,
				URL:       pageURL,
				Message:   fmt.Sprintf("heading level skips from h%d to h%d", prevLevel, level),
			})
		}
		prevLevel = level
	}

	return issues
}

// textToHTMLRatio returns the ratio of text content length to total body length.
func textToHTMLRatio(body []byte) float64 {
	if len(body) == 0 {
		return 0
	}
	text := extractText(body)
	return float64(len(text)) / float64(len(body))
}

// collectHeadings returns all heading elements (h1-h6) in document order.
func collectHeadings(n *html.Node) []*html.Node {
	var result []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && isHeadingTag(node.Data) {
			result = append(result, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return result
}

// isHeadingTag reports whether the tag is h1-h6.
func isHeadingTag(tag string) bool {
	switch tag {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}
	return false
}

// headingLevel returns the numeric level (1-6) for a heading node, or 0 if
// the node is not a heading.
func headingLevel(n *html.Node) int {
	if n.Type != html.ElementNode || len(n.Data) != 2 || n.Data[0] != 'h' {
		return 0
	}
	level := n.Data[1] - '0'
	if level >= 1 && level <= 6 {
		return int(level)
	}
	return 0
}

// countElements counts the number of elements with the given tag name.
func countElements(n *html.Node, tag string) int {
	count := 0
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag {
			count++
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return count
}

// isArticlePage reports whether a page appears to be an article or blog post.
func isArticlePage(page *model.Page) bool {
	urlLower := strings.ToLower(page.URL)
	if strings.Contains(urlLower, "/blog/") || strings.Contains(urlLower, "/post/") {
		return true
	}
	doc, err := html.Parse(bytes.NewReader(page.Body))
	if err != nil {
		return false
	}
	return countElements(doc, "article") > 0
}
