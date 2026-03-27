package schema

import (
	"strings"
	"testing"
)

// helper to check if findings contain a code.
func hasCode(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

// helper to find a specific finding by code.
func findByCode(findings []Finding, code string) (Finding, bool) {
	for _, f := range findings {
		if f.Code == code {
			return f, true
		}
	}
	return Finding{}, false
}

func testRegistry(t *testing.T) *Registry {
	t.Helper()
	reg, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("load embedded: %v", err)
	}
	return reg
}

func TestValidate_MissingType(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{"@context": "https://schema.org", "name": "Test"}

	findings := reg.Validate(obj)
	if !hasCode(findings, "missing-type") {
		t.Errorf("expected missing-type finding, got %+v", findings)
	}
}

func TestValidate_UnknownType(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{"@type": "CustomThing"}

	findings := reg.Validate(obj)
	if !hasCode(findings, "unknown-type") {
		t.Errorf("expected unknown-type finding, got %+v", findings)
	}
}

func TestValidate_KnownTypeNoIssues(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@context":      "https://schema.org",
		"@type":         "Article",
		"headline":      "Test",
		"datePublished": "2024-01-15",
		"author":        map[string]any{"@type": "Person", "name": "Author"},
	}

	findings := reg.Validate(obj)
	if hasCode(findings, "missing-required-field") {
		t.Errorf("did not expect missing-required-field, got %+v", findings)
	}
}

func TestValidate_MissingRequiredFields(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type": "Article",
	}

	findings := reg.Validate(obj)
	if !hasCode(findings, "missing-required-field") {
		t.Errorf("expected missing-required-field, got %+v", findings)
	}
	f, _ := findByCode(findings, "missing-required-field")
	if !strings.Contains(f.Message, "headline") {
		t.Errorf("expected message to mention 'headline', got %q", f.Message)
	}
}

func TestValidate_InvalidDateField(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":         "Article",
		"headline":      "T",
		"author":        "A",
		"datePublished": "January 15, 2024",
	}

	findings := reg.Validate(obj)
	if !hasCode(findings, "invalid-date-format") {
		t.Errorf("expected invalid-date-format, got %+v", findings)
	}
}

func TestValidate_InvalidURLField(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type": "Organization",
		"url":   "just some text",
	}

	findings := reg.Validate(obj)
	if !hasCode(findings, "invalid-url-field") {
		t.Errorf("expected invalid-url-field, got %+v", findings)
	}
}

func TestValidate_InvalidEnumValue(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":       "Event",
		"name":        "Conf",
		"startDate":   "2024-09-01",
		"location":    "NYC",
		"eventStatus": "EventRunning",
	}

	findings := reg.Validate(obj)
	if !hasCode(findings, "invalid-enum-value") {
		t.Errorf("expected invalid-enum-value, got %+v", findings)
	}
}

func TestValidate_ValidEnumValue(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":       "Event",
		"name":        "Conf",
		"startDate":   "2024-09-01",
		"location":    "NYC",
		"eventStatus": "EventScheduled",
	}

	findings := reg.Validate(obj)
	if hasCode(findings, "invalid-enum-value") {
		t.Errorf("did not expect invalid-enum-value, got %+v", findings)
	}
}

func TestValidate_NestedObjectWrongType(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":         "Article",
		"headline":      "Test",
		"datePublished": "2024-01-15",
		"author":        map[string]any{"@type": "Event", "name": "Wrong"},
	}

	findings := reg.Validate(obj)
	if !hasCode(findings, "invalid-nested-type") {
		t.Errorf("expected invalid-nested-type, got %+v", findings)
	}
}

func TestValidate_NestedObjectCorrectType(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":         "Article",
		"headline":      "Test",
		"datePublished": "2024-01-15",
		"author":        map[string]any{"@type": "Person", "name": "Author"},
	}

	findings := reg.Validate(obj)
	if hasCode(findings, "invalid-nested-type") {
		t.Errorf("did not expect invalid-nested-type, got %+v", findings)
	}
}

func TestValidate_StringValueForObjectProperty(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":         "Article",
		"headline":      "Test",
		"datePublished": "2024-01-15",
		"author":        "John Doe",
	}

	findings := reg.Validate(obj)
	if hasCode(findings, "invalid-nested-type") {
		t.Errorf("string value should be accepted for author, got %+v", findings)
	}
}

func TestValidate_DepthLimit(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":    "Article",
		"headline": "T", "datePublished": "2024-01-15",
		"author": map[string]any{
			"@type": "Person",
			"name":  "A",
			"image": map[string]any{
				"@type":      "ImageObject",
				"contentUrl": "https://example.com/img.jpg",
				"author": map[string]any{
					"@type": "Event", // wrong type but at depth 3 — should not be flagged
					"name":  "deep",
				},
			},
		},
	}

	findings := reg.Validate(obj)
	for _, f := range findings {
		if f.Code == "invalid-nested-type" && strings.Contains(f.Path, "author.image.author") {
			t.Errorf("should not validate beyond depth 3, got finding at %s", f.Path)
		}
	}
}

func TestValidate_Graph(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@context": "https://schema.org",
		"@graph": []any{
			map[string]any{"@type": "Article"},
			map[string]any{"@type": "BreadcrumbList"},
		},
	}

	findings := reg.Validate(obj)
	articleFound := false
	breadcrumbFound := false
	for _, f := range findings {
		if f.Code == "missing-required-field" && strings.Contains(f.Message, "Article") {
			articleFound = true
		}
		if f.Code == "missing-required-field" && strings.Contains(f.Message, "BreadcrumbList") {
			breadcrumbFound = true
		}
	}
	if !articleFound {
		t.Error("expected missing-required-field for Article in @graph")
	}
	if !breadcrumbFound {
		t.Error("expected missing-required-field for BreadcrumbList in @graph")
	}
}

func TestValidate_MultipleTypes(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type": []any{"Article", "BlogPosting"},
	}

	findings := reg.Validate(obj)
	if !hasCode(findings, "missing-required-field") {
		t.Errorf("expected missing-required-field for multi-type, got %+v", findings)
	}
}

func TestValidate_NonStringDate(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":         "Article",
		"headline":      "T",
		"author":        "A",
		"datePublished": 1705276800,
	}

	findings := reg.Validate(obj)
	if !hasCode(findings, "invalid-date-format") {
		t.Errorf("expected invalid-date-format for numeric date, got %+v", findings)
	}
}

func TestValidate_URLFieldAsArray(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":  "Organization",
		"sameAs": []any{"https://twitter.com/ex", "not a url"},
	}

	findings := reg.Validate(obj)
	if !hasCode(findings, "invalid-url-field") {
		t.Errorf("expected invalid-url-field for bad URL in sameAs array, got %+v", findings)
	}
}

func TestValidate_URLFieldAsObject(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":       "Product",
		"name":        "W",
		"description": "D",
		"image":       map[string]any{"@type": "ImageObject", "url": "https://example.com/img.jpg"},
	}

	findings := reg.Validate(obj)
	if hasCode(findings, "invalid-url-field") {
		t.Errorf("nested ImageObject should not trigger invalid-url-field, got %+v", findings)
	}
}
