// Package safenet provides URL safety checks to prevent SSRF attacks.
package safenet

import (
	"net"
	"net/url"
)

// IsSafeURL resolves the hostname of rawURL and rejects private, loopback,
// and link-local addresses to prevent server-side request forgery.
func IsSafeURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := parsed.Hostname()
	if host == "" {
		return false
	}

	// Resolve to IP addresses.
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return false
		}
	}

	return true
}
