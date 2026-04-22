package analysis

import (
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/meysam81/scry/core/model"
)

// Technology represents a detected technology on a website.
type Technology struct {
	Name     string `json:"name"`
	Category string `json:"category"` // cms, framework, analytics, cdn, etc.
	Version  string `json:"version,omitempty"`
	Evidence string `json:"evidence"` // what signal detected it
}

// signature defines how to detect a single technology.
type signature struct {
	Name     string
	Category string
	// bodyPatterns are strings to search for in the HTML body.
	bodyPatterns []string
	// bodyRegexps are compiled regexps to apply to the body (used for version extraction).
	bodyRegexps []*regexp.Regexp
	// assetPatterns are substrings to look for in asset URLs.
	assetPatterns []string
	// headerChecks test HTTP response headers.
	headerChecks []headerCheck
}

// headerCheck defines a header-based detection rule.
type headerCheck struct {
	header   string // header name (lowercase)
	contains string // substring to look for (lowercase); empty means header existence suffices
}

// signatures is the full set of technology detection rules.
var signatures = []signature{
	// ── CMS ──────────────────────────────────────────────────────────────
	{
		Name:     "WordPress",
		Category: "cms",
		bodyPatterns: []string{
			`<meta name="generator" content="WordPress`,
			"/wp-content/",
		},
		assetPatterns: []string{"/wp-content/"},
	},
	{
		Name:     "Ghost",
		Category: "cms",
		bodyPatterns: []string{
			`<meta name="generator" content="Ghost`,
			"ghost-",
		},
	},
	{
		Name:     "Hugo",
		Category: "cms",
		bodyPatterns: []string{
			`<meta name="generator" content="Hugo`,
		},
	},
	{
		Name:     "Next.js",
		Category: "cms",
		bodyPatterns: []string{
			"__NEXT_DATA__",
		},
		assetPatterns: []string{"/_next/"},
	},
	{
		Name:     "Gatsby",
		Category: "cms",
		bodyPatterns: []string{
			"gatsby-",
		},
		assetPatterns: []string{"/static/"},
	},
	{
		Name:     "Astro",
		Category: "cms",
		bodyPatterns: []string{
			`<meta name="generator" content="Astro`,
		},
	},
	{
		Name:     "Nuxt",
		Category: "cms",
		bodyPatterns: []string{
			"__NUXT__",
		},
		assetPatterns: []string{"/_nuxt/"},
	},

	// ── Analytics ────────────────────────────────────────────────────────
	{
		Name:     "Google Analytics",
		Category: "analytics",
		bodyPatterns: []string{
			"google-analytics.com",
			"gtag(",
			"ga(",
		},
	},
	{
		Name:     "Google Tag Manager",
		Category: "analytics",
		bodyPatterns: []string{
			"googletagmanager.com",
		},
	},
	{
		Name:     "Plausible",
		Category: "analytics",
		bodyPatterns: []string{
			"plausible.io",
		},
	},
	{
		Name:     "Fathom",
		Category: "analytics",
		bodyPatterns: []string{
			"usefathom.com",
		},
	},

	// ── CDN (header-based) ───────────────────────────────────────────────
	{
		Name:     "Cloudflare",
		Category: "cdn",
		headerChecks: []headerCheck{
			{header: "cf-ray", contains: ""},
			{header: "server", contains: "cloudflare"},
		},
	},
	{
		Name:     "Vercel",
		Category: "cdn",
		headerChecks: []headerCheck{
			{header: "x-vercel-id", contains: ""},
			{header: "server", contains: "vercel"},
		},
	},
	{
		Name:     "Netlify",
		Category: "cdn",
		headerChecks: []headerCheck{
			{header: "x-nf-request-id", contains: ""},
			{header: "server", contains: "netlify"},
		},
	},
	{
		Name:     "AWS CloudFront",
		Category: "cdn",
		headerChecks: []headerCheck{
			{header: "x-amz-cf-id", contains: ""},
		},
	},
	{
		Name:     "Fastly",
		Category: "cdn",
		headerChecks: []headerCheck{
			{header: "x-served-by", contains: "cache-"},
		},
	},

	// ── Frameworks ───────────────────────────────────────────────────────
	{
		Name:     "React",
		Category: "framework",
		bodyPatterns: []string{
			"data-reactroot",
			"__REACT",
		},
	},
	{
		Name:     "Vue",
		Category: "framework",
		bodyPatterns: []string{
			"data-v-",
		},
	},
	{
		Name:     "Tailwind CSS",
		Category: "framework",
		assetPatterns: []string{
			"tailwindcss",
		},
	},
	{
		Name:     "Bootstrap",
		Category: "framework",
		assetPatterns: []string{
			"bootstrap",
		},
	},
}

// DetectTechnologies scans the provided pages for known technology signatures
// and returns a deduplicated, sorted list of detected technologies.
//
// Body and asset checks use primarily the first (homepage) page.
// Header checks are applied across all pages for CDN detection.
func DetectTechnologies(pages []*model.Page) []Technology {
	if len(pages) == 0 {
		return nil
	}

	seen := make(map[string]bool) // key = Name
	var result []Technology

	add := func(t Technology) {
		if seen[t.Name] {
			return
		}
		seen[t.Name] = true
		result = append(result, t)
	}

	// Use first page for body/asset checks (typically the homepage).
	homepage := pages[0]
	bodyLower := strings.ToLower(string(homepage.Body))

	for _, sig := range signatures {
		if detected, evidence := matchBody(bodyLower, sig); detected {
			add(Technology{
				Name:     sig.Name,
				Category: sig.Category,
				Evidence: evidence,
			})
			continue
		}
		if detected, evidence := matchAssets(homepage.Assets, sig); detected {
			add(Technology{
				Name:     sig.Name,
				Category: sig.Category,
				Evidence: evidence,
			})
			continue
		}
	}

	// Header checks across all pages (important for CDN detection).
	for _, sig := range signatures {
		if seen[sig.Name] {
			continue
		}
		for _, page := range pages {
			if detected, evidence := matchHeaders(page.Headers, sig); detected {
				add(Technology{
					Name:     sig.Name,
					Category: sig.Category,
					Evidence: evidence,
				})
				break
			}
		}
	}

	// Sort by category then name for deterministic output.
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		return result[i].Name < result[j].Name
	})

	return result
}

// matchBody checks whether any body pattern in the signature matches the
// lowercase body string. Returns true with the matching evidence string.
func matchBody(bodyLower string, sig signature) (bool, string) {
	for _, pat := range sig.bodyPatterns {
		if strings.Contains(bodyLower, strings.ToLower(pat)) {
			return true, "body contains " + pat
		}
	}
	for _, re := range sig.bodyRegexps {
		if loc := re.FindStringIndex(bodyLower); loc != nil {
			return true, "body matches " + re.String()
		}
	}
	return false, ""
}

// matchAssets checks whether any asset URL contains a signature pattern.
func matchAssets(assets []string, sig signature) (bool, string) {
	for _, pat := range sig.assetPatterns {
		patLower := strings.ToLower(pat)
		for _, asset := range assets {
			if strings.Contains(strings.ToLower(asset), patLower) {
				return true, "asset URL contains " + pat
			}
		}
	}
	return false, ""
}

// matchHeaders checks whether any header check in the signature matches the
// response headers.
func matchHeaders(headers http.Header, sig signature) (bool, string) {
	if headers == nil {
		return false, ""
	}
	for _, hc := range sig.headerChecks {
		vals := headers.Values(hc.header)
		if len(vals) == 0 {
			continue
		}
		if hc.contains == "" {
			// Presence check.
			return true, "header " + hc.header + " present"
		}
		for _, v := range vals {
			if strings.Contains(strings.ToLower(v), hc.contains) {
				return true, "header " + hc.header + " contains " + hc.contains
			}
		}
	}
	return false, ""
}
