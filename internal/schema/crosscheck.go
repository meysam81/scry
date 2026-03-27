package schema

import (
	"fmt"
	"slices"
	"strings"
)

// CrossCheck validates relationships across multiple JSON-LD objects on the same page.
func CrossCheck(objects []map[string]any) []Finding {
	var findings []Finding

	findings = append(findings, checkDuplicateTypes(objects)...)
	findings = append(findings, checkBreadcrumbPositions(objects)...)
	findings = append(findings, checkSearchAction(objects)...)

	return findings
}

// checkDuplicateTypes reports when multiple objects declare the same @type.
func checkDuplicateTypes(objects []map[string]any) []Finding {
	typeCounts := make(map[string]int)
	for _, obj := range objects {
		types := extractTypesGeneric(obj["@type"])
		for _, t := range types {
			typeCounts[t]++
		}
	}

	// Collect duplicates and sort for deterministic output.
	var dupes []string
	for typeName, count := range typeCounts {
		if count > 1 {
			dupes = append(dupes, typeName)
		}
	}
	slices.Sort(dupes)

	var findings []Finding
	for _, typeName := range dupes {
		findings = append(findings, Finding{
			Code:     "duplicate-type",
			Severity: "info",
			Message:  fmt.Sprintf("multiple JSON-LD blocks declare @type %q (%d occurrences)", typeName, typeCounts[typeName]),
		})
	}
	return findings
}

// checkBreadcrumbPositions validates that BreadcrumbList items have sequential positions starting at 1.
func checkBreadcrumbPositions(objects []map[string]any) []Finding {
	var findings []Finding

	for _, obj := range objects {
		types := extractTypesGeneric(obj["@type"])
		if !slices.Contains(types, "BreadcrumbList") {
			continue
		}

		items, ok := obj["itemListElement"].([]any)
		if !ok {
			continue
		}

		for i, item := range items {
			itemObj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			pos, ok := itemObj["position"]
			if !ok {
				continue
			}
			posNum, ok := pos.(float64) // JSON numbers are float64
			if !ok {
				continue
			}
			expected := float64(i + 1)
			if posNum != expected {
				findings = append(findings, Finding{
					Code:     "breadcrumb-positions",
					Severity: "warning",
					Message:  fmt.Sprintf("BreadcrumbList item at index %d has position %.0f; expected %d", i, posNum, i+1),
				})
				break // One finding per breadcrumb is enough.
			}
		}
	}

	return findings
}

// checkSearchAction validates WebSite SearchAction target templates.
func checkSearchAction(objects []map[string]any) []Finding {
	var findings []Finding

	for _, obj := range objects {
		types := extractTypesGeneric(obj["@type"])
		if !slices.Contains(types, "WebSite") {
			continue
		}

		action, ok := obj["potentialAction"]
		if !ok {
			continue
		}

		actionObj, ok := action.(map[string]any)
		if !ok {
			continue
		}

		actionTypes := extractTypesGeneric(actionObj["@type"])
		if !slices.Contains(actionTypes, "SearchAction") {
			continue
		}

		target, ok := actionObj["target"]
		if !ok {
			continue
		}

		// target can be a string URL or an EntryPoint object with a urlTemplate.
		var targetStr string
		switch t := target.(type) {
		case string:
			targetStr = t
		case map[string]any:
			if tpl, ok := t["urlTemplate"]; ok {
				if s, ok := tpl.(string); ok {
					targetStr = s
				}
			}
		}
		if targetStr == "" {
			continue
		}

		if !strings.Contains(targetStr, "{search_term_string}") {
			findings = append(findings, Finding{
				Code:     "search-action-template",
				Severity: "warning",
				Message:  "WebSite SearchAction target is missing {search_term_string} template variable",
			})
		}
	}

	return findings
}
