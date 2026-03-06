package audit

import (
	"bytes"
	"strings"
	"sync"

	"github.com/meysam81/scry/internal/model"
	"golang.org/x/net/html"
)

// docCache caches parsed HTML documents keyed by the body content hash.
var docCache sync.Map

// clearDocCache removes all cached HTML documents to free memory.
func clearDocCache() {
	docCache.Range(func(key, _ any) bool {
		docCache.Delete(key)
		return true
	})
}

// parseHTMLDoc parses body bytes into an *html.Node tree.
// Results are cached using a sync.Map keyed by body slice identity.
func parseHTMLDoc(body []byte) (*html.Node, error) {
	key := string(body)
	if cached, ok := docCache.Load(key); ok {
		return cached.(*html.Node), nil
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
	return strings.Contains(page.ContentType, "text/html")
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
