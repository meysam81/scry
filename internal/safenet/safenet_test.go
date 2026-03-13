package safenet

import (
	"testing"
)

func TestIsSafeURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		// --- Empty / malformed ---
		{name: "empty string", url: "", want: false},
		{name: "bare path no host", url: "/just/a/path", want: false},
		{name: "scheme only", url: "http://", want: false},
		{name: "whitespace", url: "   ", want: false},
		{name: "colon only", url: ":", want: false},

		// --- Loopback addresses ---
		{name: "loopback IPv4", url: "http://127.0.0.1", want: false},
		{name: "loopback IPv4 with port", url: "http://127.0.0.1:8080", want: false},
		{name: "loopback IPv6", url: "http://[::1]", want: false},
		{name: "loopback IPv6 with port", url: "http://[::1]:443", want: false},
		{name: "localhost", url: "http://localhost", want: false},
		{name: "localhost https", url: "https://localhost", want: false},
		{name: "localhost with port", url: "http://localhost:3000", want: false},
		{name: "localhost with path", url: "http://localhost/some/path", want: false},

		// --- Private RFC 1918 ranges ---
		{name: "private 10.x", url: "http://10.0.0.1", want: false},
		{name: "private 10.x high", url: "http://10.255.255.255", want: false},
		{name: "private 172.16.x", url: "http://172.16.0.1", want: false},
		{name: "private 172.31.x", url: "http://172.31.255.255", want: false},
		{name: "private 192.168.x", url: "http://192.168.1.1", want: false},
		{name: "private 192.168.0.x", url: "http://192.168.0.100", want: false},

		// --- Link-local ---
		{name: "link-local IPv4", url: "http://169.254.1.1", want: false},
		{name: "link-local IPv4 low", url: "http://169.254.0.1", want: false},

		// --- Unresolvable hosts ---
		{name: "unresolvable host", url: "http://this-host-does-not-exist-ever.invalid", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSafeURL(tt.url)
			if got != tt.want {
				t.Errorf("IsSafeURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

// TestIsSafeURL_PublicDomains verifies that well-known public domains pass
// the safety check. These tests require real DNS resolution, so they are
// skipped when running with -short (e.g. in CI without network access).
func TestIsSafeURL_PublicDomains(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping DNS-dependent tests in short mode")
	}

	tests := []struct {
		name string
		url  string
	}{
		{name: "example.com https", url: "https://example.com"},
		{name: "example.com http", url: "http://example.com"},
		{name: "example.com with path", url: "https://example.com/page"},
		{name: "example.org", url: "https://example.org"},
		{name: "google.com", url: "https://google.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !IsSafeURL(tt.url) {
				t.Errorf("IsSafeURL(%q) = false, want true for public domain", tt.url)
			}
		})
	}
}
