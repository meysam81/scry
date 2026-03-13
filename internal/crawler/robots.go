package crawler

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/meysam81/scry/internal/logger"
)

// internalClient is a shared HTTP client with a reasonable timeout for
// internal fetches (robots.txt, sitemaps) that should not use http.DefaultClient.
var internalClient = &http.Client{Timeout: 15 * time.Second}

// robotsRule represents a single Allow or Disallow directive.
// When the path contains wildcard characters (* or $), a compiled
// regexp is stored for matching; otherwise plain prefix matching is used.
type robotsRule struct {
	path  string
	allow bool
	re    *regexp.Regexp // non-nil when path contains * or trailing $
}

// robotsRules holds the parsed rules for a single host.
type robotsRules struct {
	rules []robotsRule
}

// RobotsChecker fetches, parses, and caches robots.txt files per host.
type RobotsChecker struct {
	userAgent string
	log       logger.Logger

	mu    sync.RWMutex
	cache map[string]*robotsRules
}

// NewRobotsChecker creates a new RobotsChecker that matches against the given user agent.
func NewRobotsChecker(userAgent string, l logger.Logger) *RobotsChecker {
	return &RobotsChecker{
		userAgent: userAgent,
		log:       l,
		cache:     make(map[string]*robotsRules),
	}
}

// IsAllowed reports whether the given URL is permitted by the host's robots.txt.
func (rc *RobotsChecker) IsAllowed(ctx context.Context, rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return true
	}

	host := parsed.Host
	rules := rc.getRules(host)
	if rules == nil {
		rules = rc.fetchAndCache(ctx, parsed.Scheme, host)
	}

	// Use the full path including query string so that $ anchors
	// in robots.txt patterns can distinguish /file.pdf from /file.pdf?v=1.
	matchPath := parsed.Path
	if parsed.RawQuery != "" {
		matchPath = parsed.Path + "?" + parsed.RawQuery
	}
	return rules.isAllowed(matchPath)
}

// getRules returns cached rules for a host, or nil if not cached.
func (rc *RobotsChecker) getRules(host string) *robotsRules {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.cache[host]
}

// fetchAndCache fetches robots.txt for the host, parses it, caches the result, and returns it.
func (rc *RobotsChecker) fetchAndCache(ctx context.Context, scheme, host string) *robotsRules {
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", scheme, host)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return rc.cacheEmpty(host)
	}

	resp, err := internalClient.Do(req)
	if err != nil {
		return rc.cacheEmpty(host)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			rc.log.Warn().Err(err).Str("host", host).Msg("robots.txt body close failed")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return rc.cacheEmpty(host)
	}

	rules := parseRobotsTxt(resp.Body, rc.userAgent)

	rc.mu.Lock()
	rc.cache[host] = rules
	rc.mu.Unlock()

	return rules
}

// cacheEmpty stores an empty ruleset for the host and returns it.
func (rc *RobotsChecker) cacheEmpty(host string) *robotsRules {
	rules := &robotsRules{}
	rc.mu.Lock()
	rc.cache[host] = rules
	rc.mu.Unlock()
	return rules
}

// compileRobotsPattern converts a robots.txt path pattern into a compiled
// regexp when the pattern contains wildcard characters (* or trailing $).
// Returns nil when the pattern uses only simple prefix semantics.
func compileRobotsPattern(pattern string) *regexp.Regexp {
	if !strings.Contains(pattern, "*") && !strings.HasSuffix(pattern, "$") {
		return nil
	}

	// Build a regex from the pattern:
	//  - Escape regex metacharacters (except our special * and $)
	//  - Replace * with .*
	//  - Preserve trailing $ as end-of-string anchor
	hasEndAnchor := strings.HasSuffix(pattern, "$")
	if hasEndAnchor {
		pattern = pattern[:len(pattern)-1]
	}

	var b strings.Builder
	b.WriteString("^")
	for _, part := range strings.Split(pattern, "*") {
		b.WriteString(regexp.QuoteMeta(part))
		b.WriteString(".*")
	}
	// Remove the trailing .* that was added after the last split segment.
	s := b.String()
	s = s[:len(s)-2]

	if hasEndAnchor {
		s += "$"
	}

	re, err := regexp.Compile(s)
	if err != nil {
		return nil
	}
	return re
}

// parseRobotsTxt parses robots.txt content, extracting rules for the given user agent.
// Consecutive User-agent lines form a group that shares subsequent directives.
func parseRobotsTxt(r io.Reader, userAgent string) *robotsRules {
	scanner := bufio.NewScanner(r)
	ua := strings.ToLower(userAgent)

	var allRules robotsRules
	var specificRules robotsRules
	foundSpecific := false

	// Track state for the current UA group.
	inUABlock := false     // true while reading consecutive User-agent lines
	groupHasMatch := false // true if any UA line in the current group matches our agent
	groupHasWild := false  // true if any UA line in the current group is "*"

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Strip comments.
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])

		if key == "user-agent" {
			if !inUABlock {
				// Starting a new group — reset group flags.
				inUABlock = true
				groupHasMatch = false
				groupHasWild = false
			}
			agent := strings.ToLower(value)
			if agent == "*" {
				groupHasWild = true
			} else if strings.Contains(ua, agent) {
				groupHasMatch = true
				foundSpecific = true
			}
			continue
		}

		// Non-UA directive — transition out of UA block.
		inUABlock = false

		if key != "disallow" && key != "allow" {
			continue
		}
		if !groupHasMatch && !groupHasWild {
			continue
		}
		if value == "" {
			continue
		}

		rule := robotsRule{path: value, allow: key == "allow", re: compileRobotsPattern(value)}
		if groupHasMatch {
			specificRules.rules = append(specificRules.rules, rule)
		} else {
			allRules.rules = append(allRules.rules, rule)
		}
	}

	if foundSpecific {
		return &specificRules
	}
	return &allRules
}

// matchesRule reports whether the given path matches a robots.txt rule.
// For rules with compiled regexps (wildcard patterns), regex matching is used.
// Otherwise, plain prefix matching is applied.
func matchesRule(rule robotsRule, path string) bool {
	if rule.re != nil {
		return rule.re.MatchString(path)
	}
	return strings.HasPrefix(path, rule.path)
}

// isAllowed checks if the given path is allowed by the rules.
// Allow rules take precedence over Disallow rules.
func (rr *robotsRules) isAllowed(path string) bool {
	if len(rr.rules) == 0 {
		return true
	}

	// Find the most specific matching rule.
	// Longer pattern length takes precedence.
	// Among equal-length matches, Allow wins.
	bestLen := -1
	allowed := true

	for _, rule := range rr.rules {
		if !matchesRule(rule, path) {
			continue
		}
		if len(rule.path) > bestLen {
			bestLen = len(rule.path)
			allowed = rule.allow
		} else if len(rule.path) == bestLen && rule.allow {
			allowed = true
		}
	}

	if bestLen == -1 {
		return true
	}

	return allowed
}
