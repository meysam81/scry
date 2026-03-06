package cmdutil

import (
	"context"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestDetermineExitCode(t *testing.T) {
	tests := []struct {
		name   string
		issues []model.Issue
		failOn string
		want   int
	}{
		{
			name:   "no fail-on set returns 0",
			issues: []model.Issue{{Severity: model.SeverityCritical}},
			failOn: "",
			want:   0,
		},
		{
			name:   "no issues returns 0",
			issues: nil,
			failOn: "critical",
			want:   0,
		},
		{
			name:   "fail-on any with no issues returns 0",
			issues: nil,
			failOn: "any",
			want:   0,
		},
		{
			name:   "fail-on any with issues returns 1",
			issues: []model.Issue{{Severity: model.SeverityInfo}},
			failOn: "any",
			want:   1,
		},
		{
			name:   "fail-on critical with critical issue returns 1",
			issues: []model.Issue{{Severity: model.SeverityCritical}},
			failOn: "critical",
			want:   1,
		},
		{
			name:   "fail-on critical with warning issue returns 0",
			issues: []model.Issue{{Severity: model.SeverityWarning}},
			failOn: "critical",
			want:   0,
		},
		{
			name:   "fail-on warning with critical issue returns 1",
			issues: []model.Issue{{Severity: model.SeverityCritical}},
			failOn: "warning",
			want:   1,
		},
		{
			name:   "fail-on warning with warning issue returns 1",
			issues: []model.Issue{{Severity: model.SeverityWarning}},
			failOn: "warning",
			want:   1,
		},
		{
			name:   "fail-on warning with info issue returns 0",
			issues: []model.Issue{{Severity: model.SeverityInfo}},
			failOn: "warning",
			want:   0,
		},
		{
			name: "fail-on critical with mixed issues returns 1",
			issues: []model.Issue{
				{Severity: model.SeverityInfo},
				{Severity: model.SeverityWarning},
				{Severity: model.SeverityCritical},
			},
			failOn: "critical",
			want:   1,
		},
		{
			name: "fail-on warning with info and warning returns 1",
			issues: []model.Issue{
				{Severity: model.SeverityInfo},
				{Severity: model.SeverityWarning},
			},
			failOn: "warning",
			want:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &model.CrawlResult{Issues: tt.issues}
			got := DetermineExitCode(context.Background(), result, tt.failOn)
			if got != tt.want {
				t.Errorf("DetermineExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDetermineExitCode_UnknownFailOn(t *testing.T) {
	result := &model.CrawlResult{
		Issues: []model.Issue{{Severity: model.SeverityCritical}},
	}
	got := DetermineExitCode(context.Background(), result, "critcal") // typo on purpose
	if got != 0 {
		t.Errorf("DetermineExitCode() = %d, want 0 for unknown fail-on", got)
	}
}
