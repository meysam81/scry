package metrics

import (
	"testing"
	"time"

	"github.com/meysam81/scry/internal/model"
)

func TestPushMetrics_EmptyURL(t *testing.T) {
	result := &model.CrawlResult{
		SeedURL: "https://example.com",
		Pages:   []*model.Page{{URL: "https://example.com"}},
		Issues: []model.Issue{
			{CheckName: "seo/missing-title", Severity: model.SeverityCritical},
		},
		Duration: 5 * time.Second,
	}

	if err := PushMetrics(result, "", "test"); err != nil {
		t.Fatalf("expected nil error for empty URL, got: %v", err)
	}
}

func TestPushMetrics_NilResult(t *testing.T) {
	if err := PushMetrics(nil, "", "test"); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}
