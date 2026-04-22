package checks

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/meysam81/scry/core/model"
	"golang.org/x/net/html"
)

// AccessibilityChecker analyses pages for common accessibility issues.
type AccessibilityChecker struct{}

// NewAccessibilityChecker returns a new AccessibilityChecker.
func NewAccessibilityChecker() *AccessibilityChecker {
	return &AccessibilityChecker{}
}

// Name returns the checker name.
func (c *AccessibilityChecker) Name() string { return "accessibility" }

// Check runs per-page accessibility checks.
func (c *AccessibilityChecker) Check(_ context.Context, page *model.Page) []model.Issue {
	if !isHTMLContent(page) {
		return nil
	}
	doc := parseHTMLDocLog(page.Body, page.URL)
	if doc == nil {
		return nil
	}

	var issues []model.Issue

	issues = append(issues, c.checkFormLabels(doc, page.URL)...)
	issues = append(issues, c.checkEmptyLinks(doc, page.URL)...)
	issues = append(issues, c.checkSkipNav(doc, page.URL)...)
	issues = append(issues, c.checkHeadingHierarchy(doc, page.URL)...)
	issues = append(issues, c.checkButtonText(doc, page.URL)...)
	issues = append(issues, c.checkTableHeaders(doc, page.URL)...)
	issues = append(issues, c.checkImgAltInFigure(doc, page.URL)...)
	issues = append(issues, c.checkPositiveTabindex(doc, page.URL)...)
	issues = append(issues, c.checkMissingLandmarks(doc, page.URL)...)
	issues = append(issues, c.checkInvalidAria(doc, page.URL)...)
	issues = append(issues, c.checkOnclickWithoutKeyboard(doc, page.URL)...)
	issues = append(issues, c.checkMissingVideoCaptions(doc, page.URL)...)
	issues = append(issues, c.checkMissingAutocomplete(doc, page.URL)...)

	return issues
}

// checkFormLabels reports <input> elements without an associated label.
//
// An input is considered labelled if any of the following are true:
//   - it has an aria-label attribute
//   - it has an aria-labelledby attribute
//   - a <label for="..."> exists whose for value matches the input's id
//   - the input is a descendant of a <label> element
//
// Hidden inputs (type="hidden") and submit/button/reset/image inputs are excluded
// because they don't require visible labels.
func (c *AccessibilityChecker) checkFormLabels(doc *html.Node, url string) []model.Issue {
	// Build a set of IDs referenced by <label for="..."> elements.
	labelFor := make(map[string]struct{})
	for _, label := range findNodes(doc, "label") {
		if f, ok := getAttr(label, "for"); ok && f != "" {
			labelFor[f] = struct{}{}
		}
	}

	var issues []model.Issue

	for _, input := range findNodes(doc, "input") {
		// Skip types that don't need labels.
		if typ, ok := getAttr(input, "type"); ok {
			switch strings.ToLower(typ) {
			case "hidden", "submit", "button", "reset", "image":
				continue
			}
		}

		// aria-label or aria-labelledby present → OK.
		if v, ok := getAttr(input, "aria-label"); ok && strings.TrimSpace(v) != "" {
			continue
		}
		if v, ok := getAttr(input, "aria-labelledby"); ok && strings.TrimSpace(v) != "" {
			continue
		}

		// Matching <label for="id"> → OK.
		if id, ok := getAttr(input, "id"); ok && id != "" {
			if _, found := labelFor[id]; found {
				continue
			}
		}

		// Wrapped inside a <label> → OK.
		if isDescendantOf(input, "label") {
			continue
		}

		issues = append(issues, model.Issue{
			CheckName: "accessibility/missing-form-label",
			Severity:  model.SeverityWarning,
			URL:       url,
			Message:   "input element is missing an associated label",
		})
	}

	return issues
}

// checkEmptyLinks reports <a> elements with no text content and no aria-label.
func (c *AccessibilityChecker) checkEmptyLinks(doc *html.Node, url string) []model.Issue {
	var issues []model.Issue

	for _, a := range findNodes(doc, "a") {
		if v, ok := getAttr(a, "aria-label"); ok && strings.TrimSpace(v) != "" {
			continue
		}

		text := strings.TrimSpace(textContent(a))
		if text != "" {
			continue
		}

		// Check for child images with alt text — counts as link text.
		if hasImgWithAlt(a) {
			continue
		}

		issues = append(issues, model.Issue{
			CheckName: "accessibility/empty-link",
			Severity:  model.SeverityWarning,
			URL:       url,
			Message:   "anchor element has no accessible text",
		})
	}

	return issues
}

// checkSkipNav reports pages missing a skip navigation link.
//
// A skip-nav link is typically the first <a> in the body with an href="#..."
// targeting a landmark (e.g. #main, #content). We check the first 3 <a> tags
// for a fragment-only href to allow for minor template variations.
func (c *AccessibilityChecker) checkSkipNav(doc *html.Node, url string) []model.Issue {
	anchors := findNodes(doc, "a")

	limit := 3
	if len(anchors) < limit {
		limit = len(anchors)
	}

	for _, a := range anchors[:limit] {
		href, ok := getAttr(a, "href")
		if ok && strings.HasPrefix(href, "#") && len(href) > 1 {
			return nil
		}
	}

	return []model.Issue{{
		CheckName: "accessibility/missing-skip-nav",
		Severity:  model.SeverityInfo,
		URL:       url,
		Message:   "page is missing a skip navigation link",
	}}
}

// checkHeadingHierarchy reports heading levels that skip (e.g. h1 → h3).
func (c *AccessibilityChecker) checkHeadingHierarchy(doc *html.Node, url string) []model.Issue {
	var levels []int
	collectHeadingLevels(doc, &levels)

	if len(levels) == 0 {
		return nil
	}

	var issues []model.Issue

	for i := 1; i < len(levels); i++ {
		if levels[i] > levels[i-1]+1 {
			issues = append(issues, model.Issue{
				CheckName: "accessibility/heading-hierarchy",
				Severity:  model.SeverityWarning,
				URL:       url,
				Message:   fmt.Sprintf("heading level skips from h%d to h%d", levels[i-1], levels[i]),
			})
		}
	}

	return issues
}

// checkButtonText reports <button> elements with no text and no aria-label.
func (c *AccessibilityChecker) checkButtonText(doc *html.Node, url string) []model.Issue {
	var issues []model.Issue

	for _, btn := range findNodes(doc, "button") {
		if v, ok := getAttr(btn, "aria-label"); ok && strings.TrimSpace(v) != "" {
			continue
		}

		text := strings.TrimSpace(textContent(btn))
		if text != "" {
			continue
		}

		issues = append(issues, model.Issue{
			CheckName: "accessibility/missing-button-text",
			Severity:  model.SeverityWarning,
			URL:       url,
			Message:   "button element has no accessible text",
		})
	}

	return issues
}

// checkTableHeaders reports <table> elements without any <th> children.
func (c *AccessibilityChecker) checkTableHeaders(doc *html.Node, url string) []model.Issue {
	var issues []model.Issue

	for _, table := range findNodes(doc, "table") {
		ths := findNodes(table, "th")
		if len(ths) == 0 {
			issues = append(issues, model.Issue{
				CheckName: "accessibility/missing-table-header",
				Severity:  model.SeverityInfo,
				URL:       url,
				Message:   "table element is missing header cells (<th>)",
			})
		}
	}

	return issues
}

// checkImgAltInFigure reports <figure> elements containing an <img> that
// has neither a <figcaption> sibling nor an alt attribute.
func (c *AccessibilityChecker) checkImgAltInFigure(doc *html.Node, url string) []model.Issue {
	var issues []model.Issue

	for _, fig := range findNodes(doc, "figure") {
		imgs := findNodes(fig, "img")
		if len(imgs) == 0 {
			continue
		}

		captions := findNodes(fig, "figcaption")
		if len(captions) > 0 {
			continue
		}

		for _, img := range imgs {
			alt, hasAlt := getAttr(img, "alt")
			if !hasAlt || strings.TrimSpace(alt) == "" {
				issues = append(issues, model.Issue{
					CheckName: "accessibility/missing-img-alt-in-figure",
					Severity:  model.SeverityWarning,
					URL:       url,
					Message:   "image inside <figure> has no alt text and no <figcaption>",
				})
			}
		}
	}

	return issues
}

// checkPositiveTabindex reports elements with tabindex > 0, which disrupts
// the natural tab order and is considered an accessibility anti-pattern.
func (c *AccessibilityChecker) checkPositiveTabindex(doc *html.Node, url string) []model.Issue {
	var issues []model.Issue
	walkAllElements(doc, func(n *html.Node) {
		val, ok := getAttr(n, "tabindex")
		if !ok {
			return
		}
		idx, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil {
			return
		}
		if idx > 0 {
			issues = append(issues, model.Issue{
				CheckName: "accessibility/positive-tabindex",
				Severity:  model.SeverityWarning,
				URL:       url,
				Message:   fmt.Sprintf("element <%s> has tabindex=%d which disrupts natural tab order", n.Data, idx),
			})
		}
	})
	return issues
}

// checkMissingLandmarks reports pages that have no ARIA landmark regions.
//
// A page is considered to have landmarks if it contains at least one of:
//   - an element with role="main", role="navigation", role="banner", or role="contentinfo"
//   - a <main>, <nav> element
//   - a top-level <header> or <footer> (i.e. not nested inside <article>)
func (c *AccessibilityChecker) checkMissingLandmarks(doc *html.Node, url string) []model.Issue {
	// Check for ARIA role-based landmarks.
	landmarkRoles := map[string]struct{}{
		"main":        {},
		"navigation":  {},
		"banner":      {},
		"contentinfo": {},
	}
	found := false
	walkAllElements(doc, func(n *html.Node) {
		if found {
			return
		}
		if role, ok := getAttr(n, "role"); ok {
			if _, match := landmarkRoles[strings.ToLower(strings.TrimSpace(role))]; match {
				found = true
				return
			}
		}
	})
	if found {
		return nil
	}

	// Check for semantic landmark elements.
	if len(findNodes(doc, "main")) > 0 || len(findNodes(doc, "nav")) > 0 {
		return nil
	}

	// Check for top-level <header> and <footer> (not inside <article>).
	for _, tag := range []string{"header", "footer"} {
		for _, node := range findNodes(doc, tag) {
			if !isDescendantOf(node, "article") {
				return nil
			}
		}
	}

	return []model.Issue{{
		CheckName: "accessibility/missing-landmarks",
		Severity:  model.SeverityWarning,
		URL:       url,
		Message:   "page has no ARIA landmark regions",
	}}
}

// checkInvalidAria reports common ARIA misuse patterns.
//
// Currently detects elements with aria-hidden="true" that also have a
// non-negative tabindex (>= 0), making them focusable but hidden from
// assistive technology.
func (c *AccessibilityChecker) checkInvalidAria(doc *html.Node, url string) []model.Issue {
	var issues []model.Issue
	walkAllElements(doc, func(n *html.Node) {
		hidden, ok := getAttr(n, "aria-hidden")
		if !ok || strings.ToLower(strings.TrimSpace(hidden)) != "true" {
			return
		}
		tabVal, ok := getAttr(n, "tabindex")
		if !ok {
			return
		}
		idx, err := strconv.Atoi(strings.TrimSpace(tabVal))
		if err != nil {
			return
		}
		if idx >= 0 {
			issues = append(issues, model.Issue{
				CheckName: "accessibility/invalid-aria",
				Severity:  model.SeverityWarning,
				URL:       url,
				Message:   fmt.Sprintf("element <%s> has aria-hidden=true with positive tabindex", n.Data),
			})
		}
	})
	return issues
}

// checkOnclickWithoutKeyboard reports non-interactive elements that have an
// onclick handler but lack tabindex and role attributes, making them
// inaccessible to keyboard users.
//
// Interactive elements (<button>, <a>, <input>, <select>, <textarea>) are excluded.
func (c *AccessibilityChecker) checkOnclickWithoutKeyboard(doc *html.Node, url string) []model.Issue {
	interactive := map[string]struct{}{
		"button":   {},
		"a":        {},
		"input":    {},
		"select":   {},
		"textarea": {},
	}

	var issues []model.Issue
	walkAllElements(doc, func(n *html.Node) {
		if _, ok := interactive[n.Data]; ok {
			return
		}
		if _, ok := getAttr(n, "onclick"); !ok {
			return
		}
		_, hasTabindex := getAttr(n, "tabindex")
		_, hasRole := getAttr(n, "role")
		if !hasTabindex && !hasRole {
			issues = append(issues, model.Issue{
				CheckName: "accessibility/onclick-without-keyboard",
				Severity:  model.SeverityWarning,
				URL:       url,
				Message:   fmt.Sprintf("non-interactive <%s> has onclick but no tabindex or role", n.Data),
			})
		}
	})
	return issues
}

// checkMissingVideoCaptions reports <video> elements that lack a <track> child
// with kind="captions" or kind="subtitles".
func (c *AccessibilityChecker) checkMissingVideoCaptions(doc *html.Node, url string) []model.Issue {
	var issues []model.Issue
	for _, video := range findNodes(doc, "video") {
		tracks := findNodes(video, "track")
		hasCaptions := false
		for _, track := range tracks {
			kind, ok := getAttr(track, "kind")
			if !ok {
				continue
			}
			k := strings.ToLower(strings.TrimSpace(kind))
			if k == "captions" || k == "subtitles" {
				hasCaptions = true
				break
			}
		}
		if !hasCaptions {
			issues = append(issues, model.Issue{
				CheckName: "accessibility/missing-video-captions",
				Severity:  model.SeverityWarning,
				URL:       url,
				Message:   "video element is missing captions or subtitles track",
			})
		}
	}
	return issues
}

// checkMissingAutocomplete reports <input> elements with common personal-data
// types that are missing the autocomplete attribute. Autocomplete helps
// password managers and assistive technology fill fields correctly.
func (c *AccessibilityChecker) checkMissingAutocomplete(doc *html.Node, url string) []model.Issue {
	autoTypes := map[string]struct{}{
		"email":       {},
		"tel":         {},
		"name":        {},
		"username":    {},
		"password":    {},
		"address":     {},
		"postal-code": {},
		"cc-number":   {},
	}

	var issues []model.Issue
	for _, input := range findNodes(doc, "input") {
		typ, ok := getAttr(input, "type")
		if !ok {
			continue
		}
		t := strings.ToLower(strings.TrimSpace(typ))
		if _, match := autoTypes[t]; !match {
			continue
		}
		if _, hasAC := getAttr(input, "autocomplete"); hasAC {
			continue
		}
		issues = append(issues, model.Issue{
			CheckName: "accessibility/missing-autocomplete",
			Severity:  model.SeverityInfo,
			URL:       url,
			Message:   fmt.Sprintf("input type=%q is missing autocomplete attribute", t),
		})
	}
	return issues
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isDescendantOf checks whether node is nested inside an element with the given tag.
func isDescendantOf(n *html.Node, tag string) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == tag {
			return true
		}
	}
	return false
}

// hasImgWithAlt returns true if n contains at least one <img> with a non-empty alt.
func hasImgWithAlt(n *html.Node) bool {
	for _, img := range findNodes(n, "img") {
		if alt, ok := getAttr(img, "alt"); ok && strings.TrimSpace(alt) != "" {
			return true
		}
	}
	return false
}

// collectHeadingLevels walks the DOM tree in document order and appends heading
// levels (1-6) to the result slice.
func collectHeadingLevels(n *html.Node, levels *[]int) {
	if n.Type == html.ElementNode {
		if len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
			*levels = append(*levels, int(n.Data[1]-'0'))
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectHeadingLevels(c, levels)
	}
}

// walkAllElements calls fn for every element node in the tree.
func walkAllElements(n *html.Node, fn func(*html.Node)) {
	if n.Type == html.ElementNode {
		fn(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkAllElements(c, fn)
	}
}
