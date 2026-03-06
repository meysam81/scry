package lighthouse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/meysam81/scry/internal/model"
)

// browserlessTimeout is the default timeout for browserless requests.
const browserlessTimeout = 60 * time.Second

// browserlessSource is the source label for browserless results.
const browserlessSource = "browserless"

// browserlessRequest represents the request payload sent to browserless.
type browserlessRequest struct {
	URL    string            `json:"url"`
	Config browserlessConfig `json:"config"`
}

// browserlessConfig holds the lighthouse configuration for browserless.
type browserlessConfig struct {
	Settings browserlessSettings `json:"settings"`
}

// browserlessSettings specifies which categories to run.
type browserlessSettings struct {
	OnlyCategories []string `json:"onlyCategories"`
}

// browserlessResponse represents the lighthouse response from browserless.
type browserlessResponse struct {
	Categories struct {
		Performance   browserlessCategory `json:"performance"`
		Accessibility browserlessCategory `json:"accessibility"`
		BestPractices browserlessCategory `json:"best-practices"`
		SEO           browserlessCategory `json:"seo"`
	} `json:"categories"`
}

// browserlessCategory holds a single category score from browserless.
type browserlessCategory struct {
	Score *float64 `json:"score"`
}

// BrowserlessClient runs Lighthouse audits via a browserless container.
type BrowserlessClient struct {
	endpoint string
	client   *http.Client
}

// NewBrowserlessClient creates a new browserless client.
func NewBrowserlessClient(browserlessURL string) *BrowserlessClient {
	return &BrowserlessClient{
		endpoint: browserlessURL + "/lighthouse",
		client: &http.Client{
			Timeout: browserlessTimeout,
		},
	}
}

// Run executes a Lighthouse audit via browserless and returns the result.
func (c *BrowserlessClient) Run(ctx context.Context, targetURL string) (*model.LighthouseResult, error) {
	reqBody := browserlessRequest{
		URL: targetURL,
		Config: browserlessConfig{
			Settings: browserlessSettings{
				OnlyCategories: []string{"performance", "accessibility", "best-practices", "seo"},
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal browserless request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create browserless request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("browserless request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("browserless api returned status %d", resp.StatusCode)
	}

	var browserlessResp browserlessResponse
	if err := json.NewDecoder(resp.Body).Decode(&browserlessResp); err != nil {
		return nil, fmt.Errorf("decode browserless response: %w", err)
	}

	result := &model.LighthouseResult{
		URL:                targetURL,
		PerformanceScore:   scoreValue(browserlessResp.Categories.Performance.Score),
		AccessibilityScore: scoreValue(browserlessResp.Categories.Accessibility.Score),
		BestPracticesScore: scoreValue(browserlessResp.Categories.BestPractices.Score),
		SEOScore:           scoreValue(browserlessResp.Categories.SEO.Score),
		FetchedAt:          time.Now(),
		Source:             browserlessSource,
	}

	return result, nil
}
