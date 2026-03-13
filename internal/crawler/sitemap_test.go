package crawler

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseSitemap_URLSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>http://example.com/page1</loc></url>
  <url><loc>http://example.com/page2</loc></url>
  <url><loc>http://example.com/page3</loc></url>
</urlset>`))
	}))
	defer srv.Close()

	urls := ParseSitemap(context.Background(), srv.URL+"/sitemap.xml")
	if len(urls) != 3 {
		t.Fatalf("got %d URLs, want 3", len(urls))
	}

	expected := []string{
		"http://example.com/page1",
		"http://example.com/page2",
		"http://example.com/page3",
	}
	for i, u := range urls {
		if u != expected[i] {
			t.Errorf("url[%d] = %q, want %q", i, u, expected[i])
		}
	}
}

func TestParseSitemap_SitemapIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		switch r.URL.Path {
		case "/sitemap_index.xml":
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>` + "http://" + r.Host + `/sitemap1.xml</loc></sitemap>
</sitemapindex>`))
		case "/sitemap1.xml":
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>http://example.com/from-index</loc></url>
</urlset>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	urls := ParseSitemap(context.Background(), srv.URL+"/sitemap_index.xml")
	if len(urls) != 1 {
		t.Fatalf("got %d URLs, want 1", len(urls))
	}
	if urls[0] != "http://example.com/from-index" {
		t.Errorf("url = %q, want %q", urls[0], "http://example.com/from-index")
	}
}

func TestParseSitemap_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	urls := ParseSitemap(context.Background(), srv.URL+"/sitemap.xml")
	if len(urls) != 0 {
		t.Errorf("got %d URLs, want 0 for 404 sitemap", len(urls))
	}
}

func TestParseSitemap_MalformedXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`this is not valid XML <<>>`))
	}))
	defer srv.Close()

	urls := ParseSitemap(context.Background(), srv.URL+"/sitemap.xml")
	if len(urls) != 0 {
		t.Errorf("got %d URLs, want 0 for malformed XML", len(urls))
	}
}

// gzipCompress compresses data using gzip.
func gzipCompress(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

func TestParseSitemap_GzipByURLSuffix(t *testing.T) {
	xmlBody := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>http://example.com/gz-page1</loc></url>
  <url><loc>http://example.com/gz-page2</loc></url>
</urlset>`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(gzipCompress(t, xmlBody))
	}))
	defer srv.Close()

	urls := ParseSitemap(context.Background(), srv.URL+"/sitemap.xml.gz")
	if len(urls) != 2 {
		t.Fatalf("got %d URLs, want 2", len(urls))
	}
	if urls[0] != "http://example.com/gz-page1" {
		t.Errorf("url[0] = %q, want %q", urls[0], "http://example.com/gz-page1")
	}
}

func TestParseSitemap_GzipByContentEncoding(t *testing.T) {
	xmlBody := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>http://example.com/ce-page</loc></url>
</urlset>`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(gzipCompress(t, xmlBody))
	}))
	defer srv.Close()

	urls := ParseSitemap(context.Background(), srv.URL+"/sitemap.xml")
	if len(urls) != 1 {
		t.Fatalf("got %d URLs, want 1", len(urls))
	}
	if urls[0] != "http://example.com/ce-page" {
		t.Errorf("url[0] = %q, want %q", urls[0], "http://example.com/ce-page")
	}
}

func TestParseSitemap_GzipByContentType(t *testing.T) {
	xmlBody := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>http://example.com/ct-page</loc></url>
</urlset>`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(gzipCompress(t, xmlBody))
	}))
	defer srv.Close()

	urls := ParseSitemap(context.Background(), srv.URL+"/sitemap.xml")
	if len(urls) != 1 {
		t.Fatalf("got %d URLs, want 1", len(urls))
	}
	if urls[0] != "http://example.com/ct-page" {
		t.Errorf("url[0] = %q, want %q", urls[0], "http://example.com/ct-page")
	}
}

func TestParseSitemap_GzipByMagicBytes(t *testing.T) {
	xmlBody := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>http://example.com/magic-page</loc></url>
</urlset>`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No gzip hints in URL, Content-Type, or Content-Encoding.
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(gzipCompress(t, xmlBody))
	}))
	defer srv.Close()

	// URL does not end with .gz — detection relies on magic bytes.
	urls := ParseSitemap(context.Background(), srv.URL+"/sitemap.xml")
	if len(urls) != 1 {
		t.Fatalf("got %d URLs, want 1", len(urls))
	}
	if urls[0] != "http://example.com/magic-page" {
		t.Errorf("url[0] = %q, want %q", urls[0], "http://example.com/magic-page")
	}
}
