package analysis

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/meysam81/scry/core/model"
	"golang.org/x/net/html"
)

func htmlPage(bodyContent string) []byte {
	return []byte(fmt.Sprintf(`<!DOCTYPE html><html lang="en"><head><title>Test</title></head><body>%s</body></html>`, bodyContent))
}

func TestContentAnalyzer_Name(t *testing.T) {
	c := NewContentAnalyzer()
	if c.Name() != "content" {
		t.Errorf("unexpected name: %s", c.Name())
	}
}

func TestContentAnalyzer_SkipsNonHTML(t *testing.T) {
	c := NewContentAnalyzer()
	page := &model.Page{
		URL:         "https://example.com/image.png",
		ContentType: "image/png",
		Body:        []byte("binary"),
	}
	issues := c.Check(context.Background(), page)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for non-HTML, got %+v", issues)
	}
}

func TestContentAnalyzer_SkipsEmptyBody(t *testing.T) {
	c := NewContentAnalyzer()
	page := &model.Page{
		URL:         "https://example.com/empty",
		ContentType: "text/html",
		Body:        nil,
	}
	issues := c.Check(context.Background(), page)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for empty body, got %+v", issues)
	}
}

func TestContentAnalyzer_HeadingHierarchySkip(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		wantIssue bool
		wantMsg   string
	}{
		{
			name:      "h1 to h3 skip",
			html:      `<h1>Title</h1><h3>Subheading</h3>`,
			wantIssue: true,
			wantMsg:   "h1 to h3",
		},
		{
			name:      "h2 to h5 skip",
			html:      `<h1>Title</h1><h2>Section</h2><h5>Deep</h5>`,
			wantIssue: true,
			wantMsg:   "h2 to h5",
		},
		{
			name:      "proper hierarchy h1 h2 h3",
			html:      `<h1>Title</h1><h2>Section</h2><h3>Subsection</h3>`,
			wantIssue: false,
		},
		{
			name:      "h1 then h2",
			html:      `<h1>Title</h1><h2>Section</h2>`,
			wantIssue: false,
		},
		{
			name:      "going back up is ok",
			html:      `<h1>Title</h1><h2>Section</h2><h3>Sub</h3><h1>New Section</h1>`,
			wantIssue: false,
		},
		{
			name:      "no headings",
			html:      `<p>Just a paragraph.</p>`,
			wantIssue: false,
		},
		{
			name:      "single h1",
			html:      `<h1>Only Heading</h1>`,
			wantIssue: false,
		},
	}

	c := NewContentAnalyzer()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add a paragraph for text ratio so we don't get unrelated issues.
			body := htmlPage(tt.html + `<p>` + strings.Repeat("word ", 50) + `</p>`)
			page := &model.Page{
				URL:         "https://example.com/test",
				ContentType: "text/html",
				Body:        body,
			}
			issues := c.Check(ctx, page)

			found := false
			for _, iss := range issues {
				if iss.CheckName == "content/heading-hierarchy-skip" {
					found = true
					if iss.Severity != model.SeverityWarning {
						t.Errorf("expected warning severity, got %s", iss.Severity)
					}
					if tt.wantMsg != "" && !strings.Contains(iss.Message, tt.wantMsg) {
						t.Errorf("expected message containing %q, got %q", tt.wantMsg, iss.Message)
					}
				}
			}
			if tt.wantIssue && !found {
				t.Errorf("expected heading-hierarchy-skip issue, got %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect heading-hierarchy-skip issue")
			}
		})
	}
}

func TestContentAnalyzer_LowTextRatio(t *testing.T) {
	// Minimal text, lots of HTML boilerplate.
	body := []byte(`<!DOCTYPE html><html><head><title>Test</title>
		<style>` + strings.Repeat("body { color: red; } ", 200) + `</style>
		</head><body><p>Hi</p></body></html>`)

	c := NewContentAnalyzer()
	page := &model.Page{
		URL:         "https://example.com/sparse",
		ContentType: "text/html",
		Body:        body,
	}
	issues := c.Check(context.Background(), page)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "content/low-text-ratio" {
			found = true
			if iss.Severity != model.SeverityInfo {
				t.Errorf("expected info severity, got %s", iss.Severity)
			}
		}
	}
	if !found {
		t.Errorf("expected low-text-ratio issue")
	}
}

func TestContentAnalyzer_GoodTextRatio(t *testing.T) {
	text := strings.Repeat("content ", 200)
	body := htmlPage(fmt.Sprintf("<p>%s</p>", text))

	c := NewContentAnalyzer()
	page := &model.Page{
		URL:         "https://example.com/good",
		ContentType: "text/html",
		Body:        body,
	}
	issues := c.Check(context.Background(), page)

	for _, iss := range issues {
		if iss.CheckName == "content/low-text-ratio" {
			t.Fatalf("did not expect low-text-ratio issue, got %+v", iss)
		}
	}
}

func TestContentAnalyzer_NoParagraphs(t *testing.T) {
	body := htmlPage(`<div>Content without paragraphs</div>`)

	c := NewContentAnalyzer()
	page := &model.Page{
		URL:         "https://example.com/nop",
		ContentType: "text/html",
		Body:        body,
	}
	issues := c.Check(context.Background(), page)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "content/no-paragraphs" {
			found = true
			if iss.Severity != model.SeverityInfo {
				t.Errorf("expected info severity, got %s", iss.Severity)
			}
		}
	}
	if !found {
		t.Errorf("expected no-paragraphs issue")
	}
}

func TestContentAnalyzer_HasParagraphs(t *testing.T) {
	body := htmlPage(`<p>A nice paragraph with content.</p>`)

	c := NewContentAnalyzer()
	page := &model.Page{
		URL:         "https://example.com/hasp",
		ContentType: "text/html",
		Body:        body,
	}
	issues := c.Check(context.Background(), page)

	for _, iss := range issues {
		if iss.CheckName == "content/no-paragraphs" {
			t.Fatalf("did not expect no-paragraphs issue, got %+v", iss)
		}
	}
}

func TestContentAnalyzer_WordCountLow_ArticleTag(t *testing.T) {
	body := htmlPage(`<article><p>Short article with few words.</p></article>`)

	c := NewContentAnalyzer()
	page := &model.Page{
		URL:         "https://example.com/short-article",
		ContentType: "text/html",
		Body:        body,
	}
	issues := c.Check(context.Background(), page)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "content/word-count-low" {
			found = true
			if iss.Severity != model.SeverityInfo {
				t.Errorf("expected info severity, got %s", iss.Severity)
			}
			if !strings.Contains(iss.Message, "300") {
				t.Errorf("expected '300' in message, got %q", iss.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected word-count-low issue for article page")
	}
}

func TestContentAnalyzer_WordCountLow_BlogURL(t *testing.T) {
	body := htmlPage(`<p>Short blog post.</p>`)

	c := NewContentAnalyzer()
	page := &model.Page{
		URL:         "https://example.com/blog/short-post",
		ContentType: "text/html",
		Body:        body,
	}
	issues := c.Check(context.Background(), page)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "content/word-count-low" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected word-count-low issue for /blog/ URL")
	}
}

func TestContentAnalyzer_WordCountLow_PostURL(t *testing.T) {
	body := htmlPage(`<p>Short post.</p>`)

	c := NewContentAnalyzer()
	page := &model.Page{
		URL:         "https://example.com/post/my-article",
		ContentType: "text/html",
		Body:        body,
	}
	issues := c.Check(context.Background(), page)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "content/word-count-low" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected word-count-low issue for /post/ URL")
	}
}

func TestContentAnalyzer_WordCountOk(t *testing.T) {
	words := strings.Repeat("word ", 350)
	body := htmlPage(fmt.Sprintf(`<article><p>%s</p></article>`, words))

	c := NewContentAnalyzer()
	page := &model.Page{
		URL:         "https://example.com/long-article",
		ContentType: "text/html",
		Body:        body,
	}
	issues := c.Check(context.Background(), page)

	for _, iss := range issues {
		if iss.CheckName == "content/word-count-low" {
			t.Fatalf("did not expect word-count-low for long article, got %+v", iss)
		}
	}
}

func TestContentAnalyzer_WordCountNotArticle(t *testing.T) {
	// Short page but not an article (no <article> tag, no /blog/ or /post/ in URL).
	body := htmlPage(`<p>Short page.</p>`)

	c := NewContentAnalyzer()
	page := &model.Page{
		URL:         "https://example.com/about",
		ContentType: "text/html",
		Body:        body,
	}
	issues := c.Check(context.Background(), page)

	for _, iss := range issues {
		if iss.CheckName == "content/word-count-low" {
			t.Fatalf("non-article page should not trigger word-count-low, got %+v", iss)
		}
	}
}

func TestTextToHTMLRatio(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		wantZero bool
	}{
		{
			name:     "empty body",
			body:     nil,
			wantZero: true,
		},
		{
			name:     "all text",
			body:     []byte("just plain text"),
			wantZero: false,
		},
		{
			name:     "heavy html",
			body:     []byte(`<div class="very-long-class-name"></div>`),
			wantZero: true, // close to zero
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio := textToHTMLRatio(tt.body)
			if tt.wantZero && ratio > 0.05 {
				t.Errorf("expected near-zero ratio, got %f", ratio)
			}
			if !tt.wantZero && ratio <= 0 {
				t.Errorf("expected positive ratio, got %f", ratio)
			}
		})
	}
}

func TestIsArticlePage(t *testing.T) {
	tests := []struct {
		name string
		url  string
		body []byte
		want bool
	}{
		{
			name: "blog URL",
			url:  "https://example.com/blog/my-post",
			body: htmlPage("<p>text</p>"),
			want: true,
		},
		{
			name: "post URL",
			url:  "https://example.com/post/123",
			body: htmlPage("<p>text</p>"),
			want: true,
		},
		{
			name: "article tag",
			url:  "https://example.com/page",
			body: htmlPage("<article><p>text</p></article>"),
			want: true,
		},
		{
			name: "regular page",
			url:  "https://example.com/about",
			body: htmlPage("<p>text</p>"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := &model.Page{URL: tt.url, ContentType: "text/html", Body: tt.body}
			got := isArticlePage(page)
			if got != tt.want {
				t.Errorf("isArticlePage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHeadingLevel(t *testing.T) {
	tests := []struct {
		tag  string
		want int
	}{
		{"h1", 1},
		{"h2", 2},
		{"h6", 6},
		{"p", 0},
		{"div", 0},
		{"h0", 0},
		{"h7", 0},
	}
	for _, tt := range tests {
		node := &html.Node{Type: html.ElementNode, Data: tt.tag}
		got := headingLevel(node)
		if got != tt.want {
			t.Errorf("headingLevel(%q) = %d, want %d", tt.tag, got, tt.want)
		}
	}
}

func TestCheckHeadingHierarchy_MultipleSkips(t *testing.T) {
	body := htmlPage(`<h1>Title</h1><h3>Skip 1</h3><h6>Skip 2</h6>`)
	issues := checkHeadingHierarchy(body, "https://example.com/test")

	if len(issues) != 2 {
		t.Fatalf("expected 2 heading skip issues, got %d: %+v", len(issues), issues)
	}
}

func TestCountElements(t *testing.T) {
	body := []byte(`<html><body><p>one</p><p>two</p><div><p>three</p></div></body></html>`)
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	count := countElements(doc, "p")
	if count != 3 {
		t.Errorf("expected 3 <p> elements, got %d", count)
	}
}
