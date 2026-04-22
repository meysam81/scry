package schema

import (
	"encoding/json"
	"fmt"
	"os"
)

// Registry holds all known Schema.org type definitions.
type Registry struct {
	Version string              `json:"version"`
	Types   map[string]*TypeDef `json:"types"`
}

// LoadEmbedded loads the registry from the embedded schemas.json.
func LoadEmbedded() (*Registry, error) {
	var reg Registry
	if err := json.Unmarshal(embeddedSchemas, &reg); err != nil {
		return nil, fmt.Errorf("parse embedded schemas: %w", err)
	}
	reg.resolveInheritance()
	return &reg, nil
}

// LoadFromFile loads the registry from a local JSON file.
func LoadFromFile(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema file %s: %w", path, err)
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse schema file %s: %w", path, err)
	}
	reg.resolveInheritance()
	return &reg, nil
}

// resolveInheritance merges parent type properties into child types so that
// schemas.json can stay DRY — children only need to declare their own
// properties while inheriting everything from their parent chain.
func (r *Registry) resolveInheritance() {
	resolved := make(map[string]bool)
	for name := range r.Types {
		r.resolveType(name, resolved)
	}
}

func (r *Registry) resolveType(name string, resolved map[string]bool) {
	if resolved[name] {
		return
	}
	td := r.Types[name]
	if td == nil {
		resolved[name] = true
		return
	}
	// Resolve parent first.
	if td.Parent != "" {
		r.resolveType(td.Parent, resolved)
		if parent, ok := r.Types[td.Parent]; ok {
			// Merge parent properties that the child doesn't override.
			if td.Properties == nil {
				td.Properties = make(map[string]PropertyDef)
			}
			for k, v := range parent.Properties {
				if _, exists := td.Properties[k]; !exists {
					td.Properties[k] = v
				}
			}
		}
	}
	resolved[name] = true
}

// Load returns a Registry, trying the local file first and falling
// back to the embedded data. It never returns nil.
func Load(localPath string) *Registry {
	if localPath != "" {
		if reg, err := LoadFromFile(localPath); err == nil {
			return reg
		}
		// Fall through to embedded on any error.
	}

	reg, err := LoadEmbedded()
	if err != nil {
		// Should never happen — embedded data is compiled in.
		return &Registry{Types: make(map[string]*TypeDef)}
	}
	return reg
}

// Lookup returns the TypeDef for a type name, or false if not found.
func (r *Registry) Lookup(typeName string) (*TypeDef, bool) {
	td, ok := r.Types[typeName]
	return td, ok
}

// IsKnownType reports whether the type name exists in the registry.
func (r *Registry) IsKnownType(typeName string) bool {
	_, ok := r.Types[typeName]
	return ok
}
