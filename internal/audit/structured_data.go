package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meysam81/scry/internal/model"
	"golang.org/x/net/html"
)

// StructuredDataChecker analyses pages for structured data (JSON-LD) issues.
type StructuredDataChecker struct{}

// NewStructuredDataChecker returns a new StructuredDataChecker.
func NewStructuredDataChecker() *StructuredDataChecker {
	return &StructuredDataChecker{}
}

// Name returns the checker name.
func (c *StructuredDataChecker) Name() string { return "structured-data" }

// Check runs per-page structured data checks.
func (c *StructuredDataChecker) Check(_ context.Context, page *model.Page) []model.Issue {
	if !isHTMLContent(page) {
		return nil
	}
	doc := parseHTMLDocLog(page.Body, page.URL)
	if doc == nil {
		return nil
	}

	scripts := findNodes(doc, "script")
	var ldScripts []*html.Node
	for _, s := range scripts {
		t, _ := getAttr(s, "type")
		if strings.EqualFold(t, "application/ld+json") {
			ldScripts = append(ldScripts, s)
		}
	}

	if len(ldScripts) == 0 {
		return []model.Issue{{
			CheckName: "structured-data/missing-json-ld",
			Severity:  model.SeverityInfo,
			URL:       page.URL,
			Message:   "page has no JSON-LD structured data",
		}}
	}

	var issues []model.Issue
	for _, s := range ldScripts {
		content := textContent(s)
		var js json.RawMessage
		if err := json.Unmarshal([]byte(content), &js); err != nil {
			issues = append(issues, model.Issue{
				CheckName: "structured-data/malformed-json-ld",
				Severity:  model.SeverityWarning,
				URL:       page.URL,
				Message:   fmt.Sprintf("JSON-LD block contains invalid JSON: %s", err.Error()),
			})
		}
	}

	return issues
}
