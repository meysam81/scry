package audit

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/meysam81/scry/internal/model"
	"golang.org/x/net/html"
)

const maxImageSize = 500 * 1024

// ImageChecker analyses pages for image-related issues.
type ImageChecker struct {
	client *http.Client
}

// NewImageChecker returns a new ImageChecker with a timeout-limited HTTP client.
func NewImageChecker() *ImageChecker {
	return &ImageChecker{client: &http.Client{Timeout: 10 * time.Second}}
}

// SetHTTPClient sets the HTTP client used for HEAD requests.
func (c *ImageChecker) SetHTTPClient(client *http.Client) {
	c.client = client
}

// Name returns the checker name.
func (c *ImageChecker) Name() string { return "images" }

// Check runs per-page image checks.
func (c *ImageChecker) Check(ctx context.Context, page *model.Page) []model.Issue {
	if !isHTMLContent(page) {
		return nil
	}
	doc, err := parseHTMLDoc(page.Body)
	if err != nil {
		return nil
	}

	var issues []model.Issue
	imgs := findNodes(doc, "img")

	for _, img := range imgs {
		issues = append(issues, c.checkAlt(img, page.URL)...)
		issues = append(issues, c.checkRemoteImage(ctx, img, page.URL)...)
	}

	return issues
}

func (c *ImageChecker) checkAlt(img *html.Node, pageURL string) []model.Issue {
	src, _ := getAttr(img, "src")
	alt, hasAlt := getAttr(img, "alt")

	if !hasAlt {
		return []model.Issue{{
			CheckName: "images/missing-alt",
			Severity:  model.SeverityWarning,
			URL:       pageURL,
			Message:   fmt.Sprintf("image is missing alt attribute: %s", src),
		}}
	}

	if alt == "" && hasAnchorAncestor(img) {
		return []model.Issue{{
			CheckName: "images/empty-alt-in-link",
			Severity:  model.SeverityWarning,
			URL:       pageURL,
			Message:   fmt.Sprintf("image inside link has empty alt attribute: %s", src),
		}}
	}

	return nil
}

func (c *ImageChecker) checkRemoteImage(ctx context.Context, img *html.Node, pageURL string) []model.Issue {
	src, _ := getAttr(img, "src")
	if !strings.HasPrefix(src, "http://") && !strings.HasPrefix(src, "https://") {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, src, nil)
	if err != nil {
		return nil
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

	var issues []model.Issue

	if resp.StatusCode >= 400 {
		issues = append(issues, model.Issue{
			CheckName: "images/broken-src",
			Severity:  model.SeverityCritical,
			URL:       pageURL,
			Message:   fmt.Sprintf("image returned HTTP %d: %s", resp.StatusCode, src),
		})
	}

	if resp.ContentLength > int64(maxImageSize) {
		issues = append(issues, model.Issue{
			CheckName: "images/large-image",
			Severity:  model.SeverityWarning,
			URL:       pageURL,
			Message:   fmt.Sprintf("image is %d bytes (%.0f KB): %s", resp.ContentLength, float64(resp.ContentLength)/1024, src),
		})
	}

	return issues
}

// hasAnchorAncestor walks up the node tree to check if any ancestor is an <a> element.
func hasAnchorAncestor(n *html.Node) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "a" {
			return true
		}
	}
	return false
}
