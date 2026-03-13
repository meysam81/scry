package crawler

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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
// Gzip-compressed sitemaps (.xml.gz) are transparently decompressed.
func ParseSitemap(ctx context.Context, sitemapURL string) []string {
	parsed, err := url.Parse(sitemapURL)
	if err != nil {
		return nil
	}
	return parseSitemapRecursive(ctx, sitemapURL, parsed.Hostname(), 0)
}

// parseSitemapRecursive fetches and parses a sitemap, following sitemap indexes
// up to maxSitemapDepth. Child sitemaps on different hosts are rejected to
// prevent SSRF via sitemap index entries.
func parseSitemapRecursive(ctx context.Context, sitemapURL, allowedHost string, depth int) []string {
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
			if s.Loc == "" {
				continue
			}
			// Reject child sitemaps on different hosts.
			childParsed, err := url.Parse(s.Loc)
			if err != nil || childParsed.Hostname() != allowedHost {
				continue
			}
			urls = append(urls, parseSitemapRecursive(ctx, s.Loc, allowedHost, depth+1)...)
		}
		return urls
	}

	return nil
}

// gzipMagic contains the first two bytes of a gzip stream.
var gzipMagic = []byte{0x1f, 0x8b}

// fetchSitemapBody retrieves the raw body of a sitemap URL.
// Gzip-compressed responses are transparently decompressed based on the URL
// suffix (.gz), Content-Encoding header, Content-Type header, or gzip magic bytes.
func fetchSitemapBody(ctx context.Context, sitemapURL string) (_ []byte, retErr error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sitemapURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := internalClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("close resp body: %w", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sitemap %s: status %d", sitemapURL, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, err
	}

	if isGzipped(sitemapURL, resp, body) {
		return decompressGzip(body)
	}

	return body, nil
}

// isGzipped determines whether the response body is gzip-compressed by
// checking the URL suffix, response headers, and gzip magic bytes.
func isGzipped(sitemapURL string, resp *http.Response, body []byte) bool {
	if strings.HasSuffix(sitemapURL, ".gz") {
		return true
	}
	if resp.Header.Get("Content-Encoding") == "gzip" {
		return true
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "gzip") {
		return true
	}
	if len(body) >= 2 && bytes.HasPrefix(body, gzipMagic) {
		return true
	}
	return false
}

// decompressGzip decompresses a gzip-compressed byte slice.
func decompressGzip(data []byte) (_ []byte, retErr error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer func() {
		if err := gr.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("gzip close: %w", err)
		}
	}()

	return io.ReadAll(io.LimitReader(gr, maxBodySize))
}
