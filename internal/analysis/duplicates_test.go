package analysis

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func wrapHTML(body string) []byte {
	return []byte(fmt.Sprintf(`<!DOCTYPE html><html><head><title>Test</title></head><body>%s</body></html>`, body))
}

func TestSimHash_IdenticalTexts(t *testing.T) {
	text := "the quick brown fox jumps over the lazy dog"
	h1 := simHash(text)
	h2 := simHash(text)
	if h1 != h2 {
		t.Fatalf("identical texts produced different simhashes: %d vs %d", h1, h2)
	}
}

func TestSimHash_SimilarTexts(t *testing.T) {
	// Use long, mostly identical texts — only one word differs out of many.
	base := strings.Repeat("the quick brown fox jumps over the lazy dog ", 50)
	t1 := base + "alpha"
	t2 := base + "beta"
	h1 := simHash(t1)
	h2 := simHash(t2)
	dist := hammingDistance(h1, h2)
	if dist > simHashDistanceThreshold {
		t.Errorf("similar texts have hamming distance %d, expected <= %d", dist, simHashDistanceThreshold)
	}
}

func TestSimHash_DifferentTexts(t *testing.T) {
	t1 := "the quick brown fox jumps over the lazy dog"
	t2 := "an entirely different set of words about programming and computers in the modern world"
	h1 := simHash(t1)
	h2 := simHash(t2)
	dist := hammingDistance(h1, h2)
	if dist <= simHashDistanceThreshold {
		t.Errorf("different texts have hamming distance %d, expected > %d", dist, simHashDistanceThreshold)
	}
}

func TestSimHash_EmptyText(t *testing.T) {
	h := simHash("")
	if h != 0 {
		t.Errorf("expected 0 for empty text, got %d", h)
	}
}

func TestHammingDistance(t *testing.T) {
	tests := []struct {
		a, b uint64
		want int
	}{
		{0, 0, 0},
		{0xFF, 0xFF, 0},
		{0xFF, 0x00, 8},
		{0b1010, 0b0101, 4},
	}
	for _, tt := range tests {
		got := hammingDistance(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("hammingDistance(%#x, %#x) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestExtractText(t *testing.T) {
	body := []byte(`<html><body>
		<h1>Hello World</h1>
		<script>var x = 1;</script>
		<style>.foo { color: red; }</style>
		<p>This is a paragraph.</p>
	</body></html>`)
	text := extractText(body)
	if !strings.Contains(text, "Hello World") {
		t.Errorf("expected 'Hello World' in text, got %q", text)
	}
	if !strings.Contains(text, "This is a paragraph.") {
		t.Errorf("expected paragraph text, got %q", text)
	}
	if strings.Contains(text, "var x") {
		t.Errorf("script content should be stripped, got %q", text)
	}
	if strings.Contains(text, "color") {
		t.Errorf("style content should be stripped, got %q", text)
	}
}

func TestWordCount(t *testing.T) {
	tests := []struct {
		text string
		want int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  hello   world  ", 2},
		{"one two three four five", 5},
	}
	for _, tt := range tests {
		got := wordCount(tt.text)
		if got != tt.want {
			t.Errorf("wordCount(%q) = %d, want %d", tt.text, got, tt.want)
		}
	}
}

func TestDuplicateDetector_ExactDuplicates(t *testing.T) {
	body := wrapHTML("<p>This is the exact same content on both pages.</p>")
	pages := []*model.Page{
		{URL: "https://example.com/a", ContentType: "text/html", Body: body},
		{URL: "https://example.com/b", ContentType: "text/html", Body: body},
	}

	dd := NewDuplicateDetector()
	issues := dd.Analyze(pages)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "content/exact-duplicate" {
			found = true
			if iss.Severity != model.SeverityWarning {
				t.Errorf("expected severity warning, got %s", iss.Severity)
			}
			if !strings.Contains(iss.Message, "exact duplicate") {
				t.Errorf("expected 'exact duplicate' in message, got %q", iss.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected content/exact-duplicate issue, got %+v", issues)
	}
}

func TestDuplicateDetector_NearDuplicates(t *testing.T) {
	// Generate two pages with nearly identical content — only one word differs.
	longText := strings.Repeat("alpha beta gamma delta epsilon zeta eta theta iota kappa ", 20)
	body1 := wrapHTML(fmt.Sprintf("<p>%s first unique word</p>", longText))
	body2 := wrapHTML(fmt.Sprintf("<p>%s second unique word</p>", longText))

	pages := []*model.Page{
		{URL: "https://example.com/a", ContentType: "text/html", Body: body1},
		{URL: "https://example.com/b", ContentType: "text/html", Body: body2},
	}

	dd := NewDuplicateDetector()
	issues := dd.Analyze(pages)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "content/near-duplicate" {
			found = true
			if iss.Severity != model.SeverityInfo {
				t.Errorf("expected severity info, got %s", iss.Severity)
			}
			if !strings.Contains(iss.Message, "near-duplicate") {
				t.Errorf("expected 'near-duplicate' in message, got %q", iss.Message)
			}
			if !strings.Contains(iss.Message, "similarity:") {
				t.Errorf("expected 'similarity:' in message, got %q", iss.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected content/near-duplicate issue, got %+v", issues)
	}
}

func TestDuplicateDetector_NoNearDuplicateForExactDuplicates(t *testing.T) {
	body := wrapHTML("<p>Exact same content on both pages for testing dedup.</p>")
	pages := []*model.Page{
		{URL: "https://example.com/a", ContentType: "text/html", Body: body},
		{URL: "https://example.com/b", ContentType: "text/html", Body: body},
	}

	dd := NewDuplicateDetector()
	issues := dd.Analyze(pages)

	for _, iss := range issues {
		if iss.CheckName == "content/near-duplicate" {
			t.Fatalf("should not report near-duplicate for exact duplicates, got %+v", iss)
		}
	}
}

func TestDuplicateDetector_ThinContent(t *testing.T) {
	body := wrapHTML("<p>Short page.</p>")
	pages := []*model.Page{
		{URL: "https://example.com/thin", ContentType: "text/html", Body: body},
	}

	dd := NewDuplicateDetector()
	issues := dd.Analyze(pages)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "content/thin-content" {
			found = true
			if iss.Severity != model.SeverityInfo {
				t.Errorf("expected severity info, got %s", iss.Severity)
			}
			if !strings.Contains(iss.Message, "words") {
				t.Errorf("expected word count in message, got %q", iss.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected content/thin-content issue, got %+v", issues)
	}
}

func TestDuplicateDetector_NoThinContent(t *testing.T) {
	words := strings.Repeat("word ", 150)
	body := wrapHTML(fmt.Sprintf("<p>%s</p>", words))
	pages := []*model.Page{
		{URL: "https://example.com/long", ContentType: "text/html", Body: body},
	}

	dd := NewDuplicateDetector()
	issues := dd.Analyze(pages)

	for _, iss := range issues {
		if iss.CheckName == "content/thin-content" {
			t.Fatalf("did not expect thin content issue, got %+v", iss)
		}
	}
}

func TestDuplicateDetector_SkipsNonHTML(t *testing.T) {
	pages := []*model.Page{
		{URL: "https://example.com/image.png", ContentType: "image/png", Body: []byte("binary data")},
	}

	dd := NewDuplicateDetector()
	issues := dd.Analyze(pages)

	if len(issues) != 0 {
		t.Fatalf("expected no issues for non-HTML content, got %+v", issues)
	}
}

func TestDuplicateDetector_CustomThreshold(t *testing.T) {
	words := strings.Repeat("word ", 60)
	body := wrapHTML(fmt.Sprintf("<p>%s</p>", words))
	pages := []*model.Page{
		{URL: "https://example.com/page", ContentType: "text/html", Body: body},
	}

	dd := &DuplicateDetector{ThinContentThreshold: 50}
	issues := dd.Analyze(pages)

	for _, iss := range issues {
		if iss.CheckName == "content/thin-content" {
			t.Fatalf("60 words should not be thin at threshold 50, got %+v", iss)
		}
	}

	dd2 := &DuplicateDetector{ThinContentThreshold: 80}
	issues2 := dd2.Analyze(pages)

	found := false
	for _, iss := range issues2 {
		if iss.CheckName == "content/thin-content" {
			found = true
		}
	}
	if !found {
		t.Errorf("60 words should be thin at threshold 80")
	}
}

func TestDuplicateDetector_SiteCheckerInterface(t *testing.T) {
	dd := NewDuplicateDetector()

	// Verify Check returns nil.
	issues := dd.Check(context.Background(), &model.Page{})
	if issues != nil {
		t.Fatalf("Check should return nil, got %+v", issues)
	}

	// Verify Name.
	if dd.Name() != "content-duplicates" {
		t.Errorf("unexpected name: %s", dd.Name())
	}
}

func TestDuplicateDetector_MultipleExactDuplicateGroups(t *testing.T) {
	bodyA := wrapHTML("<p>Content group A repeated verbatim.</p>")
	bodyB := wrapHTML("<p>Content group B is entirely different material.</p>")

	pages := []*model.Page{
		{URL: "https://example.com/a1", ContentType: "text/html", Body: bodyA},
		{URL: "https://example.com/a2", ContentType: "text/html", Body: bodyA},
		{URL: "https://example.com/b1", ContentType: "text/html", Body: bodyB},
		{URL: "https://example.com/b2", ContentType: "text/html", Body: bodyB},
	}

	dd := NewDuplicateDetector()
	issues := dd.Analyze(pages)

	exactCount := 0
	for _, iss := range issues {
		if iss.CheckName == "content/exact-duplicate" {
			exactCount++
		}
	}
	// Each group of 2 produces 1 pair: a1-a2 and b1-b2 = 2 issues.
	if exactCount != 2 {
		t.Errorf("expected 2 exact-duplicate issues, got %d", exactCount)
	}
}

func TestDuplicateDetector_ThreeExactDuplicates(t *testing.T) {
	body := wrapHTML("<p>Identical content for three pages.</p>")
	pages := []*model.Page{
		{URL: "https://example.com/a", ContentType: "text/html", Body: body},
		{URL: "https://example.com/b", ContentType: "text/html", Body: body},
		{URL: "https://example.com/c", ContentType: "text/html", Body: body},
	}

	dd := NewDuplicateDetector()
	issues := dd.Analyze(pages)

	exactCount := 0
	for _, iss := range issues {
		if iss.CheckName == "content/exact-duplicate" {
			exactCount++
		}
	}
	// 3 pages = 3 pairs: a-b, a-c, b-c.
	if exactCount != 3 {
		t.Errorf("expected 3 exact-duplicate issues for 3 identical pages, got %d", exactCount)
	}
}
