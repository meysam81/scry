# scry

<!-- TODO: uncomment when ready -->
<!-- ![scry demo](assets/demo.gif) -->

**A fast, thorough website auditor for your terminal.**
94 checks. 11 categories. One command.

[![CI](https://img.shields.io/github/actions/workflow/status/meysam81/scry/ci.yml?branch=main&label=CI&logo=githubactions&logoColor=white)](https://github.com/meysam81/scry/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/meysam81/scry)](https://goreportcard.com/report/github.com/meysam81/scry)
[![Go Reference](https://pkg.go.dev/badge/github.com/meysam81/scry.svg)](https://pkg.go.dev/github.com/meysam81/scry)
[![Latest Release](https://img.shields.io/github/v/release/meysam81/scry?logo=github&label=release)](https://github.com/meysam81/scry/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/meysam81/scry?logo=go)](https://github.com/meysam81/scry/blob/main/go.mod)
[![License](https://img.shields.io/github/license/meysam81/scry)](LICENSE)
[![Downloads](https://img.shields.io/github/downloads/meysam81/scry/total?logo=github&label=downloads)](https://github.com/meysam81/scry/releases)

[![Homebrew](https://img.shields.io/badge/homebrew-meysam81%2Ftap%2Fscry-FBB040?logo=homebrew&logoColor=white)](https://github.com/meysam81/homebrew-tap)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20Windows-blue?logo=linux&logoColor=white)](#quick-start)
[![Last Commit](https://img.shields.io/github/last-commit/meysam81/scry?logo=github)](https://github.com/meysam81/scry/commits/main)
[![GitHub Stars](https://img.shields.io/github/stars/meysam81/scry?style=flat&logo=github)](https://github.com/meysam81/scry/stargazers)
[![Issues](https://img.shields.io/github/issues/meysam81/scry?logo=github)](https://github.com/meysam81/scry/issues)

## Quick Start

Install scry:

```bash
# Homebrew
brew install meysam81/tap/scry

# Go
go install github.com/meysam81/scry@latest
```

Or grab a [pre-built binary](https://github.com/meysam81/scry/releases) for your platform.

Run your first audit:

```bash
scry crawl https://example.com
```

That's it. You'll get a colorized terminal report with a site health score,
categorized issues, and actionable recommendations.

## Why Scry

### For SEO professionals

- **94 audit checks** across SEO, structured data, hreflang, links, images, performance, security, accessibility, health, external links, and TLS
- **Site health score** (0--100) with per-category breakdowns
- **Structured data validation** for 9 Schema.org types (Article, Product, FAQPage, LocalBusiness, BreadcrumbList, Event, Recipe, VideoObject, BlogPosting)
- **Hreflang cross-validation** with return-link and x-default checks
- **Content quality metrics** -- reading level, word count, content-to-HTML ratio, thin content detection
- **Content duplication** -- SimHash near-duplicate and exact-duplicate detection
- **Internal PageRank** -- see which pages concentrate link equity

### For developers

- **CI/CD native** -- SARIF for GitHub PR annotations, JUnit for Jenkins/GitLab, `--fail-on critical` for exit codes
- **Baseline comparison** -- track regressions across deploys with `--save-baseline` / `--compare-baseline`
- **9 output formats** -- terminal, JSON, CSV, Markdown, HTML, SARIF, JUnit, JSONL, PDF
- **Custom rules** -- write your own checks with CEL expressions
- **Prometheus metrics** -- push audit results to Pushgateway for dashboards and alerting
- **Watch mode** -- re-run checks on interval during development

## What Scry Checks

| Category        | Checks      | Highlights                                                                 |
| --------------- | ----------- | -------------------------------------------------------------------------- |
| SEO             | 18          | title, meta description, canonical, Open Graph, viewport, duplicate titles |
| Performance     | 14          | HTML size, compression, render-blocking, DOM size, cache headers, HTTP/2   |
| Security        | 13          | HSTS, CSP, cookies, CORS, SRI, security.txt                                |
| Accessibility   | 13          | form labels, ARIA, landmarks, heading hierarchy, keyboard, video captions  |
| Health          | 9           | 4xx/5xx, TTFB, redirect chains, mixed content, charset                     |
| Images          | 8           | alt text, broken src, large images, lazy loading, responsive, WebP/AVIF    |
| Links           | 5           | broken internal, orphan pages, deep pages, generic anchor text             |
| Structured Data | 5 + 10 deep | JSON-LD for 9 Schema.org types, date/URL validation, microdata detection   |
| TLS             | 5           | weak protocol, certificate expiry, self-signed, hostname mismatch          |
| Hreflang        | 4           | language codes, x-default, return links, self-reference                    |
| External Links  | 3           | broken outbound, redirects, timeouts                                       |

> See the full [Checks Catalog](docs/checks-catalog.md) for every check with severity levels.

## Commands

**`scry crawl <url>`** -- Crawl and audit an entire site.

```bash
scry crawl https://example.com --output json,html --output-file report
```

**`scry check <url>`** -- Audit a single page.

```bash
scry check https://example.com/blog/post --filter-category seo,performance
```

**`scry lighthouse <url>`** -- Run Lighthouse analysis.

```bash
scry lighthouse https://example.com --psi-key $PSI_API_KEY
```

**`scry validate`** -- Validate your configuration file.

```bash
scry validate
```

## Output Formats

| Format   | Flag                | Use case                                        |
| -------- | ------------------- | ----------------------------------------------- |
| Terminal | `--output terminal` | Human-readable, colorized tables                |
| JSON     | `--output json`     | Machine-readable, API integrations              |
| CSV      | `--output csv`      | Spreadsheets, data analysis                     |
| Markdown | `--output markdown` | Documentation, wikis                            |
| HTML     | `--output html`     | Self-contained visual report                    |
| SARIF    | `--output sarif`    | GitHub/GitLab PR annotations                    |
| JUnit    | `--output junit`    | CI test result integration                      |
| JSONL    | `--output jsonl`    | Streaming, log pipelines                        |
| PDF      | `--output pdf`      | Client deliverables (requires headless browser) |

Combine formats in a single run:

```bash
scry crawl https://example.com --output terminal,json,html --output-file report
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Audit website
  run: |
    go install github.com/meysam81/scry@latest
    scry crawl ${{ vars.SITE_URL }} \
      --output sarif,terminal \
      --output-file results \
      --fail-on critical

- name: Upload SARIF
  if: always()
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: results.sarif
```

### GitLab CI

```yaml
website-audit:
  image: golang:latest
  script:
    - go install github.com/meysam81/scry@latest
    - scry crawl $AUDIT_URL --output terminal,json --output-file audit-report --fail-on critical
  artifacts:
    paths:
      - audit-report.*
    when: always
```

### Baseline Comparison

Track regressions across deploys:

```bash
# Save a baseline after deploy
scry crawl https://example.com --save-baseline baseline.json

# Compare on next run
scry crawl https://example.com --compare-baseline baseline.json
```

New issues are flagged, resolved issues are reported, and existing issues stay quiet.

## Advanced Features

### Custom Rules (CEL)

Write audit rules in YAML using Common Expression Language:

```yaml
# rules/my-rules.yml
rules:
  - name: "custom/missing-csp"
    severity: warning
    condition: |
      page.status_code == 200 &&
      !('content-security-policy' in page.headers)
    message: "Missing Content-Security-Policy header"
```

```bash
scry crawl https://example.com --rules rules/my-rules.yml
```

### Parallel Domain Crawling

Audit multiple domains at once:

```bash
scry crawl --urls-file domains.txt --parallel-domains 4
```

### Checkpoint & Resume

Interrupt large crawls and pick up where you left off:

```bash
scry crawl https://large-site.com --checkpoint crawl.json
# Ctrl+C, then later:
scry crawl https://large-site.com --resume crawl.json
```

### Incremental Crawling

Only re-crawl pages that changed since the last run:

```bash
scry crawl https://example.com --incremental cache.json
```

### Prometheus Metrics

Push audit results to a Prometheus Pushgateway:

```bash
scry crawl https://example.com --metrics-push http://pushgateway:9091
```

### Watch Mode

Re-run a single-page audit on interval during development:

```bash
scry check https://localhost:3000 --watch --filter-category seo
```

## Configuration

Scry reads configuration from three sources (highest precedence first):

1. **CLI flags**
2. **Environment variables** (`SCRY_*`)
3. **`scry.yml` config file**

Minimal example:

```yaml
# scry.yml
crawl:
  max_depth: 3
  max_pages: 200
  concurrency: 8
  exclude:
    - "/admin/*"
    - "/api/*"

output:
  formats:
    - terminal
    - json
  file: report
  fail_on: critical
```

Validate your config without crawling:

```bash
scry validate
```

> See the full [Configuration Reference](docs/configuration.md) for all options.

## Contributing

Contributions are welcome!

```bash
git clone https://github.com/meysam81/scry.git
cd scry
go build
go test -race ./...
```

Please open an issue before submitting large changes.

## License

[Apache 2.0](LICENSE)
