package analysis

import (
	"strings"

	"github.com/meysam81/scry/internal/model"
)

// PageClass represents the classification of a page.
type PageClass string

const (
	ClassHomepage PageClass = "homepage"
	ClassBlog     PageClass = "blog"
	ClassProduct  PageClass = "product"
	ClassCategory PageClass = "category"
	ClassContact  PageClass = "contact"
	ClassAbout    PageClass = "about"
	ClassLegal    PageClass = "legal"
	ClassAPI      PageClass = "api"
	ClassOther    PageClass = "other"
)

// ClassifiedPage pairs a page URL with its detected class and confidence.
type ClassifiedPage struct {
	URL   string    `json:"url"`
	Class PageClass `json:"class"`
	Score float64   `json:"score"` // confidence 0-1
}

// classRule defines a heuristic for classifying a page.
type classRule struct {
	class PageClass
	score float64
	match func(page *model.Page, pathLower string, bodyLower string) bool
}

// classificationRules is the ordered set of heuristic rules. Rules are
// evaluated in declaration order; the highest-scoring match wins. On ties,
// the first matching rule wins because we use strict greater-than comparison.
var classificationRules = []classRule{
	// ── Homepage ──
	{
		class: ClassHomepage,
		score: 0.9,
		match: func(page *model.Page, pathLower string, _ string) bool {
			return page.Depth == 0 && isRootPath(pathLower)
		},
	},

	// ── API ──
	{
		class: ClassAPI,
		score: 0.9,
		match: func(_ *model.Page, pathLower string, _ string) bool {
			return strings.Contains(pathLower, "/api/")
		},
	},
	{
		class: ClassAPI,
		score: 0.8,
		match: func(page *model.Page, _ string, _ string) bool {
			return strings.Contains(strings.ToLower(page.ContentType), "application/json")
		},
	},

	// ── Contact ──
	{
		class: ClassContact,
		score: 0.8,
		match: func(_ *model.Page, pathLower string, _ string) bool {
			return strings.Contains(pathLower, "/contact")
		},
	},

	// ── About ──
	{
		class: ClassAbout,
		score: 0.8,
		match: func(_ *model.Page, pathLower string, _ string) bool {
			return strings.Contains(pathLower, "/about")
		},
	},

	// ── Legal ──
	{
		class: ClassLegal,
		score: 0.8,
		match: func(_ *model.Page, pathLower string, _ string) bool {
			return strings.Contains(pathLower, "/privacy") ||
				strings.Contains(pathLower, "/terms") ||
				strings.Contains(pathLower, "/legal") ||
				strings.Contains(pathLower, "/cookie")
		},
	},

	// ── Blog (URL) ──
	{
		class: ClassBlog,
		score: 0.7,
		match: func(_ *model.Page, pathLower string, _ string) bool {
			return strings.Contains(pathLower, "/blog/") ||
				strings.Contains(pathLower, "/post/") ||
				strings.Contains(pathLower, "/article/")
		},
	},
	// ── Blog (HTML article tag) ──
	{
		class: ClassBlog,
		score: 0.3,
		match: func(_ *model.Page, _ string, bodyLower string) bool {
			return strings.Contains(bodyLower, "<article")
		},
	},

	// ── Product ──
	{
		class: ClassProduct,
		score: 0.7,
		match: func(_ *model.Page, pathLower string, _ string) bool {
			return strings.Contains(pathLower, "/product/") ||
				strings.Contains(pathLower, "/shop/") ||
				strings.Contains(pathLower, "/item/")
		},
	},
	{
		class: ClassProduct,
		score: 0.5,
		match: func(_ *model.Page, _ string, bodyLower string) bool {
			// Detect Product structured data (JSON-LD or microdata).
			return strings.Contains(bodyLower, `"@type":"product"`) ||
				strings.Contains(bodyLower, `"@type": "product"`) ||
				strings.Contains(bodyLower, `itemtype="http://schema.org/product"`) ||
				strings.Contains(bodyLower, `itemtype="https://schema.org/product"`)
		},
	},

	// ── Category ──
	{
		class: ClassCategory,
		score: 0.7,
		match: func(_ *model.Page, pathLower string, _ string) bool {
			return strings.Contains(pathLower, "/category/") ||
				strings.Contains(pathLower, "/tag/") ||
				strings.Contains(pathLower, "/collection/")
		},
	},
}

// ClassifyPages classifies each page by applying heuristic rules and returning
// the highest-scoring class. Pages that match no rules are classified as "other"
// with a score of 0.
func ClassifyPages(pages []*model.Page) []ClassifiedPage {
	result := make([]ClassifiedPage, 0, len(pages))

	for _, page := range pages {
		pathLower := strings.ToLower(extractPath(page.URL))
		bodyLower := strings.ToLower(string(page.Body))

		bestClass := ClassOther
		bestScore := 0.0

		for _, rule := range classificationRules {
			if rule.match(page, pathLower, bodyLower) {
				if rule.score > bestScore {
					bestClass = rule.class
					bestScore = rule.score
				}
			}
		}

		result = append(result, ClassifiedPage{
			URL:   page.URL,
			Class: bestClass,
			Score: bestScore,
		})
	}

	return result
}

// extractPath returns the path component of a URL string. If parsing fails,
// returns the original string.
func extractPath(rawURL string) string {
	// Find the start of the path after the scheme+authority.
	// We avoid importing net/url to keep this lightweight.
	idx := strings.Index(rawURL, "://")
	if idx < 0 {
		return rawURL
	}
	rest := rawURL[idx+3:]
	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		return "/"
	}
	path := rest[slashIdx:]
	// Strip query string and fragment.
	if qi := strings.Index(path, "?"); qi >= 0 {
		path = path[:qi]
	}
	if fi := strings.Index(path, "#"); fi >= 0 {
		path = path[:fi]
	}
	return path
}

// isRootPath reports whether the path is the site root (/ or empty).
func isRootPath(path string) bool {
	return path == "/" || path == ""
}
