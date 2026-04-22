package schema

import (
	"strings"
	"testing"
)

func TestCrossCheck_BreadcrumbPositions_Valid(t *testing.T) {
	objects := []map[string]any{
		{
			"@type": "BreadcrumbList",
			"itemListElement": []any{
				map[string]any{"@type": "ListItem", "position": float64(1), "name": "Home"},
				map[string]any{"@type": "ListItem", "position": float64(2), "name": "Blog"},
				map[string]any{"@type": "ListItem", "position": float64(3), "name": "Post"},
			},
		},
	}
	findings := CrossCheck(objects)
	if hasCode(findings, "breadcrumb-positions") {
		t.Errorf("expected no breadcrumb-positions issue, got %+v", findings)
	}
}

func TestCrossCheck_BreadcrumbPositions_Invalid(t *testing.T) {
	objects := []map[string]any{
		{
			"@type": "BreadcrumbList",
			"itemListElement": []any{
				map[string]any{"@type": "ListItem", "position": float64(1), "name": "Home"},
				map[string]any{"@type": "ListItem", "position": float64(3), "name": "Blog"},
			},
		},
	}
	findings := CrossCheck(objects)
	if !hasCode(findings, "breadcrumb-positions") {
		t.Errorf("expected breadcrumb-positions issue, got %+v", findings)
	}
}

func TestCrossCheck_BreadcrumbPositions_NotStartingAtOne(t *testing.T) {
	objects := []map[string]any{
		{
			"@type": "BreadcrumbList",
			"itemListElement": []any{
				map[string]any{"@type": "ListItem", "position": float64(0), "name": "Home"},
				map[string]any{"@type": "ListItem", "position": float64(1), "name": "Blog"},
			},
		},
	}
	findings := CrossCheck(objects)
	if !hasCode(findings, "breadcrumb-positions") {
		t.Errorf("expected breadcrumb-positions issue for position starting at 0, got %+v", findings)
	}
}

func TestCrossCheck_DuplicateType(t *testing.T) {
	objects := []map[string]any{
		{"@type": "Article", "headline": "One"},
		{"@type": "Article", "headline": "Two"},
	}
	findings := CrossCheck(objects)
	if !hasCode(findings, "duplicate-type") {
		t.Errorf("expected duplicate-type, got %+v", findings)
	}
}

func TestCrossCheck_NoDuplicateType(t *testing.T) {
	objects := []map[string]any{
		{"@type": "Article", "headline": "One"},
		{"@type": "Product", "name": "Widget"},
	}
	findings := CrossCheck(objects)
	if hasCode(findings, "duplicate-type") {
		t.Errorf("did not expect duplicate-type, got %+v", findings)
	}
}

func TestCrossCheck_SearchAction_Valid(t *testing.T) {
	objects := []map[string]any{
		{
			"@type": "WebSite",
			"potentialAction": map[string]any{
				"@type":       "SearchAction",
				"target":      "https://example.com/search?q={search_term_string}",
				"query-input": "required name=search_term_string",
			},
		},
	}
	findings := CrossCheck(objects)
	if hasCode(findings, "search-action-template") {
		t.Errorf("did not expect search-action-template issue, got %+v", findings)
	}
}

func TestCrossCheck_SearchAction_MissingTemplate(t *testing.T) {
	objects := []map[string]any{
		{
			"@type": "WebSite",
			"potentialAction": map[string]any{
				"@type":  "SearchAction",
				"target": "https://example.com/search",
			},
		},
	}
	findings := CrossCheck(objects)
	if !hasCode(findings, "search-action-template") {
		t.Errorf("expected search-action-template issue, got %+v", findings)
	}
	f, _ := findByCode(findings, "search-action-template")
	if !strings.Contains(f.Message, "{search_term_string}") {
		t.Errorf("expected message to mention template variable, got %q", f.Message)
	}
}

func TestCrossCheck_SearchAction_EntryPointObject(t *testing.T) {
	objects := []map[string]any{
		{
			"@type": "WebSite",
			"potentialAction": map[string]any{
				"@type": "SearchAction",
				"target": map[string]any{
					"@type":       "EntryPoint",
					"urlTemplate": "https://example.com/search?q={search_term_string}",
				},
				"query-input": "required name=search_term_string",
			},
		},
	}
	findings := CrossCheck(objects)
	if hasCode(findings, "search-action-template") {
		t.Errorf("did not expect search-action-template for EntryPoint with valid template, got %+v", findings)
	}
}

func TestCrossCheck_SearchAction_EntryPointMissingTemplate(t *testing.T) {
	objects := []map[string]any{
		{
			"@type": "WebSite",
			"potentialAction": map[string]any{
				"@type": "SearchAction",
				"target": map[string]any{
					"@type":       "EntryPoint",
					"urlTemplate": "https://example.com/search",
				},
			},
		},
	}
	findings := CrossCheck(objects)
	if !hasCode(findings, "search-action-template") {
		t.Errorf("expected search-action-template for EntryPoint missing template var, got %+v", findings)
	}
}
