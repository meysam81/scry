package checks

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/meysam81/scry/core/model"
)

const (
	tlsDialTimeout      = 5 * time.Second
	tlsExpiryWarnWindow = 30 * 24 * time.Hour // 30 days
)

// tlsCacheEntry holds the cached TLS inspection result for a hostname.
type tlsCacheEntry struct {
	issues []model.Issue
}

// TLSChecker inspects TLS certificates of crawled HTTPS pages.
type TLSChecker struct {
	cache   sync.Map // hostname -> tlsCacheEntry
	nowFunc func() time.Time
	dialer  func(network, addr string, config *tls.Config) (*tls.Conn, error)
}

// NewTLSChecker returns a new TLSChecker.
func NewTLSChecker() *TLSChecker {
	return &TLSChecker{
		nowFunc: time.Now,
		dialer:  defaultTLSDial,
	}
}

// Name returns the checker name.
func (c *TLSChecker) Name() string { return "tls" }

// Check inspects the TLS certificate of the page's host. Results are cached
// per hostname so repeated pages on the same host don't trigger redundant
// TLS handshakes.
func (c *TLSChecker) Check(ctx context.Context, page *model.Page) []model.Issue {
	if page.StatusCode == 0 {
		return nil
	}

	parsed, err := url.Parse(page.URL)
	if err != nil || !strings.EqualFold(parsed.Scheme, "https") {
		return nil
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return nil
	}

	// Return cached result if available.
	if cached, ok := c.cache.Load(hostname); ok {
		entry, ok := cached.(tlsCacheEntry)
		if !ok {
			return nil
		}
		return rewriteURL(entry.issues, page.URL)
	}

	port := parsed.Port()
	if port == "" {
		port = "443"
	}
	addr := net.JoinHostPort(hostname, port)

	issues := c.inspectTLS(ctx, hostname, addr, page.URL)

	c.cache.Store(hostname, tlsCacheEntry{issues: issues})
	return rewriteURL(issues, page.URL)
}

// inspectTLS performs the TLS handshake and returns issues found.
func (c *TLSChecker) inspectTLS(_ context.Context, hostname, addr, pageURL string) []model.Issue {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,             //nolint:gosec // nosemgrep: bypass-tls-verification -- intentional: audit tool must connect to inspect certs
		MinVersion:         tls.VersionTLS10, // nosemgrep: disallow-old-tls-versions -- intentional: audit tool must detect weak TLS versions
	}

	conn, err := c.dialer("tcp", addr, tlsConfig)
	if err != nil {
		return nil // can't connect; not a TLS-specific issue
	}
	defer func() {
		if err := conn.Close(); err != nil {
			getAuditLogger().Warn().Err(err).Str("addr", addr).Msg("tls conn close failed")
		}
	}()

	state := conn.ConnectionState()
	var issues []model.Issue

	// Check TLS version.
	if state.Version < tls.VersionTLS12 {
		issues = append(issues, model.Issue{
			CheckName: "tls/weak-protocol",
			Severity:  model.SeverityWarning,
			URL:       pageURL,
			Message:   fmt.Sprintf("server negotiated %s, minimum recommended is TLS 1.2", tlsVersionName(state.Version)),
		})
	}

	// Inspect peer certificates.
	if len(state.PeerCertificates) == 0 {
		return issues
	}

	cert := state.PeerCertificates[0]
	now := c.nowFunc()

	// Check expired certificate.
	if now.After(cert.NotAfter) {
		issues = append(issues, model.Issue{
			CheckName: "tls/certificate-expired",
			Severity:  model.SeverityCritical,
			URL:       pageURL,
			Message:   fmt.Sprintf("certificate expired on %s", cert.NotAfter.Format(time.DateOnly)),
		})
	} else if cert.NotAfter.Sub(now) < tlsExpiryWarnWindow {
		// Check expiring soon (only if not already expired).
		daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)
		issues = append(issues, model.Issue{
			CheckName: "tls/certificate-expiring-soon",
			Severity:  model.SeverityWarning,
			URL:       pageURL,
			Message:   fmt.Sprintf("certificate expires in %d days (%s)", daysLeft, cert.NotAfter.Format(time.DateOnly)),
		})
	}

	// Check self-signed certificate.
	if isSelfSigned(cert) {
		issues = append(issues, model.Issue{
			CheckName: "tls/self-signed",
			Severity:  model.SeverityWarning,
			URL:       pageURL,
			Message:   "certificate is self-signed",
		})
	}

	// Check hostname mismatch.
	if err := cert.VerifyHostname(hostname); err != nil {
		issues = append(issues, model.Issue{
			CheckName: "tls/hostname-mismatch",
			Severity:  model.SeverityCritical,
			URL:       pageURL,
			Message:   fmt.Sprintf("certificate does not match hostname %q: %v", hostname, err),
		})
	}

	return issues
}

// isSelfSigned returns true if the certificate appears to have been signed by
// itself. It uses structural checks (Issuer == Subject, matching key IDs) which
// reliably identify self-signed certificates without requiring the cert to be
// marked as a CA (which CheckSignatureFrom needs in modern Go).
func isSelfSigned(cert *x509.Certificate) bool {
	if cert.Issuer.String() != cert.Subject.String() {
		return false
	}
	// If authority and subject key identifiers are both present, they must match.
	if len(cert.AuthorityKeyId) > 0 && len(cert.SubjectKeyId) > 0 {
		return bytes.Equal(cert.AuthorityKeyId, cert.SubjectKeyId)
	}
	return true
}

// tlsVersionName returns a human-readable name for a TLS version constant.
func tlsVersionName(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("unknown (0x%04x)", v)
	}
}

// rewriteURL returns a copy of issues with the URL field set to pageURL.
// This ensures cached issues carry the correct page URL.
func rewriteURL(issues []model.Issue, pageURL string) []model.Issue {
	if len(issues) == 0 {
		return nil
	}
	out := make([]model.Issue, len(issues))
	copy(out, issues)
	for i := range out {
		out[i].URL = pageURL
	}
	return out
}

// defaultTLSDial performs a TLS dial with the configured timeout.
func defaultTLSDial(network, addr string, config *tls.Config) (*tls.Conn, error) {
	dialer := &net.Dialer{Timeout: tlsDialTimeout}
	return tls.DialWithDialer(dialer, network, addr, config)
}
