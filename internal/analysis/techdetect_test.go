package analysis

import (
	"net/http"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestDetectTechnologies_Empty(t *testing.T) {
	result := DetectTechnologies(nil)
	if result != nil {
		t.Errorf("expected nil for empty pages, got %+v", result)
	}
}

func TestDetectTechnologies_WordPress(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><head><meta name="generator" content="WordPress 6.4"></head><body></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "WordPress")
	if found == nil {
		t.Fatalf("expected WordPress detection, got %+v", techs)
	}
	if found.Category != "cms" {
		t.Errorf("expected category cms, got %s", found.Category)
	}
}

func TestDetectTechnologies_WordPressAssets(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>plain page</body></html>`),
		Assets:      []string{"https://example.com/wp-content/themes/style.css"},
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "WordPress")
	if found == nil {
		t.Fatalf("expected WordPress detection via asset, got %+v", techs)
	}
}

func TestDetectTechnologies_NextJS(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body><script id="__NEXT_DATA__">{}</script></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Next.js")
	if found == nil {
		t.Fatalf("expected Next.js detection, got %+v", techs)
	}
}

func TestDetectTechnologies_NextJSAssets(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>plain</body></html>`),
		Assets:      []string{"https://example.com/_next/static/chunks/main.js"},
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Next.js")
	if found == nil {
		t.Fatalf("expected Next.js detection via asset, got %+v", techs)
	}
}

func TestDetectTechnologies_Hugo(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><head><meta name="generator" content="Hugo 0.120.0"></head><body></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Hugo")
	if found == nil {
		t.Fatalf("expected Hugo detection, got %+v", techs)
	}
	if found.Category != "cms" {
		t.Errorf("expected category cms, got %s", found.Category)
	}
}

func TestDetectTechnologies_Astro(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><head><meta name="generator" content="Astro v4.0"></head><body></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Astro")
	if found == nil {
		t.Fatalf("expected Astro detection, got %+v", techs)
	}
}

func TestDetectTechnologies_Ghost(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><head><meta name="generator" content="Ghost 5.0"></head><body></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Ghost")
	if found == nil {
		t.Fatalf("expected Ghost detection, got %+v", techs)
	}
}

func TestDetectTechnologies_Nuxt(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body><script>window.__NUXT__={}</script></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Nuxt")
	if found == nil {
		t.Fatalf("expected Nuxt detection, got %+v", techs)
	}
}

func TestDetectTechnologies_NuxtAssets(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>plain</body></html>`),
		Assets:      []string{"https://example.com/_nuxt/entry.js"},
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Nuxt")
	if found == nil {
		t.Fatalf("expected Nuxt detection via asset, got %+v", techs)
	}
}

func TestDetectTechnologies_GoogleAnalytics(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body><script src="https://www.google-analytics.com/analytics.js"></script></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Google Analytics")
	if found == nil {
		t.Fatalf("expected Google Analytics detection, got %+v", techs)
	}
	if found.Category != "analytics" {
		t.Errorf("expected category analytics, got %s", found.Category)
	}
}

func TestDetectTechnologies_GTM(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body><script src="https://www.googletagmanager.com/gtm.js?id=GTM-XXXX"></script></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Google Tag Manager")
	if found == nil {
		t.Fatalf("expected GTM detection, got %+v", techs)
	}
}

func TestDetectTechnologies_Plausible(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body><script defer data-domain="example.com" src="https://plausible.io/js/script.js"></script></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Plausible")
	if found == nil {
		t.Fatalf("expected Plausible detection, got %+v", techs)
	}
}

func TestDetectTechnologies_Fathom(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body><script src="https://cdn.usefathom.com/script.js"></script></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Fathom")
	if found == nil {
		t.Fatalf("expected Fathom detection, got %+v", techs)
	}
}

func TestDetectTechnologies_Cloudflare(t *testing.T) {
	headers := http.Header{}
	headers.Set("Cf-Ray", "abc123-LAX")

	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>page</body></html>`),
		Headers:     headers,
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Cloudflare")
	if found == nil {
		t.Fatalf("expected Cloudflare detection, got %+v", techs)
	}
	if found.Category != "cdn" {
		t.Errorf("expected category cdn, got %s", found.Category)
	}
}

func TestDetectTechnologies_CloudflareServerHeader(t *testing.T) {
	headers := http.Header{}
	headers.Set("Server", "cloudflare")

	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>page</body></html>`),
		Headers:     headers,
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Cloudflare")
	if found == nil {
		t.Fatalf("expected Cloudflare via server header, got %+v", techs)
	}
}

func TestDetectTechnologies_Vercel(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Vercel-Id", "iad1::abcdef-1234567890")

	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>page</body></html>`),
		Headers:     headers,
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Vercel")
	if found == nil {
		t.Fatalf("expected Vercel detection, got %+v", techs)
	}
}

func TestDetectTechnologies_Netlify(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Nf-Request-Id", "01234567-89ab-cdef")

	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>page</body></html>`),
		Headers:     headers,
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Netlify")
	if found == nil {
		t.Fatalf("expected Netlify detection, got %+v", techs)
	}
}

func TestDetectTechnologies_AWSCloudFront(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Amz-Cf-Id", "some-id-value")

	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>page</body></html>`),
		Headers:     headers,
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "AWS CloudFront")
	if found == nil {
		t.Fatalf("expected AWS CloudFront detection, got %+v", techs)
	}
}

func TestDetectTechnologies_Fastly(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Served-By", "cache-lax17523-LAX")

	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>page</body></html>`),
		Headers:     headers,
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Fastly")
	if found == nil {
		t.Fatalf("expected Fastly detection, got %+v", techs)
	}
}

func TestDetectTechnologies_React(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body><div id="root" data-reactroot="">content</div></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "React")
	if found == nil {
		t.Fatalf("expected React detection, got %+v", techs)
	}
	if found.Category != "framework" {
		t.Errorf("expected category framework, got %s", found.Category)
	}
}

func TestDetectTechnologies_Vue(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body><div data-v-12345abc class="app">content</div></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Vue")
	if found == nil {
		t.Fatalf("expected Vue detection, got %+v", techs)
	}
}

func TestDetectTechnologies_TailwindCSS(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>page</body></html>`),
		Assets:      []string{"https://cdn.example.com/tailwindcss/3.4.0/tailwind.min.css"},
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Tailwind CSS")
	if found == nil {
		t.Fatalf("expected Tailwind CSS detection, got %+v", techs)
	}
}

func TestDetectTechnologies_Bootstrap(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>page</body></html>`),
		Assets:      []string{"https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css"},
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Bootstrap")
	if found == nil {
		t.Fatalf("expected Bootstrap detection, got %+v", techs)
	}
}

func TestDetectTechnologies_Deduplication(t *testing.T) {
	// WordPress detected via both body and assets should appear only once.
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><head><meta name="generator" content="WordPress 6.4"></head><body></body></html>`),
		Assets:      []string{"https://example.com/wp-content/themes/style.css"},
	}}

	techs := DetectTechnologies(pages)
	count := 0
	for _, tech := range techs {
		if tech.Name == "WordPress" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected WordPress once, got %d times", count)
	}
}

func TestDetectTechnologies_CDNFromMultiplePages(t *testing.T) {
	// CDN detected on second page's headers.
	h := http.Header{}
	h.Set("Cf-Ray", "abc123-LAX")

	pages := []*model.Page{
		{
			URL:         "https://example.com",
			ContentType: "text/html",
			Body:        []byte(`<html><body>page</body></html>`),
		},
		{
			URL:         "https://example.com/about",
			ContentType: "text/html",
			Body:        []byte(`<html><body>about</body></html>`),
			Headers:     h,
		},
	}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Cloudflare")
	if found == nil {
		t.Fatalf("expected Cloudflare from second page headers, got %+v", techs)
	}
}

func TestDetectTechnologies_MultipleTechnologies(t *testing.T) {
	headers := http.Header{}
	headers.Set("Server", "Vercel")

	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body: []byte(`<html>
			<head><meta name="generator" content="Astro v4.0"></head>
			<body>
				<div data-reactroot="">app</div>
				<script src="https://plausible.io/js/script.js"></script>
			</body>
		</html>`),
		Assets:  []string{"https://example.com/tailwindcss.min.css"},
		Headers: headers,
	}}

	techs := DetectTechnologies(pages)

	for _, want := range []string{"Astro", "React", "Plausible", "Tailwind CSS", "Vercel"} {
		if findTech(techs, want) == nil {
			t.Errorf("expected %s in detections, got %+v", want, techs)
		}
	}
}

func TestDetectTechnologies_SortedByCategoryThenName(t *testing.T) {
	headers := http.Header{}
	headers.Set("Cf-Ray", "abc")

	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body: []byte(`<html><body>
			<script src="https://plausible.io/js/script.js"></script>
			<script src="https://www.google-analytics.com/analytics.js"></script>
			<div data-reactroot="">app</div>
		</body></html>`),
		Assets:  []string{"https://cdn.example.com/bootstrap.min.css"},
		Headers: headers,
	}}

	techs := DetectTechnologies(pages)
	for i := 1; i < len(techs); i++ {
		prev, curr := techs[i-1], techs[i]
		if prev.Category > curr.Category {
			t.Errorf("not sorted by category: %s (%s) before %s (%s)", prev.Name, prev.Category, curr.Name, curr.Category)
		}
		if prev.Category == curr.Category && prev.Name > curr.Name {
			t.Errorf("not sorted by name within category: %s before %s in %s", prev.Name, curr.Name, prev.Category)
		}
	}
}

func TestDetectTechnologies_NoDetections(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body><p>A simple page with no frameworks.</p></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	if len(techs) != 0 {
		t.Errorf("expected no detections, got %+v", techs)
	}
}

func TestDetectTechnologies_NilHeaders(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body>page</body></html>`),
		Headers:     nil,
	}}

	// Should not panic with nil headers.
	techs := DetectTechnologies(pages)
	_ = techs
}

func TestDetectTechnologies_CaseInsensitiveBody(t *testing.T) {
	// Generator tag with different casing.
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><head><META NAME="Generator" CONTENT="WordPress 6.4"></head><body></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "WordPress")
	if found == nil {
		t.Fatalf("expected case-insensitive WordPress detection, got %+v", techs)
	}
}

func TestDetectTechnologies_Gatsby(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com",
		ContentType: "text/html",
		Body:        []byte(`<html><body><div id="___gatsby" class="gatsby-focus-wrapper">content</div></body></html>`),
	}}

	techs := DetectTechnologies(pages)
	found := findTech(techs, "Gatsby")
	if found == nil {
		t.Fatalf("expected Gatsby detection, got %+v", techs)
	}
}

// findTech is a test helper that searches for a technology by name.
func findTech(techs []Technology, name string) *Technology {
	for i := range techs {
		if techs[i].Name == name {
			return &techs[i]
		}
	}
	return nil
}
