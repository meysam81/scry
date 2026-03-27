package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEmbedded(t *testing.T) {
	reg, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error: %v", err)
	}
	if reg.Version == "" {
		t.Error("expected non-empty version")
	}
	if len(reg.Types) == 0 {
		t.Error("expected at least one type")
	}

	// Spot-check a known type.
	article, ok := reg.Lookup("Article")
	if !ok {
		t.Fatal("expected Article type in registry")
	}
	if !article.GoogleEligible {
		t.Error("expected Article to be Google eligible")
	}
	if len(article.RequiredFields) == 0 {
		t.Error("expected Article to have required fields")
	}
}

func TestLookup_Unknown(t *testing.T) {
	reg, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error: %v", err)
	}
	_, ok := reg.Lookup("NonExistentType")
	if ok {
		t.Error("expected Lookup to return false for unknown type")
	}
}

func TestIsKnownType(t *testing.T) {
	reg, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error: %v", err)
	}
	if !reg.IsKnownType("Article") {
		t.Error("expected Article to be known")
	}
	if reg.IsKnownType("FooBarBaz") {
		t.Error("expected FooBarBaz to be unknown")
	}
}

func TestLoadFromFile_Override(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schemas.json")

	data := []byte(`{
		"version": "test-override",
		"types": {
			"CustomType": {
				"name": "CustomType",
				"required_fields": ["foo"],
				"properties": {},
				"google_eligible": false
			}
		}
	}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	reg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error: %v", err)
	}
	if reg.Version != "test-override" {
		t.Errorf("expected version %q, got %q", "test-override", reg.Version)
	}
	if _, ok := reg.Lookup("CustomType"); !ok {
		t.Error("expected CustomType in overridden registry")
	}
}

func TestLoadFromFile_Invalid_Fallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schemas.json")

	if err := os.WriteFile(path, []byte("{invalid json"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, err := LoadFromFile(path)
	if err == nil {
		t.Error("expected error for invalid JSON file")
	}
}

func TestLoad_FallbackWhenNoFile(t *testing.T) {
	reg := Load("/nonexistent/path/schemas.json")
	if reg.Version == "" {
		t.Error("expected fallback to embedded with non-empty version")
	}
	if len(reg.Types) == 0 {
		t.Error("expected fallback to have types")
	}
}

func TestInheritance_ChildGetsParentProperties(t *testing.T) {
	reg, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error: %v", err)
	}

	// BlogPosting has parent=Article; its properties in schemas.json are empty,
	// but inheritance should give it Article's properties.
	bp, ok := reg.Lookup("BlogPosting")
	if !ok {
		t.Fatal("expected BlogPosting in registry")
	}
	if _, has := bp.Properties["headline"]; !has {
		t.Error("expected BlogPosting to inherit 'headline' property from Article")
	}
	if _, has := bp.Properties["datePublished"]; !has {
		t.Error("expected BlogPosting to inherit 'datePublished' property from Article")
	}
	if _, has := bp.Properties["author"]; !has {
		t.Error("expected BlogPosting to inherit 'author' property from Article")
	}
}

func TestInheritance_ChildOverridesParent(t *testing.T) {
	reg, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() error: %v", err)
	}

	// WebApplication has parent=SoftwareApplication and adds browserRequirements.
	wa, ok := reg.Lookup("WebApplication")
	if !ok {
		t.Fatal("expected WebApplication in registry")
	}
	// Should have parent's property.
	if _, has := wa.Properties["operatingSystem"]; !has {
		t.Error("expected WebApplication to inherit 'operatingSystem' from SoftwareApplication")
	}
	// Should have its own property.
	if _, has := wa.Properties["browserRequirements"]; !has {
		t.Error("expected WebApplication to have 'browserRequirements'")
	}
}
