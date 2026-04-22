package lighthouse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/time/rate"

	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/core/model"
)

// PSI API constants.
const (
	// PSIEndpoint is the PageSpeed Insights API URL.
	PSIEndpoint = "https://www.googleapis.com/pagespeedonline/v5/runPagespeed"
	// PSITimeout is the per-request timeout for PSI calls.
	PSITimeout = 60 * time.Second
	// psiRateLimitNoKey is the rate limit without an API key (1 req/s).
	psiRateLimitNoKey = 1
	// psiRateLimitWithKey is the rate limit with an API key (25 req/s).
	psiRateLimitWithKey = 25
	// psiSource is the source label for PSI results.
	psiSource = "psi"
)

// psiResponse represents the relevant parts of the PageSpeed Insights API response.
type psiResponse struct {
	LighthouseResult struct {
		Categories struct {
			Performance   psiCategory `json:"performance"`
			Accessibility psiCategory `json:"accessibility"`
			BestPractices psiCategory `json:"best-practices"`
			SEO           psiCategory `json:"seo"`
		} `json:"categories"`
	} `json:"lighthouseResult"`
}

// psiCategory holds a single category score from the PSI response.
type psiCategory struct {
	Score *float64 `json:"score"`
}

// PSIClient calls the PageSpeed Insights API.
type PSIClient struct {
	apiKey   string
	strategy string
	client   *http.Client
	limiter  *rate.Limiter
	log      logger.Logger
}

// NewPSIClient creates a PSI client with appropriate rate limiting.
func NewPSIClient(apiKey, strategy string, l logger.Logger) *PSIClient {
	rps := psiRateLimitNoKey
	if apiKey != "" {
		rps = psiRateLimitWithKey
	}

	return &PSIClient{
		apiKey:   apiKey,
		strategy: strategy,
		client: &http.Client{
			Timeout: PSITimeout,
		},
		limiter: rate.NewLimiter(rate.Limit(rps), rps),
		log:     l,
	}
}

// Run executes a Lighthouse audit via the PSI API and returns the result.
func (c *PSIClient) Run(ctx context.Context, targetURL string) (*model.LighthouseResult, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	reqURL, err := c.buildURL(targetURL)
	if err != nil {
		return nil, fmt.Errorf("build psi url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create psi request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("psi request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.log.Warn().Err(err).Str("url", targetURL).Msg("psi resp body close failed")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1024))
		if readErr != nil {
			return nil, fmt.Errorf("psi api returned status %d (body read failed: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("psi api returned status %d: %s", resp.StatusCode, body)
	}

	var psi psiResponse
	if err := json.NewDecoder(resp.Body).Decode(&psi); err != nil {
		return nil, fmt.Errorf("decode psi response: %w", err)
	}

	result := &model.LighthouseResult{
		URL:                targetURL,
		PerformanceScore:   scoreValue(psi.LighthouseResult.Categories.Performance.Score),
		AccessibilityScore: scoreValue(psi.LighthouseResult.Categories.Accessibility.Score),
		BestPracticesScore: scoreValue(psi.LighthouseResult.Categories.BestPractices.Score),
		SEOScore:           scoreValue(psi.LighthouseResult.Categories.SEO.Score),
		FetchedAt:          time.Now(),
		Source:             psiSource,
	}

	return result, nil
}

// buildURL constructs the PSI API request URL with query parameters.
func (c *PSIClient) buildURL(targetURL string) (string, error) {
	u, err := url.Parse(PSIEndpoint)
	if err != nil {
		return "", fmt.Errorf("parse psi endpoint: %w", err)
	}

	q := u.Query()
	q.Set("url", targetURL)
	q.Set("strategy", c.strategy)
	if c.apiKey != "" {
		q.Set("key", c.apiKey)
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// scoreValue extracts a float64 from a *float64 pointer, multiplying by 100.
// Returns 0 if the pointer is nil.
func scoreValue(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v * 100
}

// RateLimit returns the current rate limiter for testing inspection.
func (c *PSIClient) RateLimit() *rate.Limiter {
	return c.limiter
}
