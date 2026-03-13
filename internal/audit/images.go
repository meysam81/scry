package audit

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/meysam81/scry/internal/model"
	"github.com/meysam81/scry/internal/safenet"
	"golang.org/x/net/html"
)

// legacyImageExts lists file extensions considered legacy image formats.
var legacyImageExts = []string{".jpg", ".jpeg", ".gif", ".bmp", ".tiff"}

const maxImageSize = 500 * 1024

// ImageChecker analyses pages for image-related issues.
type ImageChecker struct {
	client       *http.Client
	allowPrivate bool // skip SSRF checks (for testing only)
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
	doc := parseHTMLDocLog(page.Body, page.URL)
	if doc == nil {
		return nil
	}

	var issues []model.Issue
	imgs := findNodes(doc, "img")

	for i, img := range imgs {
		issues = append(issues, c.checkAlt(img, page.URL)...)
		issues = append(issues, c.checkRemoteImage(ctx, img, page.URL)...)
		issues = append(issues, c.checkLegacyFormat(img, page.URL)...)
		issues = append(issues, c.checkMissingLazyLoading(img, page.URL, i)...)
		issues = append(issues, c.checkMissingDimensions(img, page.URL)...)
		issues = append(issues, c.checkMissingResponsive(img, page.URL)...)
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

	if !c.allowPrivate && !safenet.IsSafeURL(src) {
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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			getAuditLogger().Warn().Err(cerr).Str("url", src).Msg("resp body close failed")
		}
	}()

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

// hasAncestorTag walks up the node tree to check if any ancestor matches the given tag.
func hasAncestorTag(n *html.Node, tag string) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == tag {
			return true
		}
	}
	return false
}

func (c *ImageChecker) checkLegacyFormat(img *html.Node, pageURL string) []model.Issue {
	src, _ := getAttr(img, "src")
	if src == "" {
		return nil
	}
	lower := strings.ToLower(src)
	// Strip query string and fragment before checking extension.
	if idx := strings.IndexAny(lower, "?#"); idx != -1 {
		lower = lower[:idx]
	}
	for _, ext := range legacyImageExts {
		if strings.HasSuffix(lower, ext) {
			return []model.Issue{{
				CheckName: "images/legacy-format",
				Severity:  model.SeverityInfo,
				URL:       pageURL,
				Message:   fmt.Sprintf("image uses legacy format (%s), consider WebP or AVIF: %s", ext, src),
			}}
		}
	}
	return nil
}

func (c *ImageChecker) checkMissingLazyLoading(img *html.Node, pageURL string, index int) []model.Issue {
	// Skip above-the-fold images: first 3 images or images inside <header>.
	if index < 3 || hasAncestorTag(img, "header") {
		return nil
	}

	loading, _ := getAttr(img, "loading")
	if strings.EqualFold(loading, "lazy") {
		return nil
	}

	src, _ := getAttr(img, "src")
	return []model.Issue{{
		CheckName: "images/missing-lazy-loading",
		Severity:  model.SeverityInfo,
		URL:       pageURL,
		Message:   fmt.Sprintf("image is missing loading=\"lazy\" attribute: %s", src),
	}}
}

func (c *ImageChecker) checkMissingDimensions(img *html.Node, pageURL string) []model.Issue {
	_, hasWidth := getAttr(img, "width")
	_, hasHeight := getAttr(img, "height")

	// Only flag if both are missing.
	if hasWidth || hasHeight {
		return nil
	}

	src, _ := getAttr(img, "src")
	return []model.Issue{{
		CheckName: "images/missing-dimensions",
		Severity:  model.SeverityWarning,
		URL:       pageURL,
		Message:   fmt.Sprintf("image is missing width and height attributes: %s", src),
	}}
}

func (c *ImageChecker) checkMissingResponsive(img *html.Node, pageURL string) []model.Issue {
	src, _ := getAttr(img, "src")
	if !strings.HasPrefix(src, "http") {
		return nil
	}

	// Skip small decorative images (empty alt + role="presentation").
	alt, _ := getAttr(img, "alt")
	role, _ := getAttr(img, "role")
	if alt == "" && role == "presentation" {
		return nil
	}

	_, hasSrcset := getAttr(img, "srcset")
	_, hasSizes := getAttr(img, "sizes")
	if hasSrcset || hasSizes {
		return nil
	}

	return []model.Issue{{
		CheckName: "images/missing-responsive",
		Severity:  model.SeverityInfo,
		URL:       pageURL,
		Message:   fmt.Sprintf("image is missing srcset and sizes attributes: %s", src),
	}}
}
