package lighthouse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBrowserlessClient_Run_Success(t *testing.T) {
	// Mock server that returns valid lighthouse JSON.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode request to verify format.
		var req browserlessRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.URL != "https://example.com" {
			t.Errorf("expected url https://example.com, got %s", req.URL)
		}
		if len(req.Config.Settings.OnlyCategories) != 4 {
			t.Errorf("expected 4 categories, got %d", len(req.Config.Settings.OnlyCategories))
		}

		// Return mock response.
		resp := browserlessResponse{}
		perf := 0.85
		a11y := 0.92
		bp := 0.88
		seo := 0.95
		resp.Categories.Performance.Score = &perf
		resp.Categories.Accessibility.Score = &a11y
		resp.Categories.BestPractices.Score = &bp
		resp.Categories.SEO.Score = &seo

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewBrowserlessClient(server.URL)
	result, err := client.Run(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.URL != "https://example.com" {
		t.Errorf("expected url https://example.com, got %s", result.URL)
	}
	if result.PerformanceScore != 85 {
		t.Errorf("expected performance 85, got %f", result.PerformanceScore)
	}
	if result.AccessibilityScore != 92 {
		t.Errorf("expected accessibility 92, got %f", result.AccessibilityScore)
	}
	if result.BestPracticesScore != 88 {
		t.Errorf("expected best practices 88, got %f", result.BestPracticesScore)
	}
	if result.SEOScore != 95 {
		t.Errorf("expected seo 95, got %f", result.SEOScore)
	}
	if result.Source != browserlessSource {
		t.Errorf("expected source %s, got %s", browserlessSource, result.Source)
	}
}

func TestBrowserlessClient_Run_ErrorResponse(t *testing.T) {
	// Mock server that returns 500.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewBrowserlessClient(server.URL)
	_, err := client.Run(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBrowserlessClient_Run_ConnectionFailure(t *testing.T) {
	// Use invalid URL to trigger connection failure.
	client := NewBrowserlessClient("http://invalid.localhost:99999")
	_, err := client.Run(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBrowserlessClient_Run_MalformedJSON(t *testing.T) {
	// Mock server that returns invalid JSON.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"invalid": json}`))
	}))
	defer server.Close()

	client := NewBrowserlessClient(server.URL)
	_, err := client.Run(context.Background(), "https://example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBrowserlessClient_Run_NilScores(t *testing.T) {
	// Mock server that returns nil scores.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := browserlessResponse{}
		// All scores are nil.
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := NewBrowserlessClient(server.URL)
	result, err := client.Run(context.Background(), "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// scoreValue should return 0 for nil pointers.
	if result.PerformanceScore != 0 {
		t.Errorf("expected performance 0, got %f", result.PerformanceScore)
	}
	if result.AccessibilityScore != 0 {
		t.Errorf("expected accessibility 0, got %f", result.AccessibilityScore)
	}
	if result.BestPracticesScore != 0 {
		t.Errorf("expected best practices 0, got %f", result.BestPracticesScore)
	}
	if result.SEOScore != 0 {
		t.Errorf("expected seo 0, got %f", result.SEOScore)
	}
}
