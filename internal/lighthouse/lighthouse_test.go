package lighthouse

import (
	"testing"

	"github.com/meysam81/scry/core/model"
	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/logger"
)

func TestScoreToIssues(t *testing.T) {
	tests := []struct {
		name       string
		result     *model.LighthouseResult
		wantCount  int
		wantChecks []string
		wantSevs   []model.Severity
	}{
		{
			name: "perfect scores produce no issues",
			result: &model.LighthouseResult{
				URL:                "https://example.com",
				PerformanceScore:   100,
				AccessibilityScore: 100,
				SEOScore:           100,
			},
			wantCount: 0,
		},
		{
			name: "performance exactly 90 produces no issue",
			result: &model.LighthouseResult{
				URL:                "https://example.com",
				PerformanceScore:   90,
				AccessibilityScore: 95,
				SEOScore:           95,
			},
			wantCount: 0,
		},
		{
			name: "performance below 50 is critical",
			result: &model.LighthouseResult{
				URL:                "https://example.com",
				PerformanceScore:   30,
				AccessibilityScore: 95,
				SEOScore:           95,
			},
			wantCount:  1,
			wantChecks: []string{CheckPerformance},
			wantSevs:   []model.Severity{model.SeverityCritical},
		},
		{
			name: "performance exactly 50 is warning",
			result: &model.LighthouseResult{
				URL:                "https://example.com",
				PerformanceScore:   50,
				AccessibilityScore: 95,
				SEOScore:           95,
			},
			wantCount:  1,
			wantChecks: []string{CheckPerformance},
			wantSevs:   []model.Severity{model.SeverityWarning},
		},
		{
			name: "performance 89 is warning",
			result: &model.LighthouseResult{
				URL:                "https://example.com",
				PerformanceScore:   89,
				AccessibilityScore: 95,
				SEOScore:           95,
			},
			wantCount:  1,
			wantChecks: []string{CheckPerformance},
			wantSevs:   []model.Severity{model.SeverityWarning},
		},
		{
			name: "accessibility below 90 is warning",
			result: &model.LighthouseResult{
				URL:                "https://example.com",
				PerformanceScore:   95,
				AccessibilityScore: 80,
				SEOScore:           95,
			},
			wantCount:  1,
			wantChecks: []string{CheckAccessibility},
			wantSevs:   []model.Severity{model.SeverityWarning},
		},
		{
			name: "seo below 90 is warning",
			result: &model.LighthouseResult{
				URL:                "https://example.com",
				PerformanceScore:   95,
				AccessibilityScore: 95,
				SEOScore:           70,
			},
			wantCount:  1,
			wantChecks: []string{CheckSEO},
			wantSevs:   []model.Severity{model.SeverityWarning},
		},
		{
			name: "all scores bad produces multiple issues",
			result: &model.LighthouseResult{
				URL:                "https://example.com",
				PerformanceScore:   20,
				AccessibilityScore: 50,
				SEOScore:           40,
			},
			wantCount:  3,
			wantChecks: []string{CheckPerformance, CheckAccessibility, CheckSEO},
			wantSevs:   []model.Severity{model.SeverityCritical, model.SeverityWarning, model.SeverityWarning},
		},
		{
			name: "performance 0 is critical",
			result: &model.LighthouseResult{
				URL:                "https://example.com",
				PerformanceScore:   0,
				AccessibilityScore: 90,
				SEOScore:           90,
			},
			wantCount:  1,
			wantChecks: []string{CheckPerformance},
			wantSevs:   []model.Severity{model.SeverityCritical},
		},
		{
			name: "accessibility exactly 90 produces no issue",
			result: &model.LighthouseResult{
				URL:                "https://example.com",
				PerformanceScore:   95,
				AccessibilityScore: 90,
				SEOScore:           95,
			},
			wantCount: 0,
		},
		{
			name: "seo exactly 90 produces no issue",
			result: &model.LighthouseResult{
				URL:                "https://example.com",
				PerformanceScore:   95,
				AccessibilityScore: 95,
				SEOScore:           90,
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ScoreToIssues(tt.result)

			if len(issues) != tt.wantCount {
				t.Fatalf("got %d issues, want %d: %+v", len(issues), tt.wantCount, issues)
			}

			for i, issue := range issues {
				if i < len(tt.wantChecks) && issue.CheckName != tt.wantChecks[i] {
					t.Errorf("issue[%d] check = %q, want %q", i, issue.CheckName, tt.wantChecks[i])
				}
				if i < len(tt.wantSevs) && issue.Severity != tt.wantSevs[i] {
					t.Errorf("issue[%d] severity = %q, want %q", i, issue.Severity, tt.wantSevs[i])
				}
				if issue.URL != tt.result.URL {
					t.Errorf("issue[%d] url = %q, want %q", i, issue.URL, tt.result.URL)
				}
			}
		})
	}
}

func TestNewRunner_PSI(t *testing.T) {
	cfg := &config.Config{
		LighthouseMode: "psi",
		PSIAPIKey:      "test-key",
		PSIStrategy:    "mobile",
	}

	runner, err := NewRunner(cfg, logger.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if runner == nil {
		t.Fatal("expected runner, got nil")
	}

	// Check that it's a PSIClient.
	if _, ok := runner.(*PSIClient); !ok {
		t.Errorf("expected *PSIClient, got %T", runner)
	}
}

func TestNewRunner_Browserless(t *testing.T) {
	cfg := &config.Config{
		LighthouseMode: "browserless",
		BrowserlessURL: "http://localhost:3000",
	}

	runner, err := NewRunner(cfg, logger.Nop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if runner == nil {
		t.Fatal("expected runner, got nil")
	}

	// Check that it's a BrowserlessClient.
	if _, ok := runner.(*BrowserlessClient); !ok {
		t.Errorf("expected *BrowserlessClient, got %T", runner)
	}
}

func TestNewRunner_UnknownMode(t *testing.T) {
	cfg := &config.Config{
		LighthouseMode: "unknown",
	}

	runner, err := NewRunner(cfg, logger.Nop())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if runner != nil {
		t.Errorf("expected nil runner, got %T", runner)
	}
}
