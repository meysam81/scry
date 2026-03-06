package crawler

import (
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
