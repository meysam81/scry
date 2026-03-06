package crawler

import (
	"bytes"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// skippedSchemes lists URL schemes that should not be followed.
var skippedSchemes = map[string]bool{
	"mailto":     true,
	"javascript": true,
	"data":       true,
	"tel":        true,
	"ftp":        true,
}

// ParseHTML extracts links and assets from HTML content.
func ParseHTML(base *url.URL, body []byte) (links []string, assets []string) {
	tokenizer := html.NewTokenizer(bytes.NewReader(body))

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}

		token := tokenizer.Token()
		tag := token.DataAtom.String()

		switch tag {
		case "a":
			if href := getAttr(token, "href"); href != "" {
				if resolved, ok := resolveURL(base, href); ok {
					links = append(links, resolved)
				}
			}
		case "img":
			if src := getAttr(token, "src"); src != "" {
				if resolved, ok := resolveURL(base, src); ok {
					assets = append(assets, resolved)
				}
			}
		case "link":
			if href := getAttr(token, "href"); href != "" {
				if resolved, ok := resolveURL(base, href); ok {
					assets = append(assets, resolved)
				}
			}
		case "script":
			if src := getAttr(token, "src"); src != "" {
				if resolved, ok := resolveURL(base, src); ok {
					assets = append(assets, resolved)
				}
			}
		}
	}

	return links, assets
}

// getAttr returns the value of the named attribute, or empty string if not found.
func getAttr(t html.Token, name string) string {
	for _, a := range t.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

// resolveURL resolves a potentially relative URL against the base, strips fragments,
// and filters out non-http schemes.
func resolveURL(base *url.URL, raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false
	}

	// Skip non-navigable schemes.
	if parsed.Scheme != "" && skippedSchemes[strings.ToLower(parsed.Scheme)] {
		return "", false
	}

	resolved := base.ResolveReference(parsed)

	// Only keep http(s) URLs.
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return "", false
	}

	// Strip fragment.
	resolved.Fragment = ""
	resolved.RawFragment = ""

	return resolved.String(), true
}
