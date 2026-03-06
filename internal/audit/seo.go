package audit

import (
	"context"
	"fmt"
	"strings"

	"github.com/meysam81/scry/internal/model"
	"golang.org/x/net/html"
)

const (
	titleMinLen       = 30
	titleMaxLen       = 60
	descriptionMinLen = 70
	descriptionMaxLen = 155
)

// SEOChecker analyses pages for common SEO issues.
type SEOChecker struct{}

// NewSEOChecker returns a new SEOChecker.
func NewSEOChecker() *SEOChecker {
	return &SEOChecker{}
}

// Name returns the checker name.
func (c *SEOChecker) Name() string { return "seo" }

// Check runs per-page SEO checks.
func (c *SEOChecker) Check(_ context.Context, page *model.Page) []model.Issue {
	if !isHTMLContent(page) {
		return nil
	}
	doc, err := parseHTMLDoc(page.Body)
	if err != nil {
		return nil
	}

	var issues []model.Issue

	issues = append(issues, c.checkTitle(doc, page.URL)...)
	issues = append(issues, c.checkMetaDescription(doc, page.URL)...)
	issues = append(issues, c.checkH1(doc, page.URL)...)
	issues = append(issues, c.checkCanonical(doc, page.URL)...)
	issues = append(issues, c.checkLang(doc, page.URL)...)
	issues = append(issues, c.checkOpenGraph(doc, page.URL)...)

	return issues
}

// CheckSite runs site-wide SEO checks.
func (c *SEOChecker) CheckSite(_ context.Context, pages []*model.Page) []model.Issue {
	var issues []model.Issue
	for _, p := range pages {
		if !isHTMLContent(p) || !p.InSitemap {
			continue
		}
		doc, err := parseHTMLDoc(p.Body)
		if err != nil {
			continue
		}
		robots := findMeta(doc, "robots")
		if strings.Contains(strings.ToLower(robots), "noindex") {
			issues = append(issues, model.Issue{
				CheckName: "seo/noindex-in-sitemap",
				Severity:  model.SeverityCritical,
				URL:       p.URL,
				Message:   "page has noindex but is included in the sitemap",
			})
		}
	}
	return issues
}

func (c *SEOChecker) checkTitle(doc *html.Node, url string) []model.Issue {
	titles := findNodes(doc, "title")
	if len(titles) == 0 {
		return []model.Issue{{
			CheckName: "seo/missing-title",
			Severity:  model.SeverityCritical,
			URL:       url,
			Message:   "page is missing a <title> tag",
		}}
	}

	text := textContent(titles[0])
	length := len(text)
	if length < titleMinLen || length > titleMaxLen {
		return []model.Issue{{
			CheckName: "seo/title-length",
			Severity:  model.SeverityWarning,
			URL:       url,
			Message:   fmt.Sprintf("title length is %d characters, expected %d-%d", length, titleMinLen, titleMaxLen),
		}}
	}
	return nil
}

func (c *SEOChecker) checkMetaDescription(doc *html.Node, url string) []model.Issue {
	desc := findMeta(doc, "description")
	if desc == "" {
		return []model.Issue{{
			CheckName: "seo/missing-meta-description",
			Severity:  model.SeverityWarning,
			URL:       url,
			Message:   "page is missing a meta description",
		}}
	}

	length := len(desc)
	if length < descriptionMinLen || length > descriptionMaxLen {
		return []model.Issue{{
			CheckName: "seo/meta-description-length",
			Severity:  model.SeverityInfo,
			URL:       url,
			Message:   fmt.Sprintf("meta description length is %d characters, expected %d-%d", length, descriptionMinLen, descriptionMaxLen),
		}}
	}
	return nil
}

func (c *SEOChecker) checkH1(doc *html.Node, url string) []model.Issue {
	h1s := findNodes(doc, "h1")
	if len(h1s) == 0 {
		return []model.Issue{{
			CheckName: "seo/missing-h1",
			Severity:  model.SeverityWarning,
			URL:       url,
			Message:   "page is missing an <h1> tag",
		}}
	}
	if len(h1s) > 1 {
		return []model.Issue{{
			CheckName: "seo/multiple-h1",
			Severity:  model.SeverityWarning,
			URL:       url,
			Message:   fmt.Sprintf("page has %d <h1> tags, expected 1", len(h1s)),
		}}
	}
	return nil
}

func (c *SEOChecker) checkCanonical(doc *html.Node, url string) []model.Issue {
	for _, link := range findNodes(doc, "link") {
		rel, _ := getAttr(link, "rel")
		if strings.EqualFold(rel, "canonical") {
			return nil
		}
	}
	return []model.Issue{{
		CheckName: "seo/missing-canonical",
		Severity:  model.SeverityWarning,
		URL:       url,
		Message:   "page is missing a canonical link",
	}}
}

func (c *SEOChecker) checkLang(doc *html.Node, url string) []model.Issue {
	for _, n := range findNodes(doc, "html") {
		if _, ok := getAttr(n, "lang"); ok {
			return nil
		}
	}
	return []model.Issue{{
		CheckName: "seo/missing-lang",
		Severity:  model.SeverityWarning,
		URL:       url,
		Message:   "page is missing a lang attribute on the <html> tag",
	}}
}

func (c *SEOChecker) checkOpenGraph(doc *html.Node, url string) []model.Issue {
	var issues []model.Issue
	if findMetaProperty(doc, "og:title") == "" {
		issues = append(issues, model.Issue{
			CheckName: "seo/missing-og-title",
			Severity:  model.SeverityInfo,
			URL:       url,
			Message:   "page is missing an og:title meta tag",
		})
	}
	if findMetaProperty(doc, "og:description") == "" {
		issues = append(issues, model.Issue{
			CheckName: "seo/missing-og-description",
			Severity:  model.SeverityInfo,
			URL:       url,
			Message:   "page is missing an og:description meta tag",
		})
	}
	if findMetaProperty(doc, "og:image") == "" {
		issues = append(issues, model.Issue{
			CheckName: "seo/missing-og-image",
			Severity:  model.SeverityInfo,
			URL:       url,
			Message:   "page is missing an og:image meta tag",
		})
	}
	return issues
}

// textContent returns the concatenated text content of a node and its children.
func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}
