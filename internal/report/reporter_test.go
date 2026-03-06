package report

import (
	"testing"
)

func TestAllReportersContainsExpectedEntries(t *testing.T) {
	reporters := AllReporters()

	expected := []string{"terminal", "json", "csv", "markdown", "html"}
	for _, name := range expected {
		r, ok := reporters[name]
		if !ok {
			t.Fatalf("AllReporters() missing key %q", name)
		}
		if r.Name() != name {
			t.Errorf("reporter[%q].Name() = %q, want %q", name, r.Name(), name)
		}
	}

	if len(reporters) != len(expected) {
		t.Errorf("AllReporters() has %d entries, want %d", len(reporters), len(expected))
	}
}

func TestAllReportersImplementInterface(t *testing.T) {
	for name, r := range AllReporters() {
		// Verify the concrete value satisfies Reporter.
		_ = Reporter(r)
		if r.Name() == "" {
			t.Errorf("reporter %q returned empty Name()", name)
		}
	}
}
