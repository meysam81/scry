package schema

import (
	"fmt"
	"slices"
	"strings"
)

const maxDepth = 3

// Validate validates a JSON-LD object (already parsed from JSON) against
// the registry's type definitions. It handles @graph, arrays of @type,
// and recursive nested objects.
func (r *Registry) Validate(obj map[string]any) []Finding {
	return r.validateObject(obj, "", 0)
}

// validateObject validates a single JSON-LD object at the given nesting depth.
func (r *Registry) validateObject(obj map[string]any, path string, depth int) []Finding {
	if depth >= maxDepth {
		return nil
	}

	// Handle @graph: recurse into each item.
	if graph, ok := obj["@graph"]; ok {
		if graphArr, ok := graph.([]any); ok {
			var findings []Finding
			for _, item := range graphArr {
				if itemObj, ok := item.(map[string]any); ok {
					findings = append(findings, r.validateObject(itemObj, path, depth)...)
				}
			}
			return findings
		}
	}

	typeVal, hasType := obj["@type"]
	if !hasType {
		return []Finding{{
			Code:     "missing-type",
			Severity: "warning",
			Message:  "JSON-LD block has no @type field",
			Path:     joinPath(path, "@type"),
		}}
	}

	types := extractTypesGeneric(typeVal)
	var findings []Finding

	for _, typeName := range types {
		td, known := r.Lookup(typeName)
		if !known {
			findings = append(findings, Finding{
				Code:     "unknown-type",
				Severity: "info",
				Message:  fmt.Sprintf("JSON-LD @type %q is not a commonly recognised Schema.org type", typeName),
				Path:     joinPath(path, "@type"),
			})
			continue
		}

		// Required field check.
		findings = append(findings, r.checkRequiredFields(obj, td, typeName, path)...)

		// Property validation: date, URL, enum, nested types.
		findings = append(findings, r.checkProperties(obj, td, path, depth)...)
	}

	return findings
}

// checkRequiredFields checks whether all required fields for a type are present.
// Emits one finding per missing field for actionable, field-level granularity.
func (r *Registry) checkRequiredFields(obj map[string]any, td *TypeDef, typeName, path string) []Finding {
	var findings []Finding
	for _, field := range td.RequiredFields {
		if _, exists := obj[field]; !exists {
			findings = append(findings, Finding{
				Code:     "missing-required-field",
				Severity: "warning",
				Message:  fmt.Sprintf("%s is missing required field %q", typeName, field),
				Path:     joinPath(path, field),
			})
		}
	}
	return findings
}

// checkProperties validates individual property values against their definitions.
func (r *Registry) checkProperties(obj map[string]any, td *TypeDef, path string, depth int) []Finding {
	var findings []Finding

	for propName, propDef := range td.Properties {
		val, exists := obj[propName]
		if !exists {
			continue
		}

		propPath := joinPath(path, propName)

		// Date validation.
		if propDef.IsDate {
			findings = append(findings, r.checkDateValue(propName, val, propPath)...)
		}

		// URL validation.
		if propDef.IsURL {
			findings = append(findings, r.checkURLValue(propName, val, propPath)...)
		}

		// Enum validation.
		if len(propDef.EnumValues) > 0 {
			if str, ok := val.(string); ok {
				if !ValidateEnum(str, propDef.EnumValues) {
					findings = append(findings, Finding{
						Code:     "invalid-enum-value",
						Severity: "warning",
						Message:  fmt.Sprintf("JSON-LD field %q has value %q; expected one of: %s", propName, str, strings.Join(propDef.EnumValues, ", ")),
						Path:     propPath,
					})
				}
			}
		}

		// Nested type validation (only for properties with ExpectedTypes that include Schema.org types).
		if len(propDef.ExpectedTypes) > 0 {
			findings = append(findings, r.checkNestedType(propName, val, propDef, propPath, depth)...)
		}
	}

	return findings
}

// checkDateValue validates a date property value.
func (r *Registry) checkDateValue(field string, val any, path string) []Finding {
	str, ok := val.(string)
	if !ok {
		return []Finding{{
			Code:     "invalid-date-format",
			Severity: "warning",
			Message:  fmt.Sprintf("JSON-LD field %q has a non-string value; expected ISO 8601 date", field),
			Path:     path,
		}}
	}
	if !ValidateDate(str) {
		return []Finding{{
			Code:     "invalid-date-format",
			Severity: "warning",
			Message:  fmt.Sprintf("JSON-LD field %q has invalid date value %q; expected ISO 8601 format (YYYY-MM-DD)", field, str),
			Path:     path,
		}}
	}
	return nil
}

// checkURLValue validates a URL property value. Handles strings, arrays, and objects.
func (r *Registry) checkURLValue(field string, val any, path string) []Finding {
	switch v := val.(type) {
	case string:
		if !ValidateURL(v) {
			return []Finding{{
				Code:     "invalid-url-field",
				Severity: "warning",
				Message:  fmt.Sprintf("JSON-LD field %q has invalid URL value %q; expected http://, https://, or /", field, v),
				Path:     path,
			}}
		}
	case []any:
		var findings []Finding
		for _, item := range v {
			if s, ok := item.(string); ok {
				if !ValidateURL(s) {
					findings = append(findings, Finding{
						Code:     "invalid-url-field",
						Severity: "warning",
						Message:  fmt.Sprintf("JSON-LD field %q has invalid URL value %q; expected http://, https://, or /", field, s),
						Path:     path,
					})
				}
			}
		}
		return findings
	case map[string]any:
		// Nested object (e.g. ImageObject) — skip URL check, will be validated as nested type.
	default:
		return []Finding{{
			Code:     "invalid-url-field",
			Severity: "warning",
			Message:  fmt.Sprintf("JSON-LD field %q has a non-string/non-array value; expected a URL", field),
			Path:     path,
		}}
	}
	return nil
}

// checkNestedType validates that a nested object's @type matches one of the expected types.
func (r *Registry) checkNestedType(field string, val any, propDef PropertyDef, path string, depth int) []Finding {
	obj, ok := val.(map[string]any)
	if !ok {
		// String or array values are fine — text is always acceptable.
		return nil
	}

	typeVal, hasType := obj["@type"]
	if !hasType {
		// Nested object without @type — not flagged here (missing-type is for root objects).
		return nil
	}

	types := extractTypesGeneric(typeVal)
	var matched bool
	for _, t := range types {
		if slices.Contains(propDef.ExpectedTypes, t) {
			matched = true
			break
		}
	}

	var findings []Finding
	if !matched && len(types) > 0 {
		findings = append(findings, Finding{
			Code:     "invalid-nested-type",
			Severity: "warning",
			Message:  fmt.Sprintf("JSON-LD field %q has nested @type %q; expected one of: %s", field, strings.Join(types, ", "), strings.Join(propDef.ExpectedTypes, ", ")),
			Path:     path + ".@type",
		})
	}

	// Recurse into the nested object for its own validation.
	findings = append(findings, r.validateObject(obj, path, depth+1)...)

	return findings
}

// extractTypesGeneric normalises @type to a slice of strings.
func extractTypesGeneric(v any) []string {
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

// joinPath creates a dotted JSON path.
func joinPath(base, field string) string {
	if base == "" {
		return field
	}
	return base + "." + field
}
