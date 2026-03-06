// Package cmdutil provides shared utilities for scry subcommands.
package cmdutil

import "strings"

// NormalizeURL adds an https:// prefix if no scheme is present.
func NormalizeURL(u string) string {
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return "https://" + u
	}
	return u
}
