# scry

A fast, developer-friendly website auditing tool.

## Features

- Concurrent BFS crawler with rate limiting and robots.txt support
- SEO checks (title, meta description, H1, canonical, lang, Open Graph)
- Health checks (4xx/5xx detection, TTFB, redirects, mixed content)
- Link analysis (broken internal links, orphan pages, deep pages)
- Image checks (alt text, file size, broken sources)
- Performance heuristics (HTML size, compression, render-blocking scripts)
- Structured data validation (JSON-LD)
- Security header checks (CSP, HSTS, X-Frame-Options, and more)
- Accessibility checks (form labels, heading hierarchy, skip nav, and more)
- External link validation
- TLS/SSL certificate checks
- Lighthouse integration (PageSpeed Insights + browserless)
- Multiple output formats: terminal, JSON, CSV, Markdown, HTML, SARIF, JUnit
- CEL-based custom rule engine
- Baseline comparison for CI/CD
- Site health scoring

## Quick start

```bash
go install github.com/meysam81/scry@latest
scry crawl https://example.com
```

## Usage

Crawl an entire site and export a JSON report:

```bash
scry crawl https://example.com --output json --output-file report
```

Audit a single page:

```bash
scry check https://example.com/page
```

Run Lighthouse via PageSpeed Insights:

```bash
scry lighthouse https://example.com --psi-key YOUR_KEY
```

Fail in CI when critical issues are found:

```bash
scry crawl https://example.com --fail-on critical
```

## Commands

| Command      | Description                                        |
| ------------ | -------------------------------------------------- |
| `crawl`      | BFS-crawl a site and run all enabled audit checks. |
| `check`      | Audit a single URL without crawling.               |
| `lighthouse` | Run Lighthouse analysis (PSI or browserless).      |

## Configuration

scry reads configuration from three sources in order of precedence
(highest first):

1. CLI flags
2. Environment variables
3. `scry.yml` config file

Example `scry.yml`:

```yaml
crawl:
  max_depth: 5
  max_pages: 200
  concurrency: 8
  respect_robots: true
  exclude:
    - "/admin/*"
    - "/api/*"

output:
  formats:
    - terminal
    - json
  file: ./audit-report
  fail_on: critical

lighthouse:
  enabled: false
  mode: psi
  strategy: mobile

browser:
  enabled: false
  browserless_url: http://localhost:3000
```

## Output formats

scry supports the following output formats, selectable via `--output`
(comma-separated for multiple):

- **terminal** -- human-readable table printed to stdout
- **json** -- machine-readable JSON
- **csv** -- comma-separated values
- **markdown** -- Markdown table
- **html** -- self-contained HTML report
- **sarif** -- SARIF v2.1.0 for GitHub/IDE integration
- **junit** -- JUnit XML for CI pipelines

## License

Apache 2.0. See [LICENSE](LICENSE) file.
