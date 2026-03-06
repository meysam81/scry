package crawler

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
)

// maxSitemapDepth is the maximum recursion depth for sitemap index resolution.
const maxSitemapDepth = 2

// urlSet represents a <urlset> element in a sitemap.
type urlSet struct {
	XMLName xml.Name  `xml:"urlset"`
	URLs    []siteURL `xml:"url"`
}

// siteURL represents a <url> element inside a <urlset>.
type siteURL struct {
	Loc string `xml:"loc"`
}

// sitemapIndex represents a <sitemapindex> element.
type sitemapIndex struct {
	XMLName  xml.Name  `xml:"sitemapindex"`
	Sitemaps []sitemap `xml:"sitemap"`
}

// sitemap represents a <sitemap> element inside a <sitemapindex>.
type sitemap struct {
	Loc string `xml:"loc"`
}

// ParseSitemap fetches and parses a sitemap.xml, returning discovered URLs.
func ParseSitemap(ctx context.Context, sitemapURL string) []string {
	return parseSitemapRecursive(ctx, sitemapURL, 0)
}

// parseSitemapRecursive fetches and parses a sitemap, following sitemap indexes up to maxSitemapDepth.
func parseSitemapRecursive(ctx context.Context, sitemapURL string, depth int) []string {
	if depth > maxSitemapDepth {
		return nil
	}

	body, err := fetchSitemapBody(ctx, sitemapURL)
	if err != nil {
		return nil
	}

	// Try parsing as urlset first.
	var us urlSet
	if err := xml.Unmarshal(body, &us); err == nil && len(us.URLs) > 0 {
		urls := make([]string, 0, len(us.URLs))
		for _, u := range us.URLs {
			if u.Loc != "" {
				urls = append(urls, u.Loc)
			}
		}
		return urls
	}

	// Try parsing as sitemap index.
	var si sitemapIndex
	if err := xml.Unmarshal(body, &si); err == nil && len(si.Sitemaps) > 0 {
		var urls []string
		for _, s := range si.Sitemaps {
			if s.Loc != "" {
				urls = append(urls, parseSitemapRecursive(ctx, s.Loc, depth+1)...)
			}
		}
		return urls
	}

	return nil
}

// fetchSitemapBody retrieves the raw body of a sitemap URL.
func fetchSitemapBody(ctx context.Context, sitemapURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sitemapURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	return io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
}
