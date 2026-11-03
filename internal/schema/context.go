package schema

import (
	"slices"
	"strings"
)

const codeWrongContext = "wrong-context"

// validContextPrefixes are the accepted schema.org context URL prefixes.
var validContextPrefixes = []string{
	"https://schema.org",
	"http://schema.org",
}

// ValidateContext checks the @context field of a JSON-LD object.
func ValidateContext(obj map[string]any) []Finding {
	ctxVal, hasCtx := obj[keyContext]
	if !hasCtx {
		return []Finding{{
			Code:     "missing-context",
			Severity: severityInfo,
			Message:  "JSON-LD block has no @context",
			Path:     keyContext,
		}}
	}

	switch ctx := ctxVal.(type) {
	case string:
		if !isSchemaOrgURL(ctx) {
			return []Finding{{
				Code:     codeWrongContext,
				Severity: severityWarning,
				Message:  "@context does not point to schema.org: " + ctx,
				Path:     keyContext,
			}}
		}
	case map[string]any:
		// Object context — check @vocab.
		if vocab, ok := ctx["@vocab"]; ok {
			if vocabStr, ok := vocab.(string); ok {
				if !isSchemaOrgURL(vocabStr) {
					return []Finding{{
						Code:     codeWrongContext,
						Severity: severityWarning,
						Message:  "@context @vocab does not point to schema.org: " + vocabStr,
						Path:     "@context.@vocab",
					}}
				}
			}
		}
	case []any:
		// Array context — check each string entry for schema.org.
		found := false
		for _, item := range ctx {
			if s, ok := item.(string); ok && isSchemaOrgURL(s) {
				found = true
				break
			}
			if m, ok := item.(map[string]any); ok {
				if vocab, ok := m["@vocab"]; ok {
					if vocabStr, ok := vocab.(string); ok && isSchemaOrgURL(vocabStr) {
						found = true
						break
					}
				}
			}
		}
		if !found {
			return []Finding{{
				Code:     codeWrongContext,
				Severity: severityWarning,
				Message:  "@context array does not contain a schema.org reference",
				Path:     keyContext,
			}}
		}
	}

	return nil
}

// isSchemaOrgURL reports whether u points to schema.org.
func isSchemaOrgURL(u string) bool {
	u = strings.TrimRight(u, "/")
	return slices.Contains(validContextPrefixes, u)
}
