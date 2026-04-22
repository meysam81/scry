package crawler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"github.com/meysam81/scry/internal/config"
	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/core/model"
)

// Crawler orchestrates concurrent web crawling.
type Crawler struct {
	cfg     *config.Config
	fetcher Fetcher
	log     logger.Logger
}

// NewCrawler creates a new Crawler with the given configuration and fetcher.
func NewCrawler(cfg *config.Config, fetcher Fetcher, l logger.Logger) *Crawler {
	return &Crawler{
		cfg:     cfg,
		fetcher: fetcher,
		log:     l,
	}
}

// Run crawls starting from seedURL and returns the collected results.
func (c *Crawler) Run(ctx context.Context, seedURL string) (*model.CrawlResult, error) {
	parsed, err := url.Parse(seedURL)
	if err != nil {
		return nil, fmt.Errorf("crawl: invalid seed URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("crawl: unsupported scheme %q", parsed.Scheme)
	}

	start := time.Now()
	seedHost := parsed.Hostname()

	frontier := NewFrontier(seedHost, c.cfg.MaxPages, c.cfg.IncludePatterns, c.cfg.ExcludePatterns)

	// Set up robots checker if enabled.
	var robots *RobotsChecker
	if c.cfg.RespectRobots {
		robots = NewRobotsChecker(c.cfg.UserAgent, c.log)
	}

	// Seed the frontier from sitemap.
	sitemapURL := strings.TrimRight(seedURL, "/") + "/sitemap.xml"
	sitemapURLs := ParseSitemap(ctx, sitemapURL)

	// Truncate sitemap URLs to MaxPages to bound memory usage.
	if c.cfg.MaxPages > 0 && len(sitemapURLs) > c.cfg.MaxPages {
		sitemapURLs = sitemapURLs[:c.cfg.MaxPages]
	}

	// Track sitemap URLs to mark pages later.
	sitemapSet := make(map[string]bool, len(sitemapURLs))
	for _, u := range sitemapURLs {
		sitemapSet[u] = true
		frontier.Add(u, 1)
	}

	// Add seed URL at depth 0.
	frontier.Add(seedURL, 0)

	// Rate limiter.
	burst := min(c.cfg.RateLimit, 5)
	limiter := rate.NewLimiter(rate.Limit(c.cfg.RateLimit), burst)

	// Results collection.
	var (
		pagesMu sync.Mutex
		pages   []*model.Page
	)

	// Task channel and active worker counter.
	taskChan := make(chan FrontierTask)
	var active int64

	// Launch workers.
	var wg sync.WaitGroup
	for range c.cfg.Concurrency {
		wg.Go(func() {
			for task := range taskChan {
				if err := limiter.Wait(ctx); err != nil {
					atomic.AddInt64(&active, -1)
					return
				}

				// Check robots.txt.
				if robots != nil && !robots.IsAllowed(ctx, task.URL) {
					atomic.AddInt64(&active, -1)
					continue
				}

				page, err := c.fetcher.Fetch(ctx, task.URL)
				if err != nil {
					c.log.Warn().Err(err).Str("url", task.URL).Msg("fetch failed")
					atomic.AddInt64(&active, -1)
					continue
				}

				page.Depth = task.Depth
				page.InSitemap = sitemapSet[task.URL]

				// Parse HTML pages for links and assets.
				if isHTML(page.ContentType) {
					base, parseErr := url.Parse(page.URL)
					if parseErr == nil {
						links, assets := ParseHTML(base, page.Body)
						page.Links = links
						page.Assets = assets

						// Add discovered links to frontier.
						if task.Depth+1 <= c.cfg.MaxDepth {
							for _, link := range links {
								frontier.Add(link, task.Depth+1)
							}
						}
					}
				}

				pagesMu.Lock()
				pages = append(pages, page)
				pagesMu.Unlock()

				atomic.AddInt64(&active, -1)
			}
		})
	}

	// Coordinator: feed tasks to workers.
	go func() {
		defer close(taskChan)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			task, ok := frontier.Dequeue()
			if ok {
				atomic.AddInt64(&active, 1)
				select {
				case taskChan <- task:
				case <-ctx.Done():
					return
				}
				continue
			}

			// No task available — check if all workers are idle.
			if atomic.LoadInt64(&active) == 0 {
				return
			}

			// Brief sleep to avoid busy-waiting.
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()

	return &model.CrawlResult{
		SeedURL:   seedURL,
		Pages:     pages,
		CrawledAt: start,
		Duration:  time.Since(start),
	}, nil
}

// isHTML reports whether the content type indicates an HTML document.
func isHTML(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")
}
