// Package safenet provides URL safety checks to prevent SSRF attacks.
package safenet

import (
	"context"
	"net"
	"net/url"
	"time"
)

// dnsTimeout is the maximum time allowed for DNS resolution.
const dnsTimeout = 5 * time.Second

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

	// Resolve to IP addresses with a timeout to prevent hanging on
	// attacker-controlled DNS servers.
	ctx, cancel := context.WithTimeout(context.Background(), dnsTimeout)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return false
	}

	for _, ip := range ips {
		if ip.IP.IsLoopback() || ip.IP.IsPrivate() || ip.IP.IsLinkLocalUnicast() || ip.IP.IsLinkLocalMulticast() {
			return false
		}
	}

	return true
}
