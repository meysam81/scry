package check

import (
	"testing"
	"time"

	"github.com/meysam81/scry/internal/cmdutil"
	"github.com/meysam81/scry/internal/model"
)

func TestDetermineExitCode(t *testing.T) {
	tests := []struct {
		name     string
		issues   []model.Issue
		failOn   string
		wantCode int
	}{
		{
			name:     "no fail-on set returns 0",
			issues:   []model.Issue{{Severity: model.SeverityCritical}},
			failOn:   "",
			wantCode: 0,
		},
		{
			name:     "fail-on any with no issues returns 0",
			issues:   []model.Issue{},
			failOn:   "any",
			wantCode: 0,
		},
		{
			name:     "fail-on any with issues returns 1",
			issues:   []model.Issue{{Severity: model.SeverityInfo}},
			failOn:   "any",
			wantCode: 1,
		},
		{
			name:     "fail-on critical with critical issue returns 1",
			issues:   []model.Issue{{Severity: model.SeverityCritical}},
			failOn:   "critical",
			wantCode: 1,
		},
		{
			name:     "fail-on critical with warning issue returns 0",
			issues:   []model.Issue{{Severity: model.SeverityWarning}},
			failOn:   "critical",
			wantCode: 0,
		},
		{
			name:     "fail-on warning with critical issue returns 1",
			issues:   []model.Issue{{Severity: model.SeverityCritical}},
			failOn:   "warning",
			wantCode: 1,
		},
		{
			name:     "fail-on warning with warning issue returns 1",
			issues:   []model.Issue{{Severity: model.SeverityWarning}},
			failOn:   "warning",
			wantCode: 1,
		},
		{
			name:     "fail-on warning with info issue returns 0",
			issues:   []model.Issue{{Severity: model.SeverityInfo}},
			failOn:   "warning",
			wantCode: 0,
		},
		{
			name: "fail-on critical with mixed issues returns 1",
			issues: []model.Issue{
				{Severity: model.SeverityInfo},
				{Severity: model.SeverityWarning},
				{Severity: model.SeverityCritical},
			},
			failOn:   "critical",
			wantCode: 1,
		},
		{
			name: "fail-on warning with info and warning returns 1",
			issues: []model.Issue{
				{Severity: model.SeverityInfo},
				{Severity: model.SeverityWarning},
			},
			failOn:   "warning",
			wantCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &model.CrawlResult{
				Issues:    tt.issues,
				CrawledAt: time.Now(),
			}
			got := cmdutil.DetermineExitCode(result, tt.failOn)
			if got != tt.wantCode {
				t.Errorf("DetermineExitCode() = %d, want %d", got, tt.wantCode)
			}
		})
	}
}
