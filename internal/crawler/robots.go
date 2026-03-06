package crawler

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// robotsRule represents a single Allow or Disallow directive.
type robotsRule struct {
	path  string
	allow bool
}

// robotsRules holds the parsed rules for a single host.
type robotsRules struct {
	rules []robotsRule
}

// RobotsChecker fetches, parses, and caches robots.txt files per host.
type RobotsChecker struct {
	userAgent string

	mu    sync.RWMutex
	cache map[string]*robotsRules
}

// NewRobotsChecker creates a new RobotsChecker that matches against the given user agent.
func NewRobotsChecker(userAgent string) *RobotsChecker {
	return &RobotsChecker{
		userAgent: userAgent,
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

	return rules.isAllowed(parsed.Path)
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return rc.cacheEmpty(host)
	}
	defer func() { _ = resp.Body.Close() }()

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

// parseRobotsTxt parses robots.txt content, extracting rules for the given user agent.
func parseRobotsTxt(r io.Reader, userAgent string) *robotsRules {
	scanner := bufio.NewScanner(r)
	ua := strings.ToLower(userAgent)

	var allRules robotsRules
	var specificRules robotsRules
	inMatchingSection := false
	inWildcardSection := false
	foundSpecific := false

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
			agent := strings.ToLower(value)
			if agent == "*" {
				inWildcardSection = true
				inMatchingSection = false
			} else if strings.Contains(ua, agent) || strings.Contains(agent, ua) {
				inMatchingSection = true
				inWildcardSection = false
				foundSpecific = true
			} else {
				inMatchingSection = false
				inWildcardSection = false
			}
			continue
		}

		if key == "disallow" && (inMatchingSection || inWildcardSection) {
			if value == "" {
				continue
			}
			rule := robotsRule{path: value, allow: false}
			if inMatchingSection {
				specificRules.rules = append(specificRules.rules, rule)
			} else {
				allRules.rules = append(allRules.rules, rule)
			}
		}

		if key == "allow" && (inMatchingSection || inWildcardSection) {
			if value == "" {
				continue
			}
			rule := robotsRule{path: value, allow: true}
			if inMatchingSection {
				specificRules.rules = append(specificRules.rules, rule)
			} else {
				allRules.rules = append(allRules.rules, rule)
			}
		}
	}

	if foundSpecific {
		return &specificRules
	}
	return &allRules
}

// isAllowed checks if the given path is allowed by the rules.
// Allow rules take precedence over Disallow rules.
func (rr *robotsRules) isAllowed(path string) bool {
	if len(rr.rules) == 0 {
		return true
	}

	// Find the most specific matching rule.
	// Longer prefix matches take precedence.
	// Among equal-length matches, Allow wins.
	bestLen := -1
	allowed := true

	for _, rule := range rr.rules {
		if strings.HasPrefix(path, rule.path) && len(rule.path) > bestLen {
			bestLen = len(rule.path)
			allowed = rule.allow
		} else if strings.HasPrefix(path, rule.path) && len(rule.path) == bestLen && rule.allow {
			allowed = true
		}
	}

	if bestLen == -1 {
		return true
	}

	return allowed
}
