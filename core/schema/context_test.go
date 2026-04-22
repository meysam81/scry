package schema

import "testing"

func TestValidateContext_Missing(t *testing.T) {
	obj := map[string]any{"@type": "Article"}
	findings := ValidateContext(obj)
	if !hasCode(findings, "missing-context") {
		t.Errorf("expected missing-context, got %+v", findings)
	}
}

func TestValidateContext_ValidHTTPS(t *testing.T) {
	obj := map[string]any{"@context": "https://schema.org", "@type": "Article"}
	findings := ValidateContext(obj)
	if hasCode(findings, "missing-context") || hasCode(findings, "wrong-context") {
		t.Errorf("did not expect context issues, got %+v", findings)
	}
}

func TestValidateContext_ValidHTTP(t *testing.T) {
	obj := map[string]any{"@context": "http://schema.org", "@type": "Article"}
	findings := ValidateContext(obj)
	if hasCode(findings, "wrong-context") {
		t.Errorf("http://schema.org should be valid, got %+v", findings)
	}
}

func TestValidateContext_ValidWithTrailingSlash(t *testing.T) {
	obj := map[string]any{"@context": "https://schema.org/", "@type": "Article"}
	findings := ValidateContext(obj)
	if hasCode(findings, "wrong-context") {
		t.Errorf("trailing slash should be valid, got %+v", findings)
	}
}

func TestValidateContext_Wrong(t *testing.T) {
	obj := map[string]any{"@context": "https://example.com/schema", "@type": "Article"}
	findings := ValidateContext(obj)
	if !hasCode(findings, "wrong-context") {
		t.Errorf("expected wrong-context, got %+v", findings)
	}
}

func TestValidateContext_ObjectContext(t *testing.T) {
	obj := map[string]any{
		"@context": map[string]any{"@vocab": "https://schema.org/"},
		"@type":    "Article",
	}
	findings := ValidateContext(obj)
	if hasCode(findings, "wrong-context") {
		t.Errorf("object context with schema.org @vocab should be valid, got %+v", findings)
	}
}

func TestValidateContext_ObjectContextWrong(t *testing.T) {
	obj := map[string]any{
		"@context": map[string]any{"@vocab": "https://example.com/"},
		"@type":    "Article",
	}
	findings := ValidateContext(obj)
	if !hasCode(findings, "wrong-context") {
		t.Errorf("expected wrong-context for non-schema.org @vocab, got %+v", findings)
	}
}

func TestValidateContext_ArrayContextValid(t *testing.T) {
	obj := map[string]any{
		"@context": []any{"https://schema.org", map[string]any{"@language": "en"}},
		"@type":    "Article",
	}
	findings := ValidateContext(obj)
	if hasCode(findings, "wrong-context") {
		t.Errorf("array context with schema.org should be valid, got %+v", findings)
	}
}

func TestValidateContext_ArrayContextWrong(t *testing.T) {
	obj := map[string]any{
		"@context": []any{"https://example.com", map[string]any{"@language": "en"}},
		"@type":    "Article",
	}
	findings := ValidateContext(obj)
	if !hasCode(findings, "wrong-context") {
		t.Errorf("expected wrong-context for array without schema.org, got %+v", findings)
	}
}
