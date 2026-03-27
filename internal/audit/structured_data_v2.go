package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meysam81/scry/internal/model"
	"github.com/meysam81/scry/internal/schema"
	"golang.org/x/net/html"
)

// microdataAttrs are HTML attributes that indicate microdata usage.
var microdataAttrs = []string{"itemscope", "itemprop", "itemtype"}

// DeepStructuredDataChecker performs in-depth validation of JSON-LD
// structured data blocks using the schema validation engine.
type DeepStructuredDataChecker struct {
	registry *schema.Registry
}

// NewDeepStructuredDataChecker returns a new DeepStructuredDataChecker using
// the provided schema.Registry.
func NewDeepStructuredDataChecker(reg *schema.Registry) *DeepStructuredDataChecker {
	return &DeepStructuredDataChecker{registry: reg}
}

// Name returns the checker name.
func (c *DeepStructuredDataChecker) Name() string { return "deep-structured-data" }

// Check runs per-page deep structured data checks.
func (c *DeepStructuredDataChecker) Check(_ context.Context, page *model.Page) []model.Issue {
	if !isHTMLContent(page) {
		return nil
	}
	doc := parseHTMLDocLog(page.Body, page.URL)
	if doc == nil {
		return nil
	}

	scripts := findNodes(doc, "script")
	var issues []model.Issue
	var allObjects []map[string]any

	for _, s := range scripts {
		t, _ := getAttr(s, "type")
		if !strings.EqualFold(t, "application/ld+json") {
			continue
		}

		content := textContent(s)
		objects, blockIssues := c.parseAndValidate(content, page.URL)
		issues = append(issues, blockIssues...)
		allObjects = append(allObjects, objects...)
	}

	// Cross-object checks across all JSON-LD blocks on the page.
	if len(allObjects) > 0 {
		crossFindings := schema.CrossCheck(allObjects)
		issues = append(issues, findingsToIssues(crossFindings, page.URL)...)
	}

	// Check for microdata attributes in the document.
	if attr := findMicrodataAttr(doc); attr != "" {
		issues = append(issues, model.Issue{
			CheckName: "structured-data/microdata-detected",
			Severity:  model.SeverityInfo,
			URL:       page.URL,
			Message:   fmt.Sprintf("HTML contains microdata attribute %q; consider migrating to JSON-LD for better tooling support", attr),
		})
	}

	return issues
}

// parseAndValidate parses a JSON-LD block and validates all objects within it.
// Returns the parsed objects (for cross-object checks) and any issues found.
func (c *DeepStructuredDataChecker) parseAndValidate(content, pageURL string) ([]map[string]any, []model.Issue) {
	// Try single object.
	var obj map[string]any
	if err := json.Unmarshal([]byte(content), &obj); err == nil {
		objects := flattenObjects(obj)
		issues := c.validateObject(obj, pageURL)
		return objects, issues
	}

	// Try array of objects.
	var arr []map[string]any
	if err := json.Unmarshal([]byte(content), &arr); err == nil {
		var allObjects []map[string]any
		var issues []model.Issue
		for _, item := range arr {
			objects := flattenObjects(item)
			allObjects = append(allObjects, objects...)
			issues = append(issues, c.validateObject(item, pageURL)...)
		}
		return allObjects, issues
	}

	// Malformed JSON handled by StructuredDataChecker.
	return nil, nil
}

// validateObject runs all schema validations on a single JSON-LD object.
func (c *DeepStructuredDataChecker) validateObject(obj map[string]any, pageURL string) []model.Issue {
	var findings []schema.Finding

	// @context validation.
	findings = append(findings, schema.ValidateContext(obj)...)

	// Core validation (type, required fields, properties, nested types).
	findings = append(findings, c.registry.Validate(obj)...)

	// Google Rich Results validation.
	findings = append(findings, c.registry.ValidateGoogle(obj)...)

	return findingsToIssues(findings, pageURL)
}

// findingsToIssues converts schema.Finding values to model.Issue values.
func findingsToIssues(findings []schema.Finding, pageURL string) []model.Issue {
	issues := make([]model.Issue, 0, len(findings))
	for _, f := range findings {
		issues = append(issues, model.Issue{
			CheckName: "structured-data/" + f.Code,
			Severity:  model.SeverityFromString(f.Severity),
			URL:       pageURL,
			Message:   f.Message,
			Detail:    f.Path,
		})
	}
	return issues
}

// flattenObjects extracts all top-level JSON-LD objects (including @graph items).
func flattenObjects(obj map[string]any) []map[string]any {
	if graph, ok := obj["@graph"]; ok {
		if graphArr, ok := graph.([]any); ok {
			var objects []map[string]any
			for _, item := range graphArr {
				if itemObj, ok := item.(map[string]any); ok {
					objects = append(objects, itemObj)
				}
			}
			return objects
		}
	}
	return []map[string]any{obj}
}

// findMicrodataAttr walks the HTML document tree and returns the first
// microdata attribute name found, or an empty string if none.
func findMicrodataAttr(doc *html.Node) string {
	var found string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != "" {
			return
		}
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				for _, mdAttr := range microdataAttrs {
					if attr.Key == mdAttr {
						found = mdAttr
						return
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return found
}
