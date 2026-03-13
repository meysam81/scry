package audit

import (
	"context"
	"strings"
	"testing"

	"github.com/meysam81/scry/internal/model"
)

func TestAccessibilityChecker_Name(t *testing.T) {
	c := NewAccessibilityChecker()
	if c.Name() != "accessibility" {
		t.Fatalf("expected name %q, got %q", "accessibility", c.Name())
	}
}

func TestAccessibilityChecker_NonHTML(t *testing.T) {
	c := NewAccessibilityChecker()
	page := &model.Page{
		URL:         "https://example.com/api",
		StatusCode:  200,
		ContentType: "application/json",
		Body:        []byte(`{"ok":true}`),
	}
	issues := c.Check(context.Background(), page)
	if len(issues) > 0 {
		t.Fatalf("expected no issues for non-HTML page, got %d", len(issues))
	}
}

func TestAccessibilityChecker_MissingFormLabel(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantIssue  bool
		wantSubstr string
	}{
		{
			name:      "input with no label",
			html:      `<html><body><input type="text"></body></html>`,
			wantIssue: true,
		},
		{
			name:      "input with label for",
			html:      `<html><body><label for="name">Name</label><input type="text" id="name"></body></html>`,
			wantIssue: false,
		},
		{
			name:      "input wrapped in label",
			html:      `<html><body><label>Name <input type="text"></label></body></html>`,
			wantIssue: false,
		},
		{
			name:      "input with aria-label",
			html:      `<html><body><input type="text" aria-label="Name"></body></html>`,
			wantIssue: false,
		},
		{
			name:      "input with aria-labelledby",
			html:      `<html><body><span id="lbl">Name</span><input type="text" aria-labelledby="lbl"></body></html>`,
			wantIssue: false,
		},
		{
			name:      "hidden input skipped",
			html:      `<html><body><input type="hidden" name="csrf"></body></html>`,
			wantIssue: false,
		},
		{
			name:      "submit input skipped",
			html:      `<html><body><input type="submit" value="Go"></body></html>`,
			wantIssue: false,
		},
		{
			name:      "button input skipped",
			html:      `<html><body><input type="button" value="Click"></body></html>`,
			wantIssue: false,
		},
		{
			name:      "reset input skipped",
			html:      `<html><body><input type="reset" value="Reset"></body></html>`,
			wantIssue: false,
		},
		{
			name:      "image input skipped",
			html:      `<html><body><input type="image" src="btn.png" alt="Go"></body></html>`,
			wantIssue: false,
		},
		{
			name:      "empty aria-label still flagged",
			html:      `<html><body><input type="text" aria-label="  "></body></html>`,
			wantIssue: true,
		},
		{
			name:      "input with no type defaults to text and needs label",
			html:      `<html><body><input></body></html>`,
			wantIssue: true,
		},
		{
			name:      "multiple unlabelled inputs",
			html:      `<html><body><input type="text"><input type="email"></body></html>`,
			wantIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/missing-form-label")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/missing-form-label issue, got none in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/missing-form-label issue, got %+v", issues)
			}

			// When expecting multiple issues, verify count.
			if tt.name == "multiple unlabelled inputs" {
				count := countCheck(issues, "accessibility/missing-form-label")
				if count != 2 {
					t.Errorf("expected 2 missing-form-label issues, got %d", count)
				}
			}
		})
	}
}

func TestAccessibilityChecker_EmptyLink(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name      string
		html      string
		wantIssue bool
	}{
		{
			name:      "link with no text",
			html:      `<html><body><a href="/"></a></body></html>`,
			wantIssue: true,
		},
		{
			name:      "link with text",
			html:      `<html><body><a href="/">Home</a></body></html>`,
			wantIssue: false,
		},
		{
			name:      "link with aria-label",
			html:      `<html><body><a href="/" aria-label="Home"></a></body></html>`,
			wantIssue: false,
		},
		{
			name:      "link with img alt",
			html:      `<html><body><a href="/"><img src="logo.png" alt="Home"></a></body></html>`,
			wantIssue: false,
		},
		{
			name:      "link with img no alt",
			html:      `<html><body><a href="/"><img src="logo.png"></a></body></html>`,
			wantIssue: true,
		},
		{
			name:      "link with whitespace only",
			html:      `<html><body><a href="/">   </a></body></html>`,
			wantIssue: true,
		},
		{
			name:      "empty aria-label still flagged",
			html:      `<html><body><a href="/" aria-label="  "></a></body></html>`,
			wantIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/empty-link")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/empty-link issue, got none in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/empty-link issue, got %+v", issues)
			}
		})
	}
}

func TestAccessibilityChecker_MissingSkipNav(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name      string
		html      string
		wantIssue bool
	}{
		{
			name:      "no skip nav link",
			html:      `<html><body><a href="/about">About</a></body></html>`,
			wantIssue: true,
		},
		{
			name:      "skip nav as first link",
			html:      `<html><body><a href="#main">Skip to content</a><nav><a href="/">Home</a></nav></body></html>`,
			wantIssue: false,
		},
		{
			name:      "skip nav as second link",
			html:      `<html><body><a href="/">Home</a><a href="#content">Skip</a></body></html>`,
			wantIssue: false,
		},
		{
			name:      "skip nav as third link",
			html:      `<html><body><a href="/">1</a><a href="/2">2</a><a href="#main">Skip</a></body></html>`,
			wantIssue: false,
		},
		{
			name:      "skip nav beyond third link not found",
			html:      `<html><body><a href="/">1</a><a href="/2">2</a><a href="/3">3</a><a href="#main">Skip</a></body></html>`,
			wantIssue: true,
		},
		{
			name:      "bare hash not counted as skip nav",
			html:      `<html><body><a href="#">Top</a></body></html>`,
			wantIssue: true,
		},
		{
			name:      "no links at all",
			html:      `<html><body><p>No links here</p></body></html>`,
			wantIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/missing-skip-nav")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/missing-skip-nav issue, got none")
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/missing-skip-nav issue")
			}
		})
	}
}

func TestAccessibilityChecker_HeadingHierarchy(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantIssue  bool
		wantSubstr string
		wantCount  int
	}{
		{
			name:      "good hierarchy h1 h2 h3",
			html:      `<html><body><h1>Title</h1><h2>Sub</h2><h3>Detail</h3></body></html>`,
			wantIssue: false,
		},
		{
			name:       "h1 to h3 skip",
			html:       `<html><body><h1>Title</h1><h3>Skipped</h3></body></html>`,
			wantIssue:  true,
			wantSubstr: "h1 to h3",
			wantCount:  1,
		},
		{
			name:       "h2 to h4 skip",
			html:       `<html><body><h1>Title</h1><h2>Sub</h2><h4>Oops</h4></body></html>`,
			wantIssue:  true,
			wantSubstr: "h2 to h4",
			wantCount:  1,
		},
		{
			name:      "h2 to h1 going down is OK",
			html:      `<html><body><h1>Title</h1><h2>Sub</h2><h1>Another section</h1></body></html>`,
			wantIssue: false,
		},
		{
			name:      "no headings at all",
			html:      `<html><body><p>No headings</p></body></html>`,
			wantIssue: false,
		},
		{
			name:      "single heading",
			html:      `<html><body><h1>Only one</h1></body></html>`,
			wantIssue: false,
		},
		{
			name:       "multiple skips",
			html:       `<html><body><h1>Title</h1><h3>Skip1</h3><h6>Skip2</h6></body></html>`,
			wantIssue:  true,
			wantSubstr: "h1 to h3",
			wantCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/heading-hierarchy")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/heading-hierarchy issue, got none")
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/heading-hierarchy issue, got %+v", filterCheck(issues, "accessibility/heading-hierarchy"))
			}

			if tt.wantSubstr != "" {
				assertSubstr(t, issues, "accessibility/heading-hierarchy", tt.wantSubstr)
			}
			if tt.wantCount > 0 {
				count := countCheck(issues, "accessibility/heading-hierarchy")
				if count != tt.wantCount {
					t.Errorf("expected %d heading-hierarchy issues, got %d", tt.wantCount, count)
				}
			}
		})
	}
}

func TestAccessibilityChecker_MissingButtonText(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name      string
		html      string
		wantIssue bool
	}{
		{
			name:      "button with no text",
			html:      `<html><body><button></button></body></html>`,
			wantIssue: true,
		},
		{
			name:      "button with text",
			html:      `<html><body><button>Submit</button></body></html>`,
			wantIssue: false,
		},
		{
			name:      "button with aria-label",
			html:      `<html><body><button aria-label="Close"></button></body></html>`,
			wantIssue: false,
		},
		{
			name:      "button with whitespace only",
			html:      `<html><body><button>   </button></body></html>`,
			wantIssue: true,
		},
		{
			name:      "button with empty aria-label",
			html:      `<html><body><button aria-label="  "></button></body></html>`,
			wantIssue: true,
		},
		{
			name:      "button with nested span text",
			html:      `<html><body><button><span>Click me</span></button></body></html>`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/missing-button-text")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/missing-button-text issue, got none")
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/missing-button-text issue")
			}
		})
	}
}

func TestAccessibilityChecker_MissingTableHeader(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name      string
		html      string
		wantIssue bool
	}{
		{
			name:      "table without th",
			html:      `<html><body><table><tr><td>Data</td></tr></table></body></html>`,
			wantIssue: true,
		},
		{
			name:      "table with th",
			html:      `<html><body><table><thead><tr><th>Header</th></tr></thead><tbody><tr><td>Data</td></tr></tbody></table></body></html>`,
			wantIssue: false,
		},
		{
			name:      "table with th in body row",
			html:      `<html><body><table><tr><th>Header</th><td>Data</td></tr></table></body></html>`,
			wantIssue: false,
		},
		{
			name:      "multiple tables mixed",
			html:      `<html><body><table><tr><th>OK</th></tr></table><table><tr><td>Bad</td></tr></table></body></html>`,
			wantIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/missing-table-header")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/missing-table-header issue, got none")
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/missing-table-header issue")
			}
		})
	}
}

func TestAccessibilityChecker_MissingImgAltInFigure(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name      string
		html      string
		wantIssue bool
	}{
		{
			name:      "figure with img no alt no caption",
			html:      `<html><body><figure><img src="photo.jpg"></figure></body></html>`,
			wantIssue: true,
		},
		{
			name:      "figure with img and figcaption",
			html:      `<html><body><figure><img src="photo.jpg"><figcaption>A photo</figcaption></figure></body></html>`,
			wantIssue: false,
		},
		{
			name:      "figure with img with alt",
			html:      `<html><body><figure><img src="photo.jpg" alt="A nice photo"></figure></body></html>`,
			wantIssue: false,
		},
		{
			name:      "figure with img empty alt no caption",
			html:      `<html><body><figure><img src="photo.jpg" alt=""></figure></body></html>`,
			wantIssue: true,
		},
		{
			name:      "figure without img",
			html:      `<html><body><figure><video src="clip.mp4"></video></figure></body></html>`,
			wantIssue: false,
		},
		{
			name:      "figure with img whitespace alt no caption",
			html:      `<html><body><figure><img src="photo.jpg" alt="   "></figure></body></html>`,
			wantIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/missing-img-alt-in-figure")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/missing-img-alt-in-figure issue, got none")
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/missing-img-alt-in-figure issue")
			}
		})
	}
}

func TestAccessibilityChecker_PositiveTabindex(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantIssue  bool
		wantSubstr string
	}{
		{
			name:       "positive tabindex",
			html:       `<html><body><div tabindex="5">Focus me</div></body></html>`,
			wantIssue:  true,
			wantSubstr: "tabindex=5",
		},
		{
			name:      "tabindex zero is fine",
			html:      `<html><body><div tabindex="0">OK</div></body></html>`,
			wantIssue: false,
		},
		{
			name:      "negative tabindex is fine",
			html:      `<html><body><div tabindex="-1">OK</div></body></html>`,
			wantIssue: false,
		},
		{
			name:      "no tabindex",
			html:      `<html><body><div>No tabindex</div></body></html>`,
			wantIssue: false,
		},
		{
			name:       "multiple positive tabindexes",
			html:       `<html><body><input tabindex="1"><input tabindex="2"></body></html>`,
			wantIssue:  true,
			wantSubstr: "tabindex=",
		},
		{
			name:      "non-numeric tabindex ignored",
			html:      `<html><body><div tabindex="abc">Huh</div></body></html>`,
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/positive-tabindex")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/positive-tabindex issue, got none")
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/positive-tabindex issue")
			}

			if tt.wantSubstr != "" {
				assertSubstr(t, issues, "accessibility/positive-tabindex", tt.wantSubstr)
			}

			// Verify count for multiple.
			if tt.name == "multiple positive tabindexes" {
				count := countCheck(issues, "accessibility/positive-tabindex")
				if count != 2 {
					t.Errorf("expected 2 positive-tabindex issues, got %d", count)
				}
			}
		})
	}
}

func TestAccessibilityChecker_MissingLandmarks(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name      string
		html      string
		wantIssue bool
	}{
		{
			name:      "no landmarks at all",
			html:      `<html><body><div>No landmarks</div></body></html>`,
			wantIssue: true,
		},
		{
			name:      "has main element",
			html:      `<html><body><main>Content</main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "has nav element",
			html:      `<html><body><nav><a href="/">Home</a></nav></body></html>`,
			wantIssue: false,
		},
		{
			name:      "has top-level header",
			html:      `<html><body><header>Site header</header></body></html>`,
			wantIssue: false,
		},
		{
			name:      "has top-level footer",
			html:      `<html><body><footer>Site footer</footer></body></html>`,
			wantIssue: false,
		},
		{
			name:      "header inside article does not count",
			html:      `<html><body><article><header>Article header</header></article></body></html>`,
			wantIssue: true,
		},
		{
			name:      "footer inside article does not count",
			html:      `<html><body><article><footer>Article footer</footer></article></body></html>`,
			wantIssue: true,
		},
		{
			name:      "role=main",
			html:      `<html><body><div role="main">Content</div></body></html>`,
			wantIssue: false,
		},
		{
			name:      "role=navigation",
			html:      `<html><body><div role="navigation">Nav</div></body></html>`,
			wantIssue: false,
		},
		{
			name:      "role=banner",
			html:      `<html><body><div role="banner">Banner</div></body></html>`,
			wantIssue: false,
		},
		{
			name:      "role=contentinfo",
			html:      `<html><body><div role="contentinfo">Info</div></body></html>`,
			wantIssue: false,
		},
		{
			name:      "role with different case",
			html:      `<html><body><div role="Main">Content</div></body></html>`,
			wantIssue: false,
		},
		{
			name:      "unrelated role does not count",
			html:      `<html><body><div role="dialog">Dialog</div></body></html>`,
			wantIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/missing-landmarks")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/missing-landmarks issue, got none in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/missing-landmarks issue, got %+v", filterCheck(issues, "accessibility/missing-landmarks"))
			}
		})
	}
}

func TestAccessibilityChecker_InvalidAria(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantIssue  bool
		wantSubstr string
		wantCount  int
	}{
		{
			name:       "aria-hidden with tabindex 0",
			html:       `<html><body><main><div aria-hidden="true" tabindex="0">Hidden</div></main></body></html>`,
			wantIssue:  true,
			wantSubstr: "aria-hidden=true with positive tabindex",
		},
		{
			name:       "aria-hidden with positive tabindex",
			html:       `<html><body><main><span aria-hidden="true" tabindex="3">Hidden</span></main></body></html>`,
			wantIssue:  true,
			wantSubstr: "<span>",
		},
		{
			name:      "aria-hidden with negative tabindex is fine",
			html:      `<html><body><main><div aria-hidden="true" tabindex="-1">Hidden</div></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "aria-hidden without tabindex is fine",
			html:      `<html><body><main><div aria-hidden="true">Hidden</div></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "no aria-hidden is fine",
			html:      `<html><body><main><div tabindex="0">Visible</div></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "aria-hidden=false with tabindex is fine",
			html:      `<html><body><main><div aria-hidden="false" tabindex="0">Visible</div></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "multiple violations",
			html:      `<html><body><main><div aria-hidden="true" tabindex="0">A</div><span aria-hidden="true" tabindex="1">B</span></main></body></html>`,
			wantIssue: true,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/invalid-aria")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/invalid-aria issue, got none in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/invalid-aria issue, got %+v", filterCheck(issues, "accessibility/invalid-aria"))
			}
			if tt.wantSubstr != "" {
				assertSubstr(t, issues, "accessibility/invalid-aria", tt.wantSubstr)
			}
			if tt.wantCount > 0 {
				count := countCheck(issues, "accessibility/invalid-aria")
				if count != tt.wantCount {
					t.Errorf("expected %d invalid-aria issues, got %d", tt.wantCount, count)
				}
			}
		})
	}
}

func TestAccessibilityChecker_OnclickWithoutKeyboard(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantIssue  bool
		wantSubstr string
		wantCount  int
	}{
		{
			name:       "div with onclick no tabindex no role",
			html:       `<html><body><main><div onclick="doStuff()">Click</div></main></body></html>`,
			wantIssue:  true,
			wantSubstr: "<div>",
		},
		{
			name:       "span with onclick no tabindex no role",
			html:       `<html><body><main><span onclick="doStuff()">Click</span></main></body></html>`,
			wantIssue:  true,
			wantSubstr: "<span>",
		},
		{
			name:      "div with onclick and tabindex",
			html:      `<html><body><main><div onclick="doStuff()" tabindex="0">Click</div></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "div with onclick and role",
			html:      `<html><body><main><div onclick="doStuff()" role="button">Click</div></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "div with onclick and both tabindex and role",
			html:      `<html><body><main><div onclick="doStuff()" tabindex="0" role="button">Click</div></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "button with onclick is fine",
			html:      `<html><body><main><button onclick="doStuff()">Click</button></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "a with onclick is fine",
			html:      `<html><body><main><a href="#" onclick="doStuff()">Click</a></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "input with onclick is fine",
			html:      `<html><body><main><input type="button" onclick="doStuff()"></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "select with onclick is fine",
			html:      `<html><body><main><select onclick="doStuff()"><option>A</option></select></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "textarea with onclick is fine",
			html:      `<html><body><main><textarea onclick="doStuff()"></textarea></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "div without onclick is fine",
			html:      `<html><body><main><div>No onclick</div></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "multiple violations",
			html:      `<html><body><main><div onclick="a()">A</div><span onclick="b()">B</span></main></body></html>`,
			wantIssue: true,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/onclick-without-keyboard")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/onclick-without-keyboard issue, got none in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/onclick-without-keyboard issue, got %+v", filterCheck(issues, "accessibility/onclick-without-keyboard"))
			}
			if tt.wantSubstr != "" {
				assertSubstr(t, issues, "accessibility/onclick-without-keyboard", tt.wantSubstr)
			}
			if tt.wantCount > 0 {
				count := countCheck(issues, "accessibility/onclick-without-keyboard")
				if count != tt.wantCount {
					t.Errorf("expected %d onclick-without-keyboard issues, got %d", tt.wantCount, count)
				}
			}
		})
	}
}

func TestAccessibilityChecker_MissingVideoCaptions(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name      string
		html      string
		wantIssue bool
		wantCount int
	}{
		{
			name:      "video without track",
			html:      `<html><body><main><video src="clip.mp4"></video></main></body></html>`,
			wantIssue: true,
		},
		{
			name:      "video with captions track",
			html:      `<html><body><main><video src="clip.mp4"><track kind="captions" src="en.vtt"></video></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "video with subtitles track",
			html:      `<html><body><main><video src="clip.mp4"><track kind="subtitles" src="en.vtt"></video></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "video with description track only",
			html:      `<html><body><main><video src="clip.mp4"><track kind="descriptions" src="desc.vtt"></video></main></body></html>`,
			wantIssue: true,
		},
		{
			name:      "video with chapters track only",
			html:      `<html><body><main><video src="clip.mp4"><track kind="chapters" src="ch.vtt"></video></main></body></html>`,
			wantIssue: true,
		},
		{
			name:      "multiple videos mixed",
			html:      `<html><body><main><video src="a.mp4"><track kind="captions" src="a.vtt"></video><video src="b.mp4"></video></main></body></html>`,
			wantIssue: true,
			wantCount: 1,
		},
		{
			name:      "no video elements",
			html:      `<html><body><main><p>No video here</p></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "video with Captions track case insensitive",
			html:      `<html><body><main><video src="clip.mp4"><track kind="Captions" src="en.vtt"></video></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "multiple videos all missing",
			html:      `<html><body><main><video src="a.mp4"></video><video src="b.mp4"></video></main></body></html>`,
			wantIssue: true,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/missing-video-captions")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/missing-video-captions issue, got none in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/missing-video-captions issue, got %+v", filterCheck(issues, "accessibility/missing-video-captions"))
			}
			if tt.wantCount > 0 {
				count := countCheck(issues, "accessibility/missing-video-captions")
				if count != tt.wantCount {
					t.Errorf("expected %d missing-video-captions issues, got %d", tt.wantCount, count)
				}
			}
		})
	}
}

func TestAccessibilityChecker_MissingAutocomplete(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	tests := []struct {
		name       string
		html       string
		wantIssue  bool
		wantSubstr string
		wantCount  int
	}{
		{
			name:       "email input without autocomplete",
			html:       `<html><body><main><label>Email<input type="email"></label></main></body></html>`,
			wantIssue:  true,
			wantSubstr: `type="email"`,
		},
		{
			name:      "email input with autocomplete",
			html:      `<html><body><main><label>Email<input type="email" autocomplete="email"></label></main></body></html>`,
			wantIssue: false,
		},
		{
			name:       "password input without autocomplete",
			html:       `<html><body><main><label>Password<input type="password"></label></main></body></html>`,
			wantIssue:  true,
			wantSubstr: `type="password"`,
		},
		{
			name:       "tel input without autocomplete",
			html:       `<html><body><main><label>Phone<input type="tel"></label></main></body></html>`,
			wantIssue:  true,
			wantSubstr: `type="tel"`,
		},
		{
			name:      "text input without autocomplete is fine",
			html:      `<html><body><main><label>Search<input type="text"></label></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "input without type is fine",
			html:      `<html><body><main><label>Name<input></label></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "checkbox input without autocomplete is fine",
			html:      `<html><body><main><label>Agree<input type="checkbox"></label></main></body></html>`,
			wantIssue: false,
		},
		{
			name:      "multiple inputs mixed",
			html:      `<html><body><main><label>E<input type="email"></label><label>P<input type="password"></label><label>S<input type="text"></label></main></body></html>`,
			wantIssue: true,
			wantCount: 2,
		},
		{
			name:      "name input without autocomplete",
			html:      `<html><body><main><label>Name<input type="name"></label></main></body></html>`,
			wantIssue: true,
		},
		{
			name:      "username input without autocomplete",
			html:      `<html><body><main><label>User<input type="username"></label></main></body></html>`,
			wantIssue: true,
		},
		{
			name:      "address input without autocomplete",
			html:      `<html><body><main><label>Addr<input type="address"></label></main></body></html>`,
			wantIssue: true,
		},
		{
			name:      "postal-code input without autocomplete",
			html:      `<html><body><main><label>Zip<input type="postal-code"></label></main></body></html>`,
			wantIssue: true,
		},
		{
			name:      "cc-number input without autocomplete",
			html:      `<html><body><main><label>Card<input type="cc-number"></label></main></body></html>`,
			wantIssue: true,
		},
		{
			name:      "Email type case insensitive",
			html:      `<html><body><main><label>Email<input type="Email"></label></main></body></html>`,
			wantIssue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := htmlPage(tt.html)
			issues := checker.Check(ctx, page)
			found := hasCheck(issues, "accessibility/missing-autocomplete")

			if tt.wantIssue && !found {
				t.Errorf("expected accessibility/missing-autocomplete issue, got none in %+v", issues)
			}
			if !tt.wantIssue && found {
				t.Errorf("did not expect accessibility/missing-autocomplete issue, got %+v", filterCheck(issues, "accessibility/missing-autocomplete"))
			}
			if tt.wantSubstr != "" {
				assertSubstr(t, issues, "accessibility/missing-autocomplete", tt.wantSubstr)
			}
			if tt.wantCount > 0 {
				count := countCheck(issues, "accessibility/missing-autocomplete")
				if count != tt.wantCount {
					t.Errorf("expected %d missing-autocomplete issues, got %d", tt.wantCount, count)
				}
			}
		})
	}
}

func TestAccessibilityChecker_FullPage_NoIssues(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	// A well-structured accessible page should produce no issues.
	page := htmlPage(`<!DOCTYPE html>
<html lang="en">
<head><title>Accessible Page</title></head>
<body>
  <a href="#main">Skip to main content</a>
  <nav><a href="/">Home</a><a href="/about">About</a></nav>
  <main id="main">
    <h1>Welcome</h1>
    <h2>Section</h2>
    <p>Content</p>
    <form>
      <label for="email">Email</label>
      <input id="email" type="email" autocomplete="email">
      <button type="submit">Submit</button>
    </form>
    <table>
      <thead><tr><th>Name</th><th>Value</th></tr></thead>
      <tbody><tr><td>A</td><td>1</td></tr></tbody>
    </table>
    <figure>
      <img src="photo.jpg" alt="A scenic view">
    </figure>
  </main>
</body>
</html>`)

	issues := checker.Check(ctx, page)
	if len(issues) > 0 {
		t.Errorf("expected no issues for accessible page, got %d: %+v", len(issues), issues)
	}
}

func TestAccessibilityChecker_FullPage_AllIssues(t *testing.T) {
	checker := NewAccessibilityChecker()
	ctx := context.Background()

	// A page with every accessibility problem at once.
	page := htmlPage(`<!DOCTYPE html>
<html>
<body>
  <a href="/about">About</a>
  <a href="/contact">Contact</a>
  <a href="/help">Help</a>
  <a href=""></a>
  <h1>Title</h1>
  <h3>Skipped h2</h3>
  <form>
    <input type="text">
    <input type="email">
  </form>
  <button></button>
  <table><tr><td>No header</td></tr></table>
  <figure><img src="photo.jpg"></figure>
  <div tabindex="3">Bad order</div>
  <div aria-hidden="true" tabindex="0">Hidden but focusable</div>
  <span onclick="doStuff()">Click me</span>
  <video src="clip.mp4"></video>
</body>
</html>`)

	issues := checker.Check(ctx, page)

	expectedChecks := []string{
		"accessibility/missing-skip-nav",
		"accessibility/empty-link",
		"accessibility/heading-hierarchy",
		"accessibility/missing-form-label",
		"accessibility/missing-button-text",
		"accessibility/missing-table-header",
		"accessibility/missing-img-alt-in-figure",
		"accessibility/positive-tabindex",
		"accessibility/missing-landmarks",
		"accessibility/invalid-aria",
		"accessibility/onclick-without-keyboard",
		"accessibility/missing-video-captions",
		"accessibility/missing-autocomplete",
	}

	for _, check := range expectedChecks {
		if !hasCheck(issues, check) {
			t.Errorf("expected %s issue in full-problem page, not found", check)
		}
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func hasCheck(issues []model.Issue, checkName string) bool {
	for _, iss := range issues {
		if iss.CheckName == checkName {
			return true
		}
	}
	return false
}

func countCheck(issues []model.Issue, checkName string) int {
	count := 0
	for _, iss := range issues {
		if iss.CheckName == checkName {
			count++
		}
	}
	return count
}

func filterCheck(issues []model.Issue, checkName string) []model.Issue {
	var result []model.Issue
	for _, iss := range issues {
		if iss.CheckName == checkName {
			result = append(result, iss)
		}
	}
	return result
}

func assertSubstr(t *testing.T, issues []model.Issue, checkName, substr string) {
	t.Helper()
	for _, iss := range issues {
		if iss.CheckName == checkName {
			if strings.Contains(iss.Message, substr) {
				return
			}
		}
	}
	t.Errorf("no %s issue message contains %q", checkName, substr)
}
