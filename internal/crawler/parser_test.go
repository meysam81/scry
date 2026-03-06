package crawler

import (
	"net/url"
	"testing"
)

func mustParse(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}

func TestParseHTML_Links(t *testing.T) {
	base := mustParse("http://example.com/")
	body := []byte(`<html><body>
		<a href="/page1">Page 1</a>
		<a href="http://example.com/page2">Page 2</a>
		<a href="/page3#section">Page 3</a>
	</body></html>`)

	links, _ := ParseHTML(base, body)
	if len(links) != 3 {
		t.Fatalf("got %d links, want 3", len(links))
	}

	expected := []string{
		"http://example.com/page1",
		"http://example.com/page2",
		"http://example.com/page3",
	}
	for i, l := range links {
		if l != expected[i] {
			t.Errorf("link[%d] = %q, want %q", i, l, expected[i])
		}
	}
}

func TestParseHTML_Assets(t *testing.T) {
	base := mustParse("http://example.com/")
	body := []byte(`<html>
	<head>
		<link href="/style.css">
		<script src="/app.js"></script>
	</head>
	<body>
		<img src="/logo.png">
	</body></html>`)

	_, assets := ParseHTML(base, body)
	if len(assets) != 3 {
		t.Fatalf("got %d assets, want 3", len(assets))
	}

	expected := []string{
		"http://example.com/style.css",
		"http://example.com/app.js",
		"http://example.com/logo.png",
	}
	for i, a := range assets {
		if a != expected[i] {
			t.Errorf("asset[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestParseHTML_ResolvesRelative(t *testing.T) {
	base := mustParse("http://example.com/blog/")
	body := []byte(`<html><body><a href="post1">Post 1</a></body></html>`)

	links, _ := ParseHTML(base, body)
	if len(links) != 1 {
		t.Fatalf("got %d links, want 1", len(links))
	}
	if links[0] != "http://example.com/blog/post1" {
		t.Errorf("link = %q, want %q", links[0], "http://example.com/blog/post1")
	}
}

func TestParseHTML_StripsFragments(t *testing.T) {
	base := mustParse("http://example.com/")
	body := []byte(`<html><body><a href="/page#top">Top</a></body></html>`)

	links, _ := ParseHTML(base, body)
	if len(links) != 1 {
		t.Fatalf("got %d links, want 1", len(links))
	}
	if links[0] != "http://example.com/page" {
		t.Errorf("link = %q, want %q (fragment should be stripped)", links[0], "http://example.com/page")
	}
}

func TestParseHTML_SkipsSpecialSchemes(t *testing.T) {
	base := mustParse("http://example.com/")
	body := []byte(`<html><body>
		<a href="mailto:user@example.com">Email</a>
		<a href="javascript:void(0)">JS</a>
		<a href="data:text/html,hello">Data</a>
		<a href="/valid">Valid</a>
	</body></html>`)

	links, _ := ParseHTML(base, body)
	if len(links) != 1 {
		t.Fatalf("got %d links, want 1 (only the /valid link)", len(links))
	}
	if links[0] != "http://example.com/valid" {
		t.Errorf("link = %q, want %q", links[0], "http://example.com/valid")
	}
}
