// Package browser implements a headless-browser-based page fetcher using rod.
package browser

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"

	"github.com/meysam81/scry/internal/model"
)

// defaultContentType is the content type returned for all browser-fetched pages.
const defaultContentType = "text/html"

// BrowserFetcher implements crawler.Fetcher using a headless browser via rod.
type BrowserFetcher struct {
	browser   *rod.Browser
	timeout   time.Duration
	userAgent string
}

// NewBrowserFetcher creates a new BrowserFetcher connected to a headless browser.
// If browserlessURL is non-empty, it connects to a remote browser instance;
// otherwise it launches a local headless browser.
func NewBrowserFetcher(browserlessURL, userAgent string, timeout time.Duration) (*BrowserFetcher, error) {
	var b *rod.Browser
	var err error

	if browserlessURL != "" {
		controlURL, err := launcher.ResolveURL(browserlessURL)
		if err != nil {
			return nil, fmt.Errorf("browser: resolve remote URL %q: %w", browserlessURL, err)
		}
		b = rod.New().ControlURL(controlURL)
	} else {
		b = rod.New()
	}

	err = b.Connect()
	if err != nil {
		return nil, fmt.Errorf("browser: connect: %w", err)
	}

	return &BrowserFetcher{
		browser:   b,
		timeout:   timeout,
		userAgent: userAgent,
	}, nil
}

// Fetch navigates to the given URL in a new browser tab, waits for stability,
// and returns the rendered page content.
func (f *BrowserFetcher) Fetch(ctx context.Context, u string) (*model.Page, error) {
	start := time.Now()

	page, err := f.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("browser fetch %s: create page: %w", u, err)
	}
	defer func() { _ = page.Close() }()

	// Apply timeout to the page context.
	page = page.Context(ctx).Timeout(f.timeout)

	// Set user-agent.
	if f.userAgent != "" {
		err = page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: f.userAgent,
		})
		if err != nil {
			return nil, fmt.Errorf("browser fetch %s: set user agent: %w", u, err)
		}
	}

	// Enable network events to capture response status code.
	var statusCode int
	var responseHeaders http.Header

	err = proto.NetworkEnable{}.Call(page)
	if err != nil {
		return nil, fmt.Errorf("browser fetch %s: enable network: %w", u, err)
	}

	// Set up response listener before navigation.
	done := make(chan struct{}, 1)
	go page.EachEvent(func(e *proto.NetworkResponseReceived) {
		// Capture the status from the main document response (type Document).
		if e.Type == proto.NetworkResourceTypeDocument {
			statusCode = e.Response.Status
			responseHeaders = make(http.Header)
			for k, v := range e.Response.Headers {
				responseHeaders.Set(k, fmt.Sprint(v))
			}
			select {
			case done <- struct{}{}:
			default:
			}
		}
	})()

	// Navigate to URL.
	err = page.Navigate(u)
	if err != nil {
		return nil, fmt.Errorf("browser fetch %s: navigate: %w", u, err)
	}

	// Wait for the document response event.
	select {
	case <-done:
	case <-ctx.Done():
		return nil, fmt.Errorf("browser fetch %s: %w", u, ctx.Err())
	case <-time.After(f.timeout):
		// Proceed with what we have; status may be 0.
	}

	// Wait for page stability (network idle + DOM settled).
	err = page.WaitStable(300 * time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("browser fetch %s: wait stable: %w", u, err)
	}

	// Get the final URL after any redirects.
	info, err := page.Info()
	if err != nil {
		return nil, fmt.Errorf("browser fetch %s: page info: %w", u, err)
	}
	finalURL := info.URL

	// Get the rendered HTML.
	html, err := page.HTML()
	if err != nil {
		return nil, fmt.Errorf("browser fetch %s: get HTML: %w", u, err)
	}

	duration := time.Since(start)

	result := &model.Page{
		URL:           finalURL,
		StatusCode:    statusCode,
		ContentType:   defaultContentType,
		Body:          []byte(html),
		Headers:       responseHeaders,
		FetchedAt:     start,
		FetchDuration: duration,
	}

	return result, nil
}

// Close shuts down the browser instance and releases resources.
func (f *BrowserFetcher) Close() error {
	if f.browser != nil {
		return f.browser.Close()
	}
	return nil
}
