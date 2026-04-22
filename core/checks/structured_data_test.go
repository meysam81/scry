package checks

import (
	"context"
	"strings"
	"testing"

	"github.com/meysam81/scry/core/model"
)

func TestStructuredDataChecker_Check(t *testing.T) {
	checker := NewStructuredDataChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantCheck  string
		wantSev    model.Severity
		wantIssue  bool
		wantSubstr string
	}{
		{
			name:      "missing json-ld",
			html:      `<html><head></head><body></body></html>`,
			wantCheck: "structured-data/missing-json-ld",
			wantSev:   model.SeverityInfo,
			wantIssue: true,
		},
		{
			name:       "malformed json-ld",
			html:       `<html><head><script type="application/ld+json">{invalid json</script></head><body></body></html>`,
			wantCheck:  "structured-data/malformed-json-ld",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "invalid",
		},
		{
			name:      "valid json-ld no issue",
			html:      `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"WebPage"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "non-html page skipped",
			html:      "",
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			if tt.name == "non-html page skipped" {
				page.ContentType = "application/json"
			}

			issues := checker.Check(ctx, page)

			if !tt.wantIssue {
				if len(issues) > 0 {
					// Check none of the issues match the unwanted check.
					for _, iss := range issues {
						if tt.wantCheck != "" && iss.CheckName == tt.wantCheck {
							t.Fatalf("did not expect issue %s", tt.wantCheck)
						}
					}
				}
				return
			}

			found := false
			for _, iss := range issues {
				if iss.CheckName == tt.wantCheck {
					found = true
					if iss.Severity != tt.wantSev {
						t.Errorf("expected severity %s, got %s", tt.wantSev, iss.Severity)
					}
					if tt.wantSubstr != "" && !strings.Contains(strings.ToLower(iss.Message), tt.wantSubstr) {
						t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
					}
				}
			}
			if !found {
				t.Errorf("expected issue %s not found in %+v", tt.wantCheck, issues)
			}
		})
	}
}
