package schema

import (
	"strings"
	"testing"
)

func TestValidateGoogle_MissingRequired(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":         "Article",
		"headline":      "Test",
		"datePublished": "2024-01-15",
		"author":        "Author",
	}

	findings := reg.ValidateGoogle(obj)
	found := false
	for _, f := range findings {
		if f.Code == "google-missing-required" && strings.Contains(f.Message, "image") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected google-missing-required for 'image', got %+v", findings)
	}
}

func TestValidateGoogle_AllPresent(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":         "Article",
		"headline":      "Test",
		"datePublished": "2024-01-15",
		"author":        "Author",
		"image":         "https://example.com/img.jpg",
	}

	findings := reg.ValidateGoogle(obj)
	if hasCode(findings, "google-missing-required") {
		t.Errorf("did not expect google-missing-required, got %+v", findings)
	}
}

func TestValidateGoogle_MissingRecommended(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type":         "Article",
		"headline":      "Test",
		"datePublished": "2024-01-15",
		"author":        "Author",
		"image":         "https://example.com/img.jpg",
	}

	findings := reg.ValidateGoogle(obj)
	found := false
	for _, f := range findings {
		if f.Code == "google-missing-recommended" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected google-missing-recommended, got %+v", findings)
	}
}

func TestValidateGoogle_NotEligible(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type": "Organization",
		"name":  "Acme",
	}

	findings := reg.ValidateGoogle(obj)
	if !hasCode(findings, "not-google-eligible") {
		t.Errorf("expected not-google-eligible for Organization, got %+v", findings)
	}
}

func TestValidateGoogle_UnknownType(t *testing.T) {
	reg := testRegistry(t)
	obj := map[string]any{
		"@type": "CustomThing",
	}

	findings := reg.ValidateGoogle(obj)
	if hasCode(findings, "google-missing-required") || hasCode(findings, "not-google-eligible") {
		t.Errorf("unknown type should not produce Google findings, got %+v", findings)
	}
}
