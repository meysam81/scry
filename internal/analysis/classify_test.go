package analysis

import (
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestClassifyPages_Homepage(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/",
		Depth: 0,
	}}
	result := ClassifyPages(pages)
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Class != ClassHomepage {
		t.Errorf("expected homepage, got %s", result[0].Class)
	}
	if result[0].Score != 0.9 {
		t.Errorf("expected score 0.9, got %f", result[0].Score)
	}
}

func TestClassifyPages_HomepageNoTrailingSlash(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com",
		Depth: 0,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassHomepage {
		t.Errorf("expected homepage for root without slash, got %s", result[0].Class)
	}
}

func TestClassifyPages_BlogURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/blog/my-post",
		Depth: 2,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassBlog {
		t.Errorf("expected blog, got %s", result[0].Class)
	}
	if result[0].Score != 0.7 {
		t.Errorf("expected score 0.7, got %f", result[0].Score)
	}
}

func TestClassifyPages_BlogPostURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/post/2024/hello-world",
		Depth: 3,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassBlog {
		t.Errorf("expected blog for /post/ URL, got %s", result[0].Class)
	}
}

func TestClassifyPages_BlogArticleURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/article/some-article",
		Depth: 2,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassBlog {
		t.Errorf("expected blog for /article/ URL, got %s", result[0].Class)
	}
}

func TestClassifyPages_BlogByArticleTag(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/news/story",
		Depth: 2,
		Body:  []byte(`<html><body><article><h1>News Story</h1><p>Content here.</p></article></body></html>`),
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassBlog {
		t.Errorf("expected blog for page with <article> tag, got %s", result[0].Class)
	}
	if result[0].Score != 0.3 {
		t.Errorf("expected score 0.3 for article-tag detection, got %f", result[0].Score)
	}
}

func TestClassifyPages_ProductURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/product/widget-123",
		Depth: 2,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassProduct {
		t.Errorf("expected product, got %s", result[0].Class)
	}
	if result[0].Score != 0.7 {
		t.Errorf("expected score 0.7, got %f", result[0].Score)
	}
}

func TestClassifyPages_ProductShopURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/shop/item-456",
		Depth: 2,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassProduct {
		t.Errorf("expected product for /shop/ URL, got %s", result[0].Class)
	}
}

func TestClassifyPages_ProductItemURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/item/gadget-789",
		Depth: 2,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassProduct {
		t.Errorf("expected product for /item/ URL, got %s", result[0].Class)
	}
}

func TestClassifyPages_ProductStructuredData(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/widgets/cool-widget",
		Depth: 2,
		Body:  []byte(`<html><body><script type="application/ld+json">{"@type":"Product","name":"Widget"}</script></body></html>`),
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassProduct {
		t.Errorf("expected product for JSON-LD Product, got %s", result[0].Class)
	}
	if result[0].Score != 0.5 {
		t.Errorf("expected score 0.5, got %f", result[0].Score)
	}
}

func TestClassifyPages_CategoryURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/category/electronics",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassCategory {
		t.Errorf("expected category, got %s", result[0].Class)
	}
}

func TestClassifyPages_TagURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/tag/golang",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassCategory {
		t.Errorf("expected category for /tag/ URL, got %s", result[0].Class)
	}
}

func TestClassifyPages_CollectionURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/collection/summer-2024",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassCategory {
		t.Errorf("expected category for /collection/ URL, got %s", result[0].Class)
	}
}

func TestClassifyPages_ContactURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/contact",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassContact {
		t.Errorf("expected contact, got %s", result[0].Class)
	}
	if result[0].Score != 0.8 {
		t.Errorf("expected score 0.8, got %f", result[0].Score)
	}
}

func TestClassifyPages_ContactUsURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/contact-us",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassContact {
		t.Errorf("expected contact for /contact-us, got %s", result[0].Class)
	}
}

func TestClassifyPages_AboutURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/about",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassAbout {
		t.Errorf("expected about, got %s", result[0].Class)
	}
}

func TestClassifyPages_LegalPrivacy(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/privacy",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassLegal {
		t.Errorf("expected legal for /privacy, got %s", result[0].Class)
	}
}

func TestClassifyPages_LegalTerms(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/terms",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassLegal {
		t.Errorf("expected legal for /terms, got %s", result[0].Class)
	}
}

func TestClassifyPages_LegalCookie(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/cookie-policy",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassLegal {
		t.Errorf("expected legal for /cookie-policy, got %s", result[0].Class)
	}
}

func TestClassifyPages_LegalURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/legal",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassLegal {
		t.Errorf("expected legal for /legal, got %s", result[0].Class)
	}
}

func TestClassifyPages_APIURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/api/v1/users",
		Depth: 2,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassAPI {
		t.Errorf("expected api, got %s", result[0].Class)
	}
	if result[0].Score != 0.9 {
		t.Errorf("expected score 0.9, got %f", result[0].Score)
	}
}

func TestClassifyPages_APIContentType(t *testing.T) {
	pages := []*model.Page{{
		URL:         "https://example.com/data",
		ContentType: "application/json",
		Depth:       1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassAPI {
		t.Errorf("expected api for application/json content type, got %s", result[0].Class)
	}
	if result[0].Score != 0.8 {
		t.Errorf("expected score 0.8, got %f", result[0].Score)
	}
}

func TestClassifyPages_Other(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/random-page",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassOther {
		t.Errorf("expected other, got %s", result[0].Class)
	}
	if result[0].Score != 0 {
		t.Errorf("expected score 0, got %f", result[0].Score)
	}
}

func TestClassifyPages_MultiplePages(t *testing.T) {
	pages := []*model.Page{
		{URL: "https://example.com/", Depth: 0},
		{URL: "https://example.com/blog/post-1", Depth: 2},
		{URL: "https://example.com/about", Depth: 1},
		{URL: "https://example.com/privacy", Depth: 1},
		{URL: "https://example.com/random", Depth: 1},
	}

	result := ClassifyPages(pages)
	if len(result) != 5 {
		t.Fatalf("expected 5 results, got %d", len(result))
	}

	expected := []PageClass{ClassHomepage, ClassBlog, ClassAbout, ClassLegal, ClassOther}
	for i, want := range expected {
		if result[i].Class != want {
			t.Errorf("page %d (%s): expected %s, got %s", i, pages[i].URL, want, result[i].Class)
		}
	}
}

func TestClassifyPages_EmptyInput(t *testing.T) {
	result := ClassifyPages(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 results for nil input, got %d", len(result))
	}
}

func TestClassifyPages_HighestScoreWins(t *testing.T) {
	// A page at depth 0, root path, but also contains /api/ in path.
	// Homepage rule scores 0.9 and API rule also scores 0.9.
	// First rule wins (homepage is declared before API).
	pages := []*model.Page{{
		URL:   "https://example.com/",
		Depth: 0,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassHomepage {
		t.Errorf("expected homepage (first match on tie), got %s", result[0].Class)
	}
}

func TestClassifyPages_HigherScoreBeatsEarlierRule(t *testing.T) {
	// /api/blog/post matches both API (0.9) and blog (0.7).
	// API has higher score, so it wins.
	pages := []*model.Page{{
		URL:   "https://example.com/api/blog/post",
		Depth: 3,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassAPI {
		t.Errorf("expected api (higher score), got %s", result[0].Class)
	}
}

func TestClassifyPages_CaseInsensitiveURL(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/Blog/My-Post",
		Depth: 2,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassBlog {
		t.Errorf("expected blog (case insensitive), got %s", result[0].Class)
	}
}

func TestClassifyPages_URLWithQueryString(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/contact?ref=nav",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassContact {
		t.Errorf("expected contact with query string, got %s", result[0].Class)
	}
}

func TestClassifyPages_URLWithFragment(t *testing.T) {
	pages := []*model.Page{{
		URL:   "https://example.com/about#team",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class != ClassAbout {
		t.Errorf("expected about with fragment, got %s", result[0].Class)
	}
}

func TestClassifyPages_DeepHomepage(t *testing.T) {
	// Depth > 0, so not classified as homepage even if root path.
	pages := []*model.Page{{
		URL:   "https://example.com/",
		Depth: 1,
	}}
	result := ClassifyPages(pages)
	if result[0].Class == ClassHomepage {
		t.Errorf("depth > 0 should not classify as homepage")
	}
}

func TestExtractPath(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/", "/"},
		{"https://example.com", "/"},
		{"https://example.com/blog/post", "/blog/post"},
		{"https://example.com/api/v1?key=val", "/api/v1"},
		{"https://example.com/about#team", "/about"},
		{"not-a-url", "not-a-url"},
	}
	for _, tt := range tests {
		got := extractPath(tt.url)
		if got != tt.want {
			t.Errorf("extractPath(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestIsRootPath(t *testing.T) {
	if !isRootPath("/") {
		t.Error("/ should be root path")
	}
	if !isRootPath("") {
		t.Error("empty string should be root path")
	}
	if isRootPath("/about") {
		t.Error("/about should not be root path")
	}
}
