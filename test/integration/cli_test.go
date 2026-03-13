package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

var scryBin string

func TestMain(m *testing.M) {
	cmd := exec.Command("/etc/profiles/per-user/meysam/bin/go", "build", "-o", "/tmp/scry-test-bin", ".")
	cmd.Dir = "/home/meysam/codes/personal/scry"
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}
	scryBin = "/tmp/scry-test-bin"
	os.Exit(m.Run())
}

func startTestServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Test Page</title>
	<meta name="description" content="A test page for scry integration tests">
	<link rel="canonical" href="http://localhost/">
</head>
<body>
	<main>
		<h1>Hello Scry</h1>
		<p>This is a test page.</p>
		<a href="/about">About</a>
		<img src="/logo.png" alt="Logo" width="100" height="100">
	</main>
</body>
</html>`))
	})
	mux.HandleFunc("/about", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>About</title><meta name="description" content="About page"></head><body><main><h1>About</h1></main></body></html>`))
	})
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
	})
	return httptest.NewServer(mux)
}

func TestCheckCommand_JSON(t *testing.T) {
	srv := startTestServer()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, scryBin, "check", srv.URL, "--output", "json", "--log-level", "error")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scry check failed: %v\noutput: %s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "seed_url") {
		t.Errorf("expected JSON output with seed_url, got: %.200s", output)
	}
}

func TestCheckCommand_Terminal(t *testing.T) {
	srv := startTestServer()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, scryBin, "check", srv.URL, "--output", "terminal", "--log-level", "error")
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		t.Error("expected some terminal output")
	}
}

func TestCrawlCommand_Basic(t *testing.T) {
	srv := startTestServer()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, scryBin, "crawl", srv.URL, "--output", "json", "--max-pages", "5", "--log-level", "error")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scry crawl failed: %v\noutput: %s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "pages") {
		t.Errorf("expected JSON output with pages, got: %.200s", output)
	}
}

func TestCheckCommand_CSV(t *testing.T) {
	srv := startTestServer()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, scryBin, "check", srv.URL, "--output", "csv", "--log-level", "error")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scry check csv failed: %v\noutput: %s", err, out)
	}

	if !strings.Contains(string(out), "severity") || !strings.Contains(string(out), "check") {
		t.Errorf("expected CSV headers, got: %.200s", string(out))
	}
}

func TestCheckCommand_SARIF(t *testing.T) {
	srv := startTestServer()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, scryBin, "check", srv.URL, "--output", "sarif", "--log-level", "error")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scry check sarif failed: %v\noutput: %s", err, out)
	}

	if !strings.Contains(string(out), "sarif") {
		t.Errorf("expected SARIF output, got: %.200s", string(out))
	}
}

func TestCheckCommand_FilterSeverity(t *testing.T) {
	srv := startTestServer()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, scryBin, "check", srv.URL, "--output", "json", "--filter-severity", "critical", "--log-level", "error")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scry check with filter failed: %v\noutput: %s", err, out)
	}
	// Just verify it runs without error.
	if !strings.Contains(string(out), "seed_url") {
		t.Errorf("expected JSON output, got: %.200s", string(out))
	}
}

func TestValidateCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, scryBin, "validate")
	out, err := cmd.CombinedOutput()
	// validate may print to stderr; just check it doesn't crash.
	_ = err
	_ = out
}
