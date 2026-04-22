package schema

import (
	"slices"
	"strings"
	"time"
)

// ValidateDate reports whether v looks like a valid ISO 8601 date string.
// It accepts YYYY-MM-DD optionally followed by a time component.
func ValidateDate(v string) bool {
	v = strings.TrimSpace(v)
	if len(v) < 10 {
		return false
	}
	// Parse the date portion (first 10 chars) to reject impossible dates
	// like "9999-99-99".
	_, err := time.Parse(time.DateOnly, v[:10])
	return err == nil
}

// ValidateURL reports whether v looks like a valid URL (http, https, or
// root-relative path).
func ValidateURL(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	return strings.HasPrefix(v, "http://") ||
		strings.HasPrefix(v, "https://") ||
		strings.HasPrefix(v, "/")
}

// ValidateEnum reports whether v is one of the allowed values.
// Comparison is case-sensitive per Schema.org convention.
func ValidateEnum(v string, allowed []string) bool {
	return slices.Contains(allowed, v)
}
