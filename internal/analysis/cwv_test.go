package analysis

import (
	"strings"
	"testing"

	"github.com/meysam81/scry/core/model"
)

func TestAggregateCWV_Empty(t *testing.T) {
	summary := AggregateCWV(nil)
	if summary != nil {
		t.Fatalf("expected nil for empty results, got %+v", summary)
	}

	summary = AggregateCWV([]model.LighthouseResult{})
	if summary != nil {
		t.Fatalf("expected nil for empty slice, got %+v", summary)
	}
}

func TestAggregateCWV_SinglePage(t *testing.T) {
	results := []model.LighthouseResult{
		{
			URL:                "https://example.com",
			PerformanceScore:   85,
			AccessibilityScore: 92,
			BestPracticesScore: 100,
			SEOScore:           90,
		},
	}

	summary := AggregateCWV(results)
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}

	if summary.PageCount != 1 {
		t.Errorf("expected page count 1, got %d", summary.PageCount)
	}
	if summary.AvgPerformance != 85 {
		t.Errorf("expected avg performance 85, got %f", summary.AvgPerformance)
	}
	if summary.AvgAccessibility != 92 {
		t.Errorf("expected avg accessibility 92, got %f", summary.AvgAccessibility)
	}
	if summary.AvgBestPractices != 100 {
		t.Errorf("expected avg best practices 100, got %f", summary.AvgBestPractices)
	}
	if summary.AvgSEO != 90 {
		t.Errorf("expected avg SEO 90, got %f", summary.AvgSEO)
	}
	// 85 < 90, so not passing.
	if summary.PassRate != 0 {
		t.Errorf("expected pass rate 0%%, got %f", summary.PassRate)
	}
	if len(summary.WorstPerformance) != 1 {
		t.Errorf("expected 1 worst performance entry, got %d", len(summary.WorstPerformance))
	}
}

func TestAggregateCWV_MultiplePages(t *testing.T) {
	results := []model.LighthouseResult{
		{URL: "https://a.com", PerformanceScore: 95, AccessibilityScore: 100, BestPracticesScore: 90, SEOScore: 80},
		{URL: "https://b.com", PerformanceScore: 45, AccessibilityScore: 60, BestPracticesScore: 70, SEOScore: 50},
		{URL: "https://c.com", PerformanceScore: 80, AccessibilityScore: 85, BestPracticesScore: 95, SEOScore: 90},
		{URL: "https://d.com", PerformanceScore: 92, AccessibilityScore: 98, BestPracticesScore: 88, SEOScore: 95},
	}

	summary := AggregateCWV(results)
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}

	if summary.PageCount != 4 {
		t.Errorf("expected page count 4, got %d", summary.PageCount)
	}

	// Avg performance: (95+45+80+92)/4 = 78
	expectedAvgPerf := 78.0
	if summary.AvgPerformance != expectedAvgPerf {
		t.Errorf("expected avg performance %f, got %f", expectedAvgPerf, summary.AvgPerformance)
	}

	// Avg accessibility: (100+60+85+98)/4 = 85.75
	expectedAvgAccess := 85.75
	if summary.AvgAccessibility != expectedAvgAccess {
		t.Errorf("expected avg accessibility %f, got %f", expectedAvgAccess, summary.AvgAccessibility)
	}

	// Pass rate: 2 out of 4 (95 and 92 >= 90) = 50%
	if summary.PassRate != 50.0 {
		t.Errorf("expected pass rate 50%%, got %f", summary.PassRate)
	}
}

func TestAggregateCWV_BottomN_LessThan5(t *testing.T) {
	results := []model.LighthouseResult{
		{URL: "https://a.com", PerformanceScore: 80, AccessibilityScore: 70},
		{URL: "https://b.com", PerformanceScore: 60, AccessibilityScore: 90},
	}

	summary := AggregateCWV(results)
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}

	if len(summary.WorstPerformance) != 2 {
		t.Errorf("expected 2 worst performance entries (fewer than 5), got %d", len(summary.WorstPerformance))
	}

	// Sorted ascending: b.com (60) then a.com (80).
	if summary.WorstPerformance[0].URL != "https://b.com" {
		t.Errorf("expected worst performance first to be b.com, got %s", summary.WorstPerformance[0].URL)
	}
	if summary.WorstPerformance[0].Score != 60 {
		t.Errorf("expected worst score 60, got %f", summary.WorstPerformance[0].Score)
	}

	if len(summary.WorstAccessibility) != 2 {
		t.Errorf("expected 2 worst accessibility entries, got %d", len(summary.WorstAccessibility))
	}
}

func TestAggregateCWV_BottomN_MoreThan5(t *testing.T) {
	results := []model.LighthouseResult{
		{URL: "https://1.com", PerformanceScore: 95, AccessibilityScore: 99},
		{URL: "https://2.com", PerformanceScore: 30, AccessibilityScore: 40},
		{URL: "https://3.com", PerformanceScore: 50, AccessibilityScore: 55},
		{URL: "https://4.com", PerformanceScore: 70, AccessibilityScore: 60},
		{URL: "https://5.com", PerformanceScore: 10, AccessibilityScore: 20},
		{URL: "https://6.com", PerformanceScore: 85, AccessibilityScore: 80},
		{URL: "https://7.com", PerformanceScore: 40, AccessibilityScore: 45},
	}

	summary := AggregateCWV(results)
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}

	if len(summary.WorstPerformance) != 5 {
		t.Fatalf("expected 5 worst performance entries, got %d", len(summary.WorstPerformance))
	}

	// Bottom 5 by performance ascending: 10, 30, 40, 50, 70.
	expectedScores := []float64{10, 30, 40, 50, 70}
	for i, expected := range expectedScores {
		if summary.WorstPerformance[i].Score != expected {
			t.Errorf("worst performance[%d]: expected score %f, got %f", i, expected, summary.WorstPerformance[i].Score)
		}
	}

	if len(summary.WorstAccessibility) != 5 {
		t.Fatalf("expected 5 worst accessibility entries, got %d", len(summary.WorstAccessibility))
	}
}

func TestAggregateCWV_PassRateEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		results  []model.LighthouseResult
		wantRate float64
	}{
		{
			name: "all pass",
			results: []model.LighthouseResult{
				{URL: "https://a.com", PerformanceScore: 95},
				{URL: "https://b.com", PerformanceScore: 90},
			},
			wantRate: 100.0,
		},
		{
			name: "none pass",
			results: []model.LighthouseResult{
				{URL: "https://a.com", PerformanceScore: 89},
				{URL: "https://b.com", PerformanceScore: 50},
			},
			wantRate: 0.0,
		},
		{
			name: "exactly at threshold",
			results: []model.LighthouseResult{
				{URL: "https://a.com", PerformanceScore: 90},
			},
			wantRate: 100.0,
		},
		{
			name: "just below threshold",
			results: []model.LighthouseResult{
				{URL: "https://a.com", PerformanceScore: 89.9},
			},
			wantRate: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := AggregateCWV(tt.results)
			if summary.PassRate != tt.wantRate {
				t.Errorf("expected pass rate %f, got %f", tt.wantRate, summary.PassRate)
			}
		})
	}
}

// --- Issue checks ---

func TestCheckCWVIssues_Empty(t *testing.T) {
	issues := CheckCWVIssues(nil)
	if issues != nil {
		t.Errorf("expected nil issues for nil results, got %+v", issues)
	}
}

func TestCheckCWVIssues_PoorPerformance(t *testing.T) {
	results := []model.LighthouseResult{
		{URL: "https://a.com", PerformanceScore: 30, AccessibilityScore: 95},
		{URL: "https://b.com", PerformanceScore: 40, AccessibilityScore: 95},
	}

	issues := CheckCWVIssues(results)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "lighthouse/poor-site-performance" {
			found = true
			if iss.Severity != model.SeverityWarning {
				t.Errorf("expected warning severity, got %s", iss.Severity)
			}
			if !strings.Contains(iss.Message, "35.0") {
				t.Errorf("expected message to contain avg score '35.0', got %q", iss.Message)
			}
		}
	}
	if !found {
		t.Error("expected lighthouse/poor-site-performance issue not found")
	}
}

func TestCheckCWVIssues_NoPoorPerformanceAboveThreshold(t *testing.T) {
	results := []model.LighthouseResult{
		{URL: "https://a.com", PerformanceScore: 60, AccessibilityScore: 95},
		{URL: "https://b.com", PerformanceScore: 70, AccessibilityScore: 95},
	}

	issues := CheckCWVIssues(results)

	for _, iss := range issues {
		if iss.CheckName == "lighthouse/poor-site-performance" {
			t.Errorf("did not expect poor-site-performance issue, got %+v", iss)
		}
	}
}

func TestCheckCWVIssues_PoorAccessibility(t *testing.T) {
	results := []model.LighthouseResult{
		{URL: "https://a.com", PerformanceScore: 95, AccessibilityScore: 70},
		{URL: "https://b.com", PerformanceScore: 95, AccessibilityScore: 80},
	}

	issues := CheckCWVIssues(results)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "lighthouse/poor-site-accessibility" {
			found = true
			if iss.Severity != model.SeverityWarning {
				t.Errorf("expected warning severity, got %s", iss.Severity)
			}
		}
	}
	if !found {
		t.Error("expected lighthouse/poor-site-accessibility issue not found")
	}
}

func TestCheckCWVIssues_LowPassRate(t *testing.T) {
	results := []model.LighthouseResult{
		{URL: "https://a.com", PerformanceScore: 30, AccessibilityScore: 95},
		{URL: "https://b.com", PerformanceScore: 40, AccessibilityScore: 95},
		{URL: "https://c.com", PerformanceScore: 95, AccessibilityScore: 95},
	}

	issues := CheckCWVIssues(results)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "lighthouse/low-pass-rate" {
			found = true
			if iss.Severity != model.SeverityInfo {
				t.Errorf("expected info severity, got %s", iss.Severity)
			}
			if !strings.Contains(iss.Message, "33.3%") {
				t.Errorf("expected message to contain '33.3%%', got %q", iss.Message)
			}
		}
	}
	if !found {
		t.Error("expected lighthouse/low-pass-rate issue not found")
	}
}

func TestCheckCWVIssues_NoIssuesWhenGood(t *testing.T) {
	results := []model.LighthouseResult{
		{URL: "https://a.com", PerformanceScore: 95, AccessibilityScore: 98, BestPracticesScore: 100, SEOScore: 95},
		{URL: "https://b.com", PerformanceScore: 92, AccessibilityScore: 95, BestPracticesScore: 90, SEOScore: 90},
	}

	issues := CheckCWVIssues(results)
	if len(issues) != 0 {
		t.Errorf("expected no issues for good scores, got %d: %+v", len(issues), issues)
	}
}

func TestCheckCWVIssues_AllIssuesFired(t *testing.T) {
	// Average perf < 50, avg access < 90, pass rate < 50%.
	results := []model.LighthouseResult{
		{URL: "https://a.com", PerformanceScore: 20, AccessibilityScore: 40},
		{URL: "https://b.com", PerformanceScore: 30, AccessibilityScore: 50},
	}

	issues := CheckCWVIssues(results)

	checks := make(map[string]bool)
	for _, iss := range issues {
		checks[iss.CheckName] = true
	}

	if !checks["lighthouse/poor-site-performance"] {
		t.Error("expected lighthouse/poor-site-performance issue")
	}
	if !checks["lighthouse/poor-site-accessibility"] {
		t.Error("expected lighthouse/poor-site-accessibility issue")
	}
	if !checks["lighthouse/low-pass-rate"] {
		t.Error("expected lighthouse/low-pass-rate issue")
	}
}

func TestBottomN_Empty(t *testing.T) {
	result := bottomN(nil, 5)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}

func TestBottomN_TiedScores(t *testing.T) {
	scores := []URLScore{
		{URL: "https://c.com", Score: 50},
		{URL: "https://a.com", Score: 50},
		{URL: "https://b.com", Score: 50},
	}

	result := bottomN(scores, 5)
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	// With equal scores, should be sorted by URL ascending.
	if result[0].URL != "https://a.com" || result[1].URL != "https://b.com" || result[2].URL != "https://c.com" {
		t.Errorf("expected stable sort by URL for tied scores, got %v", result)
	}
}

func TestBottomN_DoesNotMutateInput(t *testing.T) {
	scores := []URLScore{
		{URL: "https://b.com", Score: 80},
		{URL: "https://a.com", Score: 60},
	}

	_ = bottomN(scores, 5)

	// Original order should be preserved.
	if scores[0].URL != "https://b.com" {
		t.Error("bottomN mutated the input slice")
	}
}
