package schema

import "testing"

func TestValidateDate(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid YYYY-MM-DD", "2024-01-15", true},
		{"valid with time", "2024-01-15T10:30:00Z", true},
		{"valid with timezone", "2024-01-15T10:30:00+02:00", true},
		{"plain text", "January 15, 2024", false},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"slash format", "01/15/2024", false},
		{"word", "tomorrow", false},
		{"impossible month", "9999-99-99", false},
		{"impossible day", "2024-02-30", false},
		{"month 13", "2024-13-01", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateDate(tt.value); got != tt.valid {
				t.Errorf("ValidateDate(%q) = %v, want %v", tt.value, got, tt.valid)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"https URL", "https://example.com", true},
		{"http URL", "http://example.com", true},
		{"relative path", "/about", true},
		{"plain text", "just some text", false},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"mailto", "mailto:test@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateURL(tt.value); got != tt.valid {
				t.Errorf("ValidateURL(%q) = %v, want %v", tt.value, got, tt.valid)
			}
		})
	}
}

func TestValidateEnum(t *testing.T) {
	allowed := []string{"EventScheduled", "EventCancelled", "EventPostponed"}

	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid value", "EventScheduled", true},
		{"another valid", "EventCancelled", true},
		{"invalid value", "EventRunning", false},
		{"empty string", "", false},
		{"case mismatch", "eventscheduled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateEnum(tt.value, allowed); got != tt.valid {
				t.Errorf("ValidateEnum(%q, ...) = %v, want %v", tt.value, got, tt.valid)
			}
		})
	}
}
