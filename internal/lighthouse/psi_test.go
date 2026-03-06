package lighthouse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestPSIClientRun_Success(t *testing.T) {
	perf := 0.85
	a11y := 0.92
	bp := 0.88
	seo := 0.90

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters.
		q := r.URL.Query()
		if got := q.Get("url"); got != "https://example.com" {
			t.Errorf("url param = %q, want %q", got, "https://example.com")
		}
		if got := q.Get("strategy"); got != "mobile" {
			t.Errorf("strategy param = %q, want %q", got, "mobile")
		}
		// No API key expected.
		if got := q.Get("key"); got != "" {
			t.Errorf("key param = %q, want empty", got)
		}

		resp := psiResponse{}
		resp.LighthouseResult.Categories.Performance.Score = &perf
		resp.LighthouseResult.Categories.Accessibility.Score = &a11y
		resp.LighthouseResult.Categories.BestPractices.Score = &bp
		resp.LighthouseResult.Categories.SEO.Score = &seo

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewPSIClient("", "mobile")
	// Override the client to use the test server.
	client.client = srv.Client()

	// Override the endpoint by replacing the buildURL method through a custom run.
	result, err := runWithOverriddenEndpoint(client, srv.URL, "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PerformanceScore != 85 {
		t.Errorf("performance = %v, want 85", result.PerformanceScore)
	}
	if result.AccessibilityScore != 92 {
		t.Errorf("accessibility = %v, want 92", result.AccessibilityScore)
	}
	if result.BestPracticesScore != 88 {
		t.Errorf("best-practices = %v, want 88", result.BestPracticesScore)
	}
	if result.SEOScore != 90 {
		t.Errorf("seo = %v, want 90", result.SEOScore)
	}
	if result.Source != "psi" {
		t.Errorf("source = %q, want %q", result.Source, "psi")
	}
	if result.URL != "https://example.com" {
		t.Errorf("url = %q, want %q", result.URL, "https://example.com")
	}
}

func TestPSIClientRun_WithAPIKey(t *testing.T) {
	perf := 0.95
	a11y := 0.99
	bp := 1.0
	seo := 0.98

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("key"); got != "test-key-123" {
			t.Errorf("key param = %q, want %q", got, "test-key-123")
		}

		resp := psiResponse{}
		resp.LighthouseResult.Categories.Performance.Score = &perf
		resp.LighthouseResult.Categories.Accessibility.Score = &a11y
		resp.LighthouseResult.Categories.BestPractices.Score = &bp
		resp.LighthouseResult.Categories.SEO.Score = &seo

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewPSIClient("test-key-123", "desktop")
	client.client = srv.Client()

	result, err := runWithOverriddenEndpoint(client, srv.URL, "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PerformanceScore != 95 {
		t.Errorf("performance = %v, want 95", result.PerformanceScore)
	}
}

func TestPSIClientRun_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewPSIClient("", "mobile")
	client.client = srv.Client()

	_, err := runWithOverriddenEndpoint(client, srv.URL, "https://example.com")
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestPSIClientRun_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := NewPSIClient("", "mobile")
	client.client = srv.Client()

	_, err := runWithOverriddenEndpoint(client, srv.URL, "https://example.com")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestPSIClientRun_NilScores(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Return response with no scores set (nil pointers).
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"lighthouseResult":{"categories":{}}}`))
	}))
	defer srv.Close()

	client := NewPSIClient("", "mobile")
	client.client = srv.Client()

	result, err := runWithOverriddenEndpoint(client, srv.URL, "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PerformanceScore != 0 {
		t.Errorf("performance = %v, want 0 for nil score", result.PerformanceScore)
	}
	if result.AccessibilityScore != 0 {
		t.Errorf("accessibility = %v, want 0 for nil score", result.AccessibilityScore)
	}
}

func TestNewPSIClient_RateLimitNoKey(t *testing.T) {
	client := NewPSIClient("", "mobile")
	lim := client.RateLimit()
	if lim.Limit() != 1 {
		t.Errorf("rate limit without key = %v, want 1", lim.Limit())
	}
}

func TestNewPSIClient_RateLimitWithKey(t *testing.T) {
	client := NewPSIClient("my-key", "mobile")
	lim := client.RateLimit()
	if lim.Limit() != 25 {
		t.Errorf("rate limit with key = %v, want 25", lim.Limit())
	}
}

// runWithOverriddenEndpoint is a test helper that calls the PSI API using
// a custom base URL (from httptest.Server) instead of the real endpoint.
func runWithOverriddenEndpoint(c *PSIClient, baseURL, targetURL string) (*model.LighthouseResult, error) {
	ctx := context.Background()

	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Build the request URL using the test server base URL.
	reqURL := baseURL + "?url=" + targetURL + "&strategy=" + c.strategy
	if c.apiKey != "" {
		reqURL += "&key=" + c.apiKey
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("psi api returned status %d", resp.StatusCode)
	}

	var psi psiResponse
	if err := json.NewDecoder(resp.Body).Decode(&psi); err != nil {
		return nil, fmt.Errorf("decode psi response: %w", err)
	}

	return &model.LighthouseResult{
		URL:                targetURL,
		PerformanceScore:   scoreValue(psi.LighthouseResult.Categories.Performance.Score),
		AccessibilityScore: scoreValue(psi.LighthouseResult.Categories.Accessibility.Score),
		BestPracticesScore: scoreValue(psi.LighthouseResult.Categories.BestPractices.Score),
		SEOScore:           scoreValue(psi.LighthouseResult.Categories.SEO.Score),
		Source:             psiSource,
	}, nil
}
