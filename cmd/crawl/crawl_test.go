package crawl

import (
	"testing"

	"github.com/meysam81/scry/internal/cmdutil"
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
			name:   "empty fail-on returns 0",
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
			name: "critical issues with fail-on critical",
			issues: []model.Issue{
				{Severity: model.SeverityCritical, CheckName: "test"},
			},
			failOn: "critical",
			want:   1,
		},
		{
			name: "warning issues with fail-on critical",
			issues: []model.Issue{
				{Severity: model.SeverityWarning, CheckName: "test"},
			},
			failOn: "critical",
			want:   0,
		},
		{
			name: "warning issues with fail-on warning",
			issues: []model.Issue{
				{Severity: model.SeverityWarning, CheckName: "test"},
			},
			failOn: "warning",
			want:   1,
		},
		{
			name: "critical issues with fail-on warning",
			issues: []model.Issue{
				{Severity: model.SeverityCritical, CheckName: "test"},
			},
			failOn: "warning",
			want:   1,
		},
		{
			name: "info issues with fail-on warning",
			issues: []model.Issue{
				{Severity: model.SeverityInfo, CheckName: "test"},
			},
			failOn: "warning",
			want:   0,
		},
		{
			name: "any issues with fail-on any",
			issues: []model.Issue{
				{Severity: model.SeverityInfo, CheckName: "test"},
			},
			failOn: "any",
			want:   1,
		},
		{
			name:   "no issues with fail-on any",
			issues: nil,
			failOn: "any",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &model.CrawlResult{Issues: tt.issues}
			got := cmdutil.DetermineExitCode(result, tt.failOn)
			if got != tt.want {
				t.Errorf("DetermineExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}
