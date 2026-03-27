package schema

import (
	"fmt"
	"strings"
)

// ValidateGoogle checks a JSON-LD object against Google Rich Results requirements.
// It only produces findings for types known to the registry.
func (r *Registry) ValidateGoogle(obj map[string]any) []Finding {
	typeVal, hasType := obj["@type"]
	if !hasType {
		return nil
	}

	types := extractTypesGeneric(typeVal)
	var findings []Finding

	for _, typeName := range types {
		td, known := r.Lookup(typeName)
		if !known {
			continue
		}

		if !td.GoogleEligible {
			findings = append(findings, Finding{
				Code:     "not-google-eligible",
				Severity: "info",
				Message:  fmt.Sprintf("%s is not eligible for Google Rich Results", typeName),
				Path:     "@type",
			})
			continue
		}

		// Check Google-required fields.
		var missingRequired []string
		for _, field := range td.GoogleRequired {
			if _, exists := obj[field]; !exists {
				missingRequired = append(missingRequired, field)
			}
		}
		if len(missingRequired) > 0 {
			findings = append(findings, Finding{
				Code:     "google-missing-required",
				Severity: "warning",
				Message:  fmt.Sprintf("Google Rich Results: %s is missing required fields: %s", typeName, strings.Join(missingRequired, ", ")),
				Path:     "@type",
			})
		}

		// Check Google-recommended fields.
		var missingRecommended []string
		for _, field := range td.GoogleRecommended {
			if _, exists := obj[field]; !exists {
				missingRecommended = append(missingRecommended, field)
			}
		}
		if len(missingRecommended) > 0 {
			findings = append(findings, Finding{
				Code:     "google-missing-recommended",
				Severity: "info",
				Message:  fmt.Sprintf("Google Rich Results: %s is missing recommended fields: %s", typeName, strings.Join(missingRecommended, ", ")),
				Path:     "@type",
			})
		}
	}

	return findings
}
