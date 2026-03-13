package audit

import (
	"context"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestDeepStructuredDataChecker_Name(t *testing.T) {
	c := NewDeepStructuredDataChecker()
	if c.Name() != "deep-structured-data" {
		t.Fatalf("expected name %q, got %q", "deep-structured-data", c.Name())
	}
}

func TestDeepStructuredDataChecker_Check(t *testing.T) {
	checker := NewDeepStructuredDataChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantCheck  string
		wantSev    model.Severity
		wantIssue  bool
		wantSubstr string
	}{
		{
			name:      "non-html page skipped",
			html:      "",
			wantIssue: false,
		},
		{
			name:      "no script tags - no issues",
			html:      `<html><head></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "valid JSON-LD with known type and all fields",
			html:      `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"Article","headline":"Test","datePublished":"2024-01-01","author":"Author"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "missing @type",
			html:      `<html><head><script type="application/ld+json">{"@context":"https://schema.org","name":"Test"}</script></head><body></body></html>`,
			wantCheck: "structured-data/missing-type",
			wantSev:   model.SeverityWarning,
			wantIssue: true,
		},
		{
			name:       "unknown @type",
			html:       `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"CustomThing"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/unknown-type",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "CustomThing",
		},
		{
			name:       "Article missing headline and author",
			html:       `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"Article","datePublished":"2024-01-01"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/article-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "headline",
		},
		{
			name:       "Article missing all required fields",
			html:       `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"Article"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/article-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "headline",
		},
		{
			name:       "BlogPosting missing fields (uses Article check)",
			html:       `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"BlogPosting"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/article-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "headline",
		},
		{
			name:       "Product missing description",
			html:       `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"Product","name":"Widget"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/product-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "description",
		},
		{
			name:      "Product with all fields - no issue",
			html:      `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"Product","name":"Widget","description":"A great widget"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:       "FAQPage missing mainEntity",
			html:       `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"FAQPage"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/faq-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "mainEntity",
		},
		{
			name:      "FAQPage with mainEntity - no issue",
			html:      `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"FAQPage","mainEntity":[{"@type":"Question","name":"Q?","acceptedAnswer":{"@type":"Answer","text":"A."}}]}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:       "BreadcrumbList missing itemListElement",
			html:       `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"BreadcrumbList"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/breadcrumb-missing-fields",
			wantSev:    model.SeverityInfo,
			wantIssue:  true,
			wantSubstr: "itemListElement",
		},
		{
			name:      "BreadcrumbList with itemListElement - no issue",
			html:      `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"BreadcrumbList","itemListElement":[{"@type":"ListItem","position":1,"name":"Home","item":"https://example.com"}]}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "known type without required field rules (Organization)",
			html:      `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"Organization"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "malformed JSON - no issues from this checker",
			html:      `<html><head><script type="application/ld+json">{not valid json</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "non-ld+json script tag ignored",
			html:      `<html><head><script type="text/javascript">var x = 1;</script></head><body></body></html>`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			if tt.name == "non-html page skipped" {
				page.ContentType = "application/json"
			}

			issues := checker.Check(ctx, page)

			if !tt.wantIssue {
				for _, iss := range issues {
					if tt.wantCheck != "" && iss.CheckName == tt.wantCheck {
						t.Fatalf("did not expect issue %s, got %+v", tt.wantCheck, iss)
					}
				}
				return
			}

			found := false
			for _, iss := range issues {
				if iss.CheckName == tt.wantCheck {
					found = true
					if iss.Severity != tt.wantSev {
						t.Errorf("expected severity %s, got %s", tt.wantSev, iss.Severity)
					}
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
					}
				}
			}
			if !found {
				t.Errorf("expected issue %s not found in %+v", tt.wantCheck, issues)
			}
		})
	}
}

func TestDeepStructuredDataChecker_ArrayJSONLD(t *testing.T) {
	checker := NewDeepStructuredDataChecker()
	ctx := context.Background()

	page := htmlPage(`<html><head><script type="application/ld+json">[
		{"@context":"https://schema.org","@type":"Article"},
		{"@context":"https://schema.org","@type":"Product"}
	]</script></head><body></body></html>`)

	issues := checker.Check(ctx, page)

	// Article missing fields
	foundArticle := false
	foundProduct := false
	for _, iss := range issues {
		if iss.CheckName == "structured-data/article-missing-fields" {
			foundArticle = true
		}
		if iss.CheckName == "structured-data/product-missing-fields" {
			foundProduct = true
		}
	}
	if !foundArticle {
		t.Error("expected structured-data/article-missing-fields for array JSON-LD")
	}
	if !foundProduct {
		t.Error("expected structured-data/product-missing-fields for array JSON-LD")
	}
}

func TestDeepStructuredDataChecker_GraphJSONLD(t *testing.T) {
	checker := NewDeepStructuredDataChecker()
	ctx := context.Background()

	page := htmlPage(`<html><head><script type="application/ld+json">{
		"@context":"https://schema.org",
		"@graph":[
			{"@type":"Article"},
			{"@type":"BreadcrumbList"}
		]
	}</script></head><body></body></html>`)

	issues := checker.Check(ctx, page)

	foundArticle := false
	foundBreadcrumb := false
	for _, iss := range issues {
		if iss.CheckName == "structured-data/article-missing-fields" {
			foundArticle = true
		}
		if iss.CheckName == "structured-data/breadcrumb-missing-fields" {
			foundBreadcrumb = true
		}
	}
	if !foundArticle {
		t.Error("expected structured-data/article-missing-fields for @graph JSON-LD")
	}
	if !foundBreadcrumb {
		t.Error("expected structured-data/breadcrumb-missing-fields for @graph JSON-LD")
	}
}

func TestDeepStructuredDataChecker_MultipleScriptTags(t *testing.T) {
	checker := NewDeepStructuredDataChecker()
	ctx := context.Background()

	page := htmlPage(`<html><head>
		<script type="application/ld+json">{"@context":"https://schema.org","@type":"Article"}</script>
		<script type="application/ld+json">{"@context":"https://schema.org","@type":"Product"}</script>
	</head><body></body></html>`)

	issues := checker.Check(ctx, page)

	foundArticle := false
	foundProduct := false
	for _, iss := range issues {
		if iss.CheckName == "structured-data/article-missing-fields" {
			foundArticle = true
		}
		if iss.CheckName == "structured-data/product-missing-fields" {
			foundProduct = true
		}
	}
	if !foundArticle {
		t.Error("expected structured-data/article-missing-fields from first script tag")
	}
	if !foundProduct {
		t.Error("expected structured-data/product-missing-fields from second script tag")
	}
}

func TestDeepStructuredDataChecker_MultiTypeArray(t *testing.T) {
	checker := NewDeepStructuredDataChecker()
	ctx := context.Background()

	// @type can be an array of types.
	page := htmlPage(`<html><head><script type="application/ld+json">{
		"@context":"https://schema.org",
		"@type":["Article","BlogPosting"]
	}</script></head><body></body></html>`)

	issues := checker.Check(ctx, page)

	// Both Article and BlogPosting map to article-missing-fields.
	count := 0
	for _, iss := range issues {
		if iss.CheckName == "structured-data/article-missing-fields" {
			count++
		}
	}
	// We expect issues from both type entries since fields are missing.
	if count < 1 {
		t.Errorf("expected at least 1 article-missing-fields issue for multi-type, got %d; issues=%+v", count, issues)
	}
}

func TestDeepStructuredDataChecker_AllKnownTypes(t *testing.T) {
	checker := NewDeepStructuredDataChecker()
	ctx := context.Background()

	knownTypes := []string{
		"Article", "BlogPosting", "Product", "FAQPage", "BreadcrumbList",
		"Organization", "Person", "WebPage", "WebSite", "LocalBusiness",
		"Event", "Recipe", "HowTo", "VideoObject",
	}

	for _, typeName := range knownTypes {
		t.Run(typeName, func(t *testing.T) {
			html := `<html><head><script type="application/ld+json">{"@context":"https://schema.org","@type":"` + typeName + `"}</script></head><body></body></html>`
			page := htmlPage(html)
			issues := checker.Check(ctx, page)

			// Should NOT produce an unknown-type issue for any known type.
			for _, iss := range issues {
				if iss.CheckName == "structured-data/unknown-type" {
					t.Errorf("type %q should be recognised, but got unknown-type issue", typeName)
				}
			}
		})
	}
}

func TestDeepStructuredDataChecker_ArticleWithAllFields(t *testing.T) {
	checker := NewDeepStructuredDataChecker()
	ctx := context.Background()

	page := htmlPage(`<html><head><script type="application/ld+json">{
		"@context":"https://schema.org",
		"@type":"Article",
		"headline":"Test Article",
		"datePublished":"2024-01-15",
		"author":{"@type":"Person","name":"Author"}
	}</script></head><body></body></html>`)

	issues := checker.Check(ctx, page)
	for _, iss := range issues {
		if iss.CheckName == "structured-data/article-missing-fields" {
			t.Errorf("did not expect article-missing-fields when all fields present, got %+v", iss)
		}
	}
}

// ---------------------------------------------------------------------------
// Date validation tests
// ---------------------------------------------------------------------------

func TestDeepStructuredDataChecker_InvalidDateFormat(t *testing.T) {
	checker := NewDeepStructuredDataChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantIssue  bool
		wantSubstr string
	}{
		{
			name:      "valid date YYYY-MM-DD",
			html:      `<html><head><script type="application/ld+json">{"@type":"Article","headline":"T","author":"A","datePublished":"2024-01-15"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "valid date with time",
			html:      `<html><head><script type="application/ld+json">{"@type":"Article","headline":"T","author":"A","datePublished":"2024-01-15T10:30:00Z"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "valid date with timezone offset",
			html:      `<html><head><script type="application/ld+json">{"@type":"Article","headline":"T","author":"A","datePublished":"2024-01-15T10:30:00+02:00"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:       "plain text instead of date",
			html:       `<html><head><script type="application/ld+json">{"@type":"Article","headline":"T","author":"A","datePublished":"January 15, 2024"}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "datePublished",
		},
		{
			name:       "empty string date",
			html:       `<html><head><script type="application/ld+json">{"@type":"Article","headline":"T","author":"A","datePublished":""}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "datePublished",
		},
		{
			name:       "numeric date value",
			html:       `<html><head><script type="application/ld+json">{"@type":"Article","headline":"T","author":"A","datePublished":1705276800}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "non-string",
		},
		{
			name:       "dateCreated with invalid format",
			html:       `<html><head><script type="application/ld+json">{"@type":"WebPage","dateCreated":"last week"}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "dateCreated",
		},
		{
			name:       "dateModified with slash format",
			html:       `<html><head><script type="application/ld+json">{"@type":"WebPage","dateModified":"01/15/2024"}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "dateModified",
		},
		{
			name:      "valid dateModified",
			html:      `<html><head><script type="application/ld+json">{"@type":"WebPage","dateModified":"2024-06-01"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:       "startDate with invalid format",
			html:       `<html><head><script type="application/ld+json">{"@type":"Event","name":"Conf","location":"NYC","startDate":"tomorrow"}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "startDate",
		},
		{
			name:      "valid startDate and endDate",
			html:      `<html><head><script type="application/ld+json">{"@type":"Event","name":"Conf","location":"NYC","startDate":"2024-09-01","endDate":"2024-09-03"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:       "uploadDate with invalid format",
			html:       `<html><head><script type="application/ld+json">{"@type":"VideoObject","name":"V","description":"D","thumbnailUrl":"https://example.com/thumb.jpg","uploadDate":"last year"}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "uploadDate",
		},
		{
			name:       "whitespace-only date",
			html:       `<html><head><script type="application/ld+json">{"@type":"Article","headline":"T","author":"A","datePublished":"   "}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "datePublished",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)

			found := false
			for _, iss := range issues {
				if iss.CheckName == "structured-data/invalid-date-format" {
					found = true
					if iss.Severity != model.SeverityWarning {
						t.Errorf("expected severity warning, got %s", iss.Severity)
					}
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
					}
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected structured-data/invalid-date-format issue not found in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect structured-data/invalid-date-format issue, but found one in %+v", issues)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// URL field validation tests
// ---------------------------------------------------------------------------

func TestDeepStructuredDataChecker_InvalidURLField(t *testing.T) {
	checker := NewDeepStructuredDataChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantIssue  bool
		wantSubstr string
	}{
		{
			name:      "valid https URL",
			html:      `<html><head><script type="application/ld+json">{"@type":"Organization","url":"https://example.com"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "valid http URL",
			html:      `<html><head><script type="application/ld+json">{"@type":"Organization","url":"http://example.com"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "valid relative URL",
			html:      `<html><head><script type="application/ld+json">{"@type":"Organization","url":"/about"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:       "plain text in url field",
			html:       `<html><head><script type="application/ld+json">{"@type":"Organization","url":"just some text"}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "url",
		},
		{
			name:       "empty string url",
			html:       `<html><head><script type="application/ld+json">{"@type":"Organization","url":""}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "url",
		},
		{
			name:       "plain text in image field",
			html:       `<html><head><script type="application/ld+json">{"@type":"Product","name":"W","description":"D","image":"not a url"}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "image",
		},
		{
			name:      "valid image URL",
			html:      `<html><head><script type="application/ld+json">{"@type":"Product","name":"W","description":"D","image":"https://example.com/img.jpg"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "image as nested object (ImageObject) - no issue",
			html:      `<html><head><script type="application/ld+json">{"@type":"Product","name":"W","description":"D","image":{"@type":"ImageObject","url":"https://example.com/img.jpg"}}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:       "logo with plain text",
			html:       `<html><head><script type="application/ld+json">{"@type":"Organization","logo":"my logo"}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "logo",
		},
		{
			name:      "valid logo URL",
			html:      `<html><head><script type="application/ld+json">{"@type":"Organization","logo":"https://example.com/logo.png"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:      "sameAs with valid URL array",
			html:      `<html><head><script type="application/ld+json">{"@type":"Organization","sameAs":["https://twitter.com/ex","https://facebook.com/ex"]}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:       "sameAs with invalid URL in array",
			html:       `<html><head><script type="application/ld+json">{"@type":"Organization","sameAs":["https://twitter.com/ex","not a url"]}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "sameAs",
		},
		{
			name:       "thumbnailUrl with plain text",
			html:       `<html><head><script type="application/ld+json">{"@type":"VideoObject","name":"V","description":"D","thumbnailUrl":"thumbnail","uploadDate":"2024-01-01"}</script></head><body></body></html>`,
			wantIssue:  true,
			wantSubstr: "thumbnailUrl",
		},
		{
			name:      "valid thumbnailUrl",
			html:      `<html><head><script type="application/ld+json">{"@type":"VideoObject","name":"V","description":"D","thumbnailUrl":"https://example.com/thumb.jpg","uploadDate":"2024-01-01"}</script></head><body></body></html>`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)

			found := false
			for _, iss := range issues {
				if iss.CheckName == "structured-data/invalid-url-field" {
					found = true
					if iss.Severity != model.SeverityWarning {
						t.Errorf("expected severity warning, got %s", iss.Severity)
					}
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
					}
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected structured-data/invalid-url-field issue not found in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect structured-data/invalid-url-field issue, but found one in %+v", issues)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Microdata detection tests
// ---------------------------------------------------------------------------

func TestDeepStructuredDataChecker_MicrodataDetected(t *testing.T) {
	checker := NewDeepStructuredDataChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantIssue  bool
		wantSubstr string
	}{
		{
			name:       "itemscope attribute present",
			html:       `<html><head></head><body><div itemscope itemtype="https://schema.org/Product"><span itemprop="name">Widget</span></div></body></html>`,
			wantIssue:  true,
			wantSubstr: "itemscope",
		},
		{
			name:       "itemprop attribute present without itemscope",
			html:       `<html><head></head><body><span itemprop="name">Widget</span></body></html>`,
			wantIssue:  true,
			wantSubstr: "itemprop",
		},
		{
			name:       "itemtype attribute present",
			html:       `<html><head></head><body><div itemtype="https://schema.org/Product"></div></body></html>`,
			wantIssue:  true,
			wantSubstr: "itemtype",
		},
		{
			name:      "no microdata attributes",
			html:      `<html><head></head><body><div class="product"><span>Widget</span></div></body></html>`,
			wantIssue: false,
		},
		{
			name:      "no microdata - only JSON-LD",
			html:      `<html><head><script type="application/ld+json">{"@type":"Product","name":"W","description":"D"}</script></head><body></body></html>`,
			wantIssue: false,
		},
		{
			name:       "microdata alongside JSON-LD",
			html:       `<html><head><script type="application/ld+json">{"@type":"Product","name":"W","description":"D"}</script></head><body><div itemscope itemtype="https://schema.org/Product"></div></body></html>`,
			wantIssue:  true,
			wantSubstr: "microdata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)

			found := false
			for _, iss := range issues {
				if iss.CheckName == "structured-data/microdata-detected" {
					found = true
					if iss.Severity != model.SeverityInfo {
						t.Errorf("expected severity info, got %s", iss.Severity)
					}
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
					}
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected structured-data/microdata-detected issue not found in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect structured-data/microdata-detected issue, but found one in %+v", issues)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// New type required field tests (Event, Recipe, VideoObject, LocalBusiness)
// ---------------------------------------------------------------------------

func TestDeepStructuredDataChecker_NewTypeRequiredFields(t *testing.T) {
	checker := NewDeepStructuredDataChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantCheck  string
		wantSev    model.Severity
		wantIssue  bool
		wantSubstr string
	}{
		// Event tests
		{
			name:       "Event missing all required fields",
			html:       `<html><head><script type="application/ld+json">{"@type":"Event"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/event-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "name",
		},
		{
			name:       "Event missing startDate and location",
			html:       `<html><head><script type="application/ld+json">{"@type":"Event","name":"Conference"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/event-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "startDate",
		},
		{
			name:      "Event with all required fields",
			html:      `<html><head><script type="application/ld+json">{"@type":"Event","name":"Conference","startDate":"2024-09-01","location":"Convention Center"}</script></head><body></body></html>`,
			wantCheck: "structured-data/event-missing-fields",
			wantIssue: false,
		},
		// Recipe tests
		{
			name:       "Recipe missing all required fields",
			html:       `<html><head><script type="application/ld+json">{"@type":"Recipe"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/recipe-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "name",
		},
		{
			name:       "Recipe missing image and recipeIngredient",
			html:       `<html><head><script type="application/ld+json">{"@type":"Recipe","name":"Cake"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/recipe-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "image",
		},
		{
			name:      "Recipe with all required fields",
			html:      `<html><head><script type="application/ld+json">{"@type":"Recipe","name":"Cake","image":"https://example.com/cake.jpg","recipeIngredient":["flour","sugar"]}</script></head><body></body></html>`,
			wantCheck: "structured-data/recipe-missing-fields",
			wantIssue: false,
		},
		// VideoObject tests
		{
			name:       "VideoObject missing all required fields",
			html:       `<html><head><script type="application/ld+json">{"@type":"VideoObject"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/video-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "name",
		},
		{
			name:       "VideoObject missing thumbnailUrl and uploadDate",
			html:       `<html><head><script type="application/ld+json">{"@type":"VideoObject","name":"Demo","description":"A demo video"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/video-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "thumbnailUrl",
		},
		{
			name:      "VideoObject with all required fields",
			html:      `<html><head><script type="application/ld+json">{"@type":"VideoObject","name":"Demo","description":"A demo","thumbnailUrl":"https://example.com/thumb.jpg","uploadDate":"2024-01-01"}</script></head><body></body></html>`,
			wantCheck: "structured-data/video-missing-fields",
			wantIssue: false,
		},
		// LocalBusiness tests
		{
			name:       "LocalBusiness missing all required fields",
			html:       `<html><head><script type="application/ld+json">{"@type":"LocalBusiness"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/localbusiness-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "name",
		},
		{
			name:       "LocalBusiness missing address and telephone",
			html:       `<html><head><script type="application/ld+json">{"@type":"LocalBusiness","name":"Acme Corp"}</script></head><body></body></html>`,
			wantCheck:  "structured-data/localbusiness-missing-fields",
			wantSev:    model.SeverityWarning,
			wantIssue:  true,
			wantSubstr: "address",
		},
		{
			name:      "LocalBusiness with all required fields",
			html:      `<html><head><script type="application/ld+json">{"@type":"LocalBusiness","name":"Acme Corp","address":"123 Main St","telephone":"+1-555-0100"}</script></head><body></body></html>`,
			wantCheck: "structured-data/localbusiness-missing-fields",
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)

			if !tt.wantIssue {
				for _, iss := range issues {
					if iss.CheckName == tt.wantCheck {
						t.Fatalf("did not expect issue %s, got %+v", tt.wantCheck, iss)
					}
				}
				return
			}

			found := false
			for _, iss := range issues {
				if iss.CheckName == tt.wantCheck {
					found = true
					if iss.Severity != tt.wantSev {
						t.Errorf("expected severity %s, got %s", tt.wantSev, iss.Severity)
					}
					if tt.wantSubstr != "" && !strings.Contains(iss.Message, tt.wantSubstr) {
						t.Errorf("expected message containing %q, got %q", tt.wantSubstr, iss.Message)
					}
				}
			}
			if !found {
				t.Errorf("expected issue %s not found in %+v", tt.wantCheck, issues)
			}
		})
	}
}
