package audit

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/meysam81/scry/internal/logger"
	"github.com/meysam81/scry/internal/model"
	"golang.org/x/net/html"
)

// auditLoggerPtr stores the logger atomically to avoid data races.
// It is set once via setAuditLogger before checks run.
var auditLoggerPtr atomic.Pointer[logger.Logger]

func init() {
	l := logger.Nop()
	auditLoggerPtr.Store(&l)
}

// setAuditLogger atomically sets the audit logger.
func setAuditLogger(l logger.Logger) {
	auditLoggerPtr.Store(&l)
}

// getAuditLogger atomically loads the audit logger.
func getAuditLogger() logger.Logger {
	return *auditLoggerPtr.Load()
}

// docCache caches parsed HTML documents keyed by the body content hash.
var docCache sync.Map

// clearDocCache removes all cached HTML documents to free memory.
func clearDocCache() {
	docCache.Range(func(key, _ any) bool {
		docCache.Delete(key)
		return true
	})
}

// parseHTMLDocLog parses body bytes and logs a warning on failure.
func parseHTMLDocLog(body []byte, pageURL string) *html.Node {
	doc, err := parseHTMLDoc(body)
	if err != nil {
		getAuditLogger().Warn().Err(err).Str("url", pageURL).Msg("html parse failed")
		return nil
	}
	return doc
}

// parseHTMLDoc parses body bytes into an *html.Node tree.
// Results are cached using a sync.Map keyed by SHA-256 hash of the body.
func parseHTMLDoc(body []byte) (*html.Node, error) {
	key := sha256.Sum256(body)
	if cached, ok := docCache.Load(key); ok {
		node, ok := cached.(*html.Node)
		if !ok {
			return nil, fmt.Errorf("docCache: unexpected type %T", cached)
		}
		return node, nil
	}
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	docCache.Store(key, doc)
	return doc, nil
}

// isHTMLContent reports whether the page's ContentType indicates HTML.
func isHTMLContent(page *model.Page) bool {
	return strings.Contains(strings.ToLower(page.ContentType), "text/html")
}

// findNodes recursively finds all elements with the given tag name.
func findNodes(n *html.Node, tag string) []*html.Node {
	var result []*html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == tag {
			result = append(result, node)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return result
}

// getAttr returns the value of the named attribute on a node.
func getAttr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

// findMeta returns the content attribute of the first <meta name="name"> tag.
func findMeta(doc *html.Node, name string) string {
	for _, m := range findNodes(doc, "meta") {
		n, ok := getAttr(m, "name")
		if ok && strings.EqualFold(n, name) {
			v, _ := getAttr(m, "content")
			return v
		}
	}
	return ""
}

// findMetaProperty returns the content attribute of the first <meta property="property"> tag.
func findMetaProperty(doc *html.Node, property string) string {
	for _, m := range findNodes(doc, "meta") {
		p, ok := getAttr(m, "property")
		if ok && strings.EqualFold(p, property) {
			v, _ := getAttr(m, "content")
			return v
		}
	}
	return ""
}
