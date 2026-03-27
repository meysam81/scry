// Package schema provides data-driven JSON-LD structured data validation
// against Schema.org type definitions and Google Rich Results requirements.
package schema

// TypeDef describes a Schema.org type's validation rules.
type TypeDef struct {
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	Parent            string                 `json:"parent,omitempty"`
	Properties        map[string]PropertyDef `json:"properties"`
	RequiredFields    []string               `json:"required_fields"`
	GoogleRequired    []string               `json:"google_required,omitempty"`
	GoogleRecommended []string               `json:"google_recommended,omitempty"`
	GoogleEligible    bool                   `json:"google_eligible"`
}

// PropertyDef describes a single property's validation constraints.
type PropertyDef struct {
	Description   string   `json:"description,omitempty"`
	ExpectedTypes []string `json:"expected_types"`
	IsDate        bool     `json:"is_date,omitempty"`
	IsURL         bool     `json:"is_url,omitempty"`
	EnumValues    []string `json:"enum_values,omitempty"`
}

// Finding is a schema validation result, decoupled from the audit model.
type Finding struct {
	Code     string // e.g. "missing-required-field", "invalid-nested-type"
	Severity string // "critical", "warning", "info"
	Message  string
	Path     string // JSON path e.g. "author.@type"
}
