package metrics

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"

	"github.com/meysam81/scry/internal/model"
	"github.com/meysam81/scry/internal/report"
)

const namespace = "scry"

// PushMetrics computes metrics from a CrawlResult and pushes them to a
// Prometheus Pushgateway.
func PushMetrics(result *model.CrawlResult, pushgatewayURL string, jobName string) error {
	if pushgatewayURL == "" {
		return nil
	}

	if result == nil {
		return nil
	}

	issuesTotalGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "issues_total",
			Help:      "Total number of audit issues found.",
		},
		[]string{"severity"},
	)

	pagesCrawledGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "pages_crawled",
		Help:      "Number of pages crawled.",
	})

	crawlDurationGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "crawl_duration_seconds",
		Help:      "Total crawl duration in seconds.",
	})

	healthScoreGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "health_score",
		Help:      "Overall site health score (0-100).",
	})

	issuesByCategoryGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "issues_by_category",
			Help:      "Number of issues by category.",
		},
		[]string{"category"},
	)

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		issuesTotalGauge,
		pagesCrawledGauge,
		crawlDurationGauge,
		healthScoreGauge,
		issuesByCategoryGauge,
	)

	summary := report.ComputeSummary(result)

	for sev, count := range summary.BySeverity {
		issuesTotalGauge.WithLabelValues(string(sev)).Set(float64(count))
	}

	for cat, count := range summary.ByCategory {
		issuesByCategoryGauge.WithLabelValues(cat).Set(float64(count))
	}

	pagesCrawledGauge.Set(float64(len(result.Pages)))
	crawlDurationGauge.Set(result.Duration.Seconds())

	score := 100.0
	score -= float64(summary.BySeverity[model.SeverityCritical]) * 10
	score -= float64(summary.BySeverity[model.SeverityWarning]) * 3
	score -= float64(summary.BySeverity[model.SeverityInfo]) * 1
	if score < 0 {
		score = 0
	}
	healthScoreGauge.Set(score)

	if jobName == "" {
		jobName = "scry"
	}

	pusher := push.New(pushgatewayURL, jobName).Gatherer(registry)
	if err := pusher.Add(); err != nil {
		return fmt.Errorf("push metrics: %w", err)
	}

	return nil
}
