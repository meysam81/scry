package checks

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meysam81/scry/core/model"
)

func TestTLSChecker_Name(t *testing.T) {
	checker := NewTLSChecker()
	if checker.Name() != "tls" {
		t.Fatalf("expected name 'tls', got %q", checker.Name())
	}
}

func TestTLSChecker_SkipNonHTTPS(t *testing.T) {
	checker := NewTLSChecker()
	ctx := context.Background()

	tests := []struct {
		name string
		page *model.Page
	}{
		{
			name: "http url",
			page: &model.Page{URL: "http://example.com", StatusCode: 200},
		},
		{
			name: "zero status code",
			page: &model.Page{URL: "https://example.com", StatusCode: 0},
		},
		{
			name: "empty url",
			page: &model.Page{URL: "", StatusCode: 200},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checker.Check(ctx, tt.page)
			if len(issues) != 0 {
				t.Fatalf("expected no issues, got %+v", issues)
			}
		})
	}
}

func TestTLSChecker_ValidCert(t *testing.T) {
	ts := httptest.NewTLSServer(nil)
	defer ts.Close()

	checker := NewTLSChecker()
	// Override the dialer to use the test server's TLS config.
	checker.dialer = func(network, addr string, _ *tls.Config) (*tls.Conn, error) {
		return tls.Dial(network, ts.Listener.Addr().String(), &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
			MinVersion:         tls.VersionTLS10,
		})
	}

	page := &model.Page{
		URL:        ts.URL + "/page",
		StatusCode: 200,
	}

	issues := checker.Check(context.Background(), page)

	// httptest.NewTLSServer generates self-signed certs.
	// We should only get self-signed and possibly hostname-mismatch, but NOT expired or expiring-soon.
	for _, iss := range issues {
		if iss.CheckName == "tls/certificate-expired" || iss.CheckName == "tls/certificate-expiring-soon" {
			t.Errorf("did not expect %s for fresh test cert, got %+v", iss.CheckName, iss)
		}
	}
}

func TestTLSChecker_ExpiredCert(t *testing.T) {
	// Create a self-signed certificate that expired yesterday.
	cert, key := generateTestCert(t, "localhost", time.Now().Add(-48*time.Hour), time.Now().Add(-24*time.Hour))
	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		t.Fatalf("failed to load key pair: %v", err)
	}

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Logf("failed to close listener: %v", err)
		}
	}()

	// Accept connections in a goroutine — complete the TLS handshake so
	// the client can read peer certificates before closing.
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			// Complete the TLS handshake on the server side.
			if tlsConn, ok := conn.(*tls.Conn); ok {
				_ = tlsConn.Handshake()
			}
			_ = conn.Close()
		}
	}()

	checker := NewTLSChecker()
	checker.dialer = func(network, addr string, _ *tls.Config) (*tls.Conn, error) {
		return tls.Dial(network, listener.Addr().String(), &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
			MinVersion:         tls.VersionTLS10,
		})
	}

	page := &model.Page{
		URL:        "https://localhost/page",
		StatusCode: 200,
	}

	issues := checker.Check(context.Background(), page)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "tls/certificate-expired" {
			found = true
			if iss.Severity != model.SeverityCritical {
				t.Errorf("expected critical severity, got %s", iss.Severity)
			}
		}
	}
	if !found {
		t.Errorf("expected tls/certificate-expired issue, got %+v", issues)
	}
}

func TestTLSChecker_ExpiringCert(t *testing.T) {
	// Certificate expires in 10 days (within the 30-day window).
	notBefore := time.Now().Add(-24 * time.Hour)
	notAfter := time.Now().Add(10 * 24 * time.Hour)

	cert, key := generateTestCert(t, "localhost", notBefore, notAfter)
	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		t.Fatalf("failed to load key pair: %v", err)
	}

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Logf("failed to close listener: %v", err)
		}
	}()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			if tlsConn, ok := conn.(*tls.Conn); ok {
				_ = tlsConn.Handshake()
			}
			_ = conn.Close()
		}
	}()

	checker := NewTLSChecker()
	checker.dialer = func(network, addr string, _ *tls.Config) (*tls.Conn, error) {
		return tls.Dial(network, listener.Addr().String(), &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
			MinVersion:         tls.VersionTLS10,
		})
	}

	page := &model.Page{
		URL:        "https://localhost/page",
		StatusCode: 200,
	}

	issues := checker.Check(context.Background(), page)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "tls/certificate-expiring-soon" {
			found = true
			if iss.Severity != model.SeverityWarning {
				t.Errorf("expected warning severity, got %s", iss.Severity)
			}
			if !strings.Contains(iss.Message, "expires in") {
				t.Errorf("expected message containing 'expires in', got %q", iss.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected tls/certificate-expiring-soon issue, got %+v", issues)
	}
}

func TestTLSChecker_SelfSigned(t *testing.T) {
	ts := httptest.NewTLSServer(nil)
	defer ts.Close()

	checker := NewTLSChecker()
	checker.dialer = func(network, addr string, _ *tls.Config) (*tls.Conn, error) {
		return tls.Dial(network, ts.Listener.Addr().String(), &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
			MinVersion:         tls.VersionTLS10,
		})
	}

	page := &model.Page{
		URL:        ts.URL + "/page",
		StatusCode: 200,
	}

	issues := checker.Check(context.Background(), page)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "tls/self-signed" {
			found = true
			if iss.Severity != model.SeverityWarning {
				t.Errorf("expected warning severity, got %s", iss.Severity)
			}
		}
	}
	if !found {
		t.Errorf("expected tls/self-signed issue, got %+v", issues)
	}
}

func TestTLSChecker_HostnameMismatch(t *testing.T) {
	// Create a cert valid for "other.example.com" but we connect as "localhost".
	cert, key := generateTestCert(t, "other.example.com", time.Now().Add(-1*time.Hour), time.Now().Add(365*24*time.Hour))
	tlsCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		t.Fatalf("failed to load key pair: %v", err)
	}

	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	})
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Logf("failed to close listener: %v", err)
		}
	}()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			if tlsConn, ok := conn.(*tls.Conn); ok {
				_ = tlsConn.Handshake()
			}
			_ = conn.Close()
		}
	}()

	checker := NewTLSChecker()
	checker.dialer = func(network, addr string, _ *tls.Config) (*tls.Conn, error) {
		return tls.Dial(network, listener.Addr().String(), &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
			MinVersion:         tls.VersionTLS10,
		})
	}

	page := &model.Page{
		URL:        "https://localhost/page",
		StatusCode: 200,
	}

	issues := checker.Check(context.Background(), page)

	found := false
	for _, iss := range issues {
		if iss.CheckName == "tls/hostname-mismatch" {
			found = true
			if iss.Severity != model.SeverityCritical {
				t.Errorf("expected critical severity, got %s", iss.Severity)
			}
			if !strings.Contains(iss.Message, "localhost") {
				t.Errorf("expected message containing 'localhost', got %q", iss.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected tls/hostname-mismatch issue, got %+v", issues)
	}
}

func TestTLSChecker_CacheHit(t *testing.T) {
	dialCount := 0
	ts := httptest.NewTLSServer(nil)
	defer ts.Close()

	checker := NewTLSChecker()
	checker.dialer = func(network, addr string, _ *tls.Config) (*tls.Conn, error) {
		dialCount++
		return tls.Dial(network, ts.Listener.Addr().String(), &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
			MinVersion:         tls.VersionTLS10,
		})
	}

	page1 := &model.Page{
		URL:        ts.URL + "/page1",
		StatusCode: 200,
	}
	page2 := &model.Page{
		URL:        ts.URL + "/page2",
		StatusCode: 200,
	}

	ctx := context.Background()
	issues1 := checker.Check(ctx, page1)
	issues2 := checker.Check(ctx, page2)

	if dialCount != 1 {
		t.Errorf("expected 1 TLS dial (cached), got %d", dialCount)
	}

	// Both should return the same set of check names.
	if len(issues1) != len(issues2) {
		t.Errorf("cached result length mismatch: %d vs %d", len(issues1), len(issues2))
	}
}

func TestTLSVersionName(t *testing.T) {
	tests := []struct {
		version uint16
		want    string
	}{
		{tls.VersionTLS10, "TLS 1.0"},
		{tls.VersionTLS11, "TLS 1.1"},
		{tls.VersionTLS12, "TLS 1.2"},
		{tls.VersionTLS13, "TLS 1.3"},
		{0x0200, "unknown (0x0200)"},
	}

	for _, tt := range tests {
		got := tlsVersionName(tt.version)
		if got != tt.want {
			t.Errorf("tlsVersionName(0x%04x) = %q, want %q", tt.version, got, tt.want)
		}
	}
}

func TestRewriteURL(t *testing.T) {
	input := []model.Issue{
		{URL: "https://original.com", CheckName: "tls/self-signed"},
	}
	result := rewriteURL(input, "https://new.com/page")
	if len(result) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result))
	}
	if result[0].URL != "https://new.com/page" {
		t.Errorf("expected URL rewritten to 'https://new.com/page', got %q", result[0].URL)
	}
	// Original should not be mutated.
	if input[0].URL != "https://original.com" {
		t.Errorf("original issue URL was mutated to %q", input[0].URL)
	}
}

func TestRewriteURL_Empty(t *testing.T) {
	result := rewriteURL(nil, "https://example.com")
	if result != nil {
		t.Fatalf("expected nil, got %+v", result)
	}
}

// generateTestCert creates a self-signed PEM certificate and private key for testing.
func generateTestCert(t *testing.T, hostname string, notBefore, notAfter time.Time) (certPEM, keyPEM []byte) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: hostname},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{hostname},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},

		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("failed to marshal key: %v", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return certPEM, keyPEM
}
