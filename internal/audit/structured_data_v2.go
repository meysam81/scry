package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/meysam81/scry/internal/model"
	"golang.org/x/net/html"
)

// commonSchemaTypes is the set of widely-used Schema.org types that the
// checker recognises.
var commonSchemaTypes = map[string]struct{}{
	"Article":        {},
	"BlogPosting":    {},
	"Product":        {},
	"FAQPage":        {},
	"BreadcrumbList": {},
	"Organization":   {},
	"Person":         {},
	"WebPage":        {},
	"WebSite":        {},
	"LocalBusiness":  {},
	"Event":          {},
	"Recipe":         {},
	"HowTo":          {},
	"VideoObject":    {},
}

// typeRequiredFields maps specific Schema.org types to the fields that must
// be present for the structured data to be considered complete.
var typeRequiredFields = map[string]struct {
	checkName string
	severity  model.Severity
	fields    []string
}{
	"Article": {
		checkName: "structured-data/article-missing-fields",
		severity:  model.SeverityWarning,
		fields:    []string{"headline", "datePublished", "author"},
	},
	"BlogPosting": {
		checkName: "structured-data/article-missing-fields",
		severity:  model.SeverityWarning,
		fields:    []string{"headline", "datePublished", "author"},
	},
	"Product": {
		checkName: "structured-data/product-missing-fields",
		severity:  model.SeverityWarning,
		fields:    []string{"name", "description"},
	},
	"FAQPage": {
		checkName: "structured-data/faq-missing-fields",
		severity:  model.SeverityWarning,
		fields:    []string{"mainEntity"},
	},
	"BreadcrumbList": {
		checkName: "structured-data/breadcrumb-missing-fields",
		severity:  model.SeverityInfo,
		fields:    []string{"itemListElement"},
	},
	"Event": {
		checkName: "structured-data/event-missing-fields",
		severity:  model.SeverityWarning,
		fields:    []string{"name", "startDate", "location"},
	},
	"Recipe": {
		checkName: "structured-data/recipe-missing-fields",
		severity:  model.SeverityWarning,
		fields:    []string{"name", "image", "recipeIngredient"},
	},
	"VideoObject": {
		checkName: "structured-data/video-missing-fields",
		severity:  model.SeverityWarning,
		fields:    []string{"name", "description", "thumbnailUrl", "uploadDate"},
	},
	"LocalBusiness": {
		checkName: "structured-data/localbusiness-missing-fields",
		severity:  model.SeverityWarning,
		fields:    []string{"name", "address", "telephone"},
	},
}

// dateFields are JSON-LD fields that should contain ISO 8601 date values.
var dateFields = map[string]struct{}{
	"datePublished": {},
	"dateCreated":   {},
	"dateModified":  {},
	"startDate":     {},
	"endDate":       {},
	"uploadDate":    {},
}

// urlFields are JSON-LD fields that should contain valid URL values.
var urlFields = map[string]struct{}{
	"url":          {},
	"image":        {},
	"logo":         {},
	"sameAs":       {},
	"thumbnailUrl": {},
}

// iso8601DateRe matches dates starting with YYYY-MM-DD, optionally followed by
// time components.
var iso8601DateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// microdataAttrs are HTML attributes that indicate microdata usage.
var microdataAttrs = []string{"itemscope", "itemprop", "itemtype"}

// DeepStructuredDataChecker performs in-depth validation of JSON-LD
// structured data blocks, checking for @type presence, recognised types,
// and required fields per type.
type DeepStructuredDataChecker struct{}

// NewDeepStructuredDataChecker returns a new DeepStructuredDataChecker.
func NewDeepStructuredDataChecker() *DeepStructuredDataChecker {
	return &DeepStructuredDataChecker{}
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

	for _, s := range scripts {
		t, _ := getAttr(s, "type")
		if !strings.EqualFold(t, "application/ld+json") {
			continue
		}

		content := textContent(s)
		issues = append(issues, c.checkJSONLD(content, page.URL)...)
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

// checkJSONLD validates a single JSON-LD block's content.
func (c *DeepStructuredDataChecker) checkJSONLD(content, pageURL string) []model.Issue {
	// Try to parse as a single object first.
	var obj map[string]any
	if err := json.Unmarshal([]byte(content), &obj); err == nil {
		return c.checkObject(obj, pageURL)
	}

	// Try to parse as an array of objects.
	var arr []map[string]any
	if err := json.Unmarshal([]byte(content), &arr); err == nil {
		var issues []model.Issue
		for _, item := range arr {
			issues = append(issues, c.checkObject(item, pageURL)...)
		}
		return issues
	}

	// Malformed JSON is handled by the existing StructuredDataChecker.
	return nil
}

// checkObject validates a single JSON-LD object for type and required fields.
func (c *DeepStructuredDataChecker) checkObject(obj map[string]any, pageURL string) []model.Issue {
	var issues []model.Issue

	// Check for @graph.
	if graph, ok := obj["@graph"]; ok {
		if graphArr, ok := graph.([]any); ok {
			for _, item := range graphArr {
				if itemObj, ok := item.(map[string]any); ok {
					issues = append(issues, c.checkObject(itemObj, pageURL)...)
				}
			}
			return issues
		}
	}

	typeVal, hasType := obj["@type"]
	if !hasType {
		return []model.Issue{{
			CheckName: "structured-data/missing-type",
			Severity:  model.SeverityWarning,
			URL:       pageURL,
			Message:   "JSON-LD block has no @type field",
		}}
	}

	// @type can be a string or an array of strings.
	types := extractTypes(typeVal)
	for _, typeName := range types {
		if _, known := commonSchemaTypes[typeName]; !known {
			issues = append(issues, model.Issue{
				CheckName: "structured-data/unknown-type",
				Severity:  model.SeverityInfo,
				URL:       pageURL,
				Message:   fmt.Sprintf("JSON-LD @type %q is not a commonly recognised Schema.org type", typeName),
			})
		}

		if req, ok := typeRequiredFields[typeName]; ok {
			var missing []string
			for _, field := range req.fields {
				if _, exists := obj[field]; !exists {
					missing = append(missing, field)
				}
			}
			if len(missing) > 0 {
				issues = append(issues, model.Issue{
					CheckName: req.checkName,
					Severity:  req.severity,
					URL:       pageURL,
					Message:   fmt.Sprintf("%s is missing required fields: %s", typeName, strings.Join(missing, ", ")),
				})
			}
		}
	}

	// Validate date and URL field values.
	issues = append(issues, c.checkDateFields(obj, pageURL)...)
	issues = append(issues, c.checkURLFields(obj, pageURL)...)

	return issues
}

// checkDateFields validates that known date fields in the JSON-LD object
// contain ISO 8601 formatted values.
func (c *DeepStructuredDataChecker) checkDateFields(obj map[string]any, pageURL string) []model.Issue {
	var issues []model.Issue
	for field := range dateFields {
		val, exists := obj[field]
		if !exists {
			continue
		}
		str, ok := val.(string)
		if !ok {
			// Non-string value (e.g. number, object) in a date field.
			issues = append(issues, model.Issue{
				CheckName: "structured-data/invalid-date-format",
				Severity:  model.SeverityWarning,
				URL:       pageURL,
				Message:   fmt.Sprintf("JSON-LD field %q has a non-string value; expected ISO 8601 date", field),
			})
			continue
		}
		str = strings.TrimSpace(str)
		if str == "" || !iso8601DateRe.MatchString(str) {
			issues = append(issues, model.Issue{
				CheckName: "structured-data/invalid-date-format",
				Severity:  model.SeverityWarning,
				URL:       pageURL,
				Message:   fmt.Sprintf("JSON-LD field %q has invalid date value %q; expected ISO 8601 format (YYYY-MM-DD)", field, str),
			})
		}
	}
	return issues
}

// checkURLFields validates that known URL fields in the JSON-LD object
// contain values that look like valid URLs.
func (c *DeepStructuredDataChecker) checkURLFields(obj map[string]any, pageURL string) []model.Issue {
	var issues []model.Issue
	for field := range urlFields {
		val, exists := obj[field]
		if !exists {
			continue
		}
		// URL fields can be strings or arrays of strings (e.g. sameAs).
		switch v := val.(type) {
		case string:
			if issue, ok := checkSingleURL(field, v, pageURL); ok {
				issues = append(issues, issue)
			}
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					if issue, ok := checkSingleURL(field, s, pageURL); ok {
						issues = append(issues, issue)
					}
				}
			}
		case map[string]any:
			// Nested object (e.g. {"@type":"ImageObject","url":"..."}), skip.
		default:
			issues = append(issues, model.Issue{
				CheckName: "structured-data/invalid-url-field",
				Severity:  model.SeverityWarning,
				URL:       pageURL,
				Message:   fmt.Sprintf("JSON-LD field %q has a non-string/non-array value; expected a URL", field),
			})
		}
	}
	return issues
}

// checkSingleURL validates a single URL string value and returns an issue if
// the value does not look like a valid URL.
func checkSingleURL(field, val, pageURL string) (model.Issue, bool) {
	v := strings.TrimSpace(val)
	if v == "" || (!strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") && !strings.HasPrefix(v, "/")) {
		return model.Issue{
			CheckName: "structured-data/invalid-url-field",
			Severity:  model.SeverityWarning,
			URL:       pageURL,
			Message:   fmt.Sprintf("JSON-LD field %q has invalid URL value %q; expected http://, https://, or /", field, v),
		}, true
	}
	return model.Issue{}, false
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

// extractTypes normalises @type to a slice of strings. The value may be a
// single string, an array of strings, or an array of mixed types.
func extractTypes(v any) []string {
	switch val := v.(type) {
	case string:
		return []string{val}
	case []any:
		var types []string
		for _, item := range val {
			if s, ok := item.(string); ok {
				types = append(types, s)
			}
		}
		return types
	default:
		return nil
	}
}
