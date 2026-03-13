# Scry Configuration Reference

Scry supports three layers of configuration with the following precedence (highest to lowest):

1. **CLI flags** -- override everything
2. **Environment variables** -- override YAML and defaults
3. **YAML config file** (`scry.yml`) -- override defaults only
4. **Built-in defaults**

---

## Environment Variables

| Variable                | Default                 | Description                                                                                                               |
| ----------------------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| `SCRY_MAX_DEPTH`        | `5`                     | Maximum crawl depth from the seed URL                                                                                     |
| `SCRY_MAX_PAGES`        | `500`                   | Maximum number of pages to crawl                                                                                          |
| `SCRY_CONCURRENCY`      | `10`                    | Number of parallel fetchers                                                                                               |
| `SCRY_REQUEST_TIMEOUT`  | `10s`                   | Per-request timeout (Go duration format, e.g. `10s`, `1m`)                                                                |
| `SCRY_RATE_LIMIT`       | `50`                    | Maximum requests per second                                                                                               |
| `SCRY_USER_AGENT`       | `scry/1.0`              | HTTP User-Agent string sent with requests                                                                                 |
| `SCRY_RESPECT_ROBOTS`   | `true`                  | Whether to respect `robots.txt` directives                                                                                |
| `SCRY_OUTPUT`           | `terminal`              | Output format(s), comma-separated. Valid: `terminal`, `json`, `csv`, `markdown`, `html`, `sarif`, `junit`, `jsonl`, `pdf` |
| `SCRY_OUTPUT_FILE`      | _(empty)_               | File path for non-terminal output                                                                                         |
| `SCRY_FAIL_ON`          | _(empty)_               | Severity threshold for non-zero exit code. Valid: `critical`, `warning`, `any`                                            |
| `SCRY_BROWSER_MODE`     | `false`                 | Enable headless browser rendering via rod                                                                                 |
| `SCRY_BROWSERLESS_URL`  | `http://localhost:3000` | Browserless endpoint URL for headless rendering                                                                           |
| `SCRY_LIGHTHOUSE`       | `false`                 | Enable Lighthouse performance scoring                                                                                     |
| `SCRY_LIGHTHOUSE_MODE`  | `psi`                   | Lighthouse mode. Valid: `psi`, `browserless`                                                                              |
| `SCRY_PSI_API_KEY`      | _(empty)_               | PageSpeed Insights API key                                                                                                |
| `SCRY_PSI_STRATEGY`     | `mobile`                | PSI strategy. Valid: `mobile`, `desktop`                                                                                  |
| `SCRY_LOG_LEVEL`        | `info`                  | Log verbosity. Valid: `debug`, `info`, `warn`, `error`                                                                    |
| `SCRY_LOG_FORMAT`       | `pretty`                | Log format. Valid: `pretty`, `json`                                                                                       |
| `SCRY_FILTER_SEVERITY`  | _(empty)_               | Comma-separated severity filter (e.g. `critical,warning`)                                                                 |
| `SCRY_FILTER_CATEGORY`  | _(empty)_               | Comma-separated category filter (e.g. `seo,performance`)                                                                  |
| `SCRY_CHECKPOINT_FILE`  | _(empty)_               | Save crawl checkpoint to this file for later resume                                                                       |
| `SCRY_RESUME_FILE`      | _(empty)_               | Resume crawl from a previous checkpoint file                                                                              |
| `SCRY_INCREMENTAL_FILE` | _(empty)_               | Incremental crawl cache file for delta-only re-crawls                                                                     |
| `SCRY_PARALLEL_DOMAINS` | `3`                     | Number of domains to crawl in parallel                                                                                    |
| `SCRY_METRICS_PUSH_URL` | _(empty)_               | Prometheus Pushgateway URL for audit metrics                                                                              |
| `SCRY_RULES_FILE`       | _(empty)_               | Path to CEL custom rules YAML file                                                                                        |
| `SCRY_SAVE_BASELINE`    | _(empty)_               | Save issues to baseline file for future comparison                                                                        |
| `SCRY_COMPARE_BASELINE` | _(empty)_               | Compare current issues against a saved baseline                                                                           |

---

## CLI Flags

### Global Flags (available on all commands)

| Flag            | Alias | Default    | Description                                                         |
| --------------- | ----- | ---------- | ------------------------------------------------------------------- |
| `--log-level`   | `-l`  | `info`     | Log level: `debug`, `info`, `warn`, `error`                         |
| `--log-format`  |       | `pretty`   | Log format: `pretty`, `json`                                        |
| `--output`      | `-o`  | `terminal` | Output format(s), comma-separated                                   |
| `--output-file` |       | _(empty)_  | File path for non-terminal output                                   |
| `--fail-on`     |       | _(empty)_  | Severity threshold for non-zero exit (`critical`, `warning`, `any`) |
| `--config`      |       | _(empty)_  | Path to `scry.yml` config file                                      |

### `check` Command Flags

Run audit checks on a single URL.

```
scry check [flags] <url>
```

| Flag                | Default                 | Description                             |
| ------------------- | ----------------------- | --------------------------------------- |
| `--browser`         | `false`                 | Enable headless browser rendering       |
| `--browserless-url` | `http://localhost:3000` | Browserless endpoint URL                |
| `--lighthouse`      | `false`                 | Enable Lighthouse scoring               |
| `--lighthouse-mode` | `psi`                   | Lighthouse mode: `psi` or `browserless` |
| `--psi-key`         | _(empty)_               | PageSpeed Insights API key              |
| `--psi-strategy`    | `mobile`                | PSI strategy: `mobile` or `desktop`     |
| `--timeout`         | `10s`                   | Per-request timeout                     |
| `--user-agent`      | `scry/1.0`              | HTTP User-Agent string                  |
| `--filter-severity` | _(empty)_               | Comma-separated severity filter         |
| `--filter-category` | _(empty)_               | Comma-separated category filter         |
| `--watch`           | `false`                 | Re-run check on interval                |
| `--watch-interval`  | `30s`                   | Watch re-run interval                   |
| `--rules`           | _(empty)_               | Path to CEL custom rules YAML file      |

### `crawl` Command Flags

Crawl a website and run audit checks.

```
scry crawl [flags] <url>
```

| Flag                 | Alias | Default                 | Description                             |
| -------------------- | ----- | ----------------------- | --------------------------------------- |
| `--depth`            | `-d`  | `5`                     | Maximum crawl depth                     |
| `--max-pages`        |       | `500`                   | Page cap                                |
| `--concurrency`      | `-c`  | `10`                    | Parallel fetchers                       |
| `--browser`          |       | `false`                 | Enable headless browser rendering       |
| `--browserless-url`  |       | `http://localhost:3000` | Browserless endpoint URL                |
| `--lighthouse`       |       | `false`                 | Enable Lighthouse scoring               |
| `--lighthouse-mode`  |       | `psi`                   | Lighthouse mode: `psi` or `browserless` |
| `--psi-key`          |       | _(empty)_               | PageSpeed Insights API key              |
| `--psi-strategy`     |       | `mobile`                | PSI strategy: `mobile` or `desktop`     |
| `--ignore-robots`    |       | `false`                 | Bypass `robots.txt` directives          |
| `--include`          |       | _(empty)_               | Glob patterns for URL inclusion         |
| `--exclude`          |       | _(empty)_               | Glob patterns for URL exclusion         |
| `--rate-limit`       |       | `50`                    | Requests per second                     |
| `--timeout`          |       | `10s`                   | Per-request timeout                     |
| `--user-agent`       |       | `scry/1.0`              | HTTP User-Agent string                  |
| `--filter-severity`  |       | _(empty)_               | Comma-separated severity filter         |
| `--filter-category`  |       | _(empty)_               | Comma-separated category filter         |
| `--checkpoint`       |       | _(empty)_               | Save crawl checkpoint to file           |
| `--resume`           |       | _(empty)_               | Resume crawl from checkpoint file       |
| `--urls-file`        |       | _(empty)_               | File with URLs for multi-domain crawl   |
| `--parallel-domains` |       | `3`                     | Domains to crawl in parallel            |
| `--metrics-push`     |       | _(empty)_               | Prometheus Pushgateway URL              |
| `--incremental`      |       | _(empty)_               | Incremental crawl cache file            |
| `--rules`            |       | _(empty)_               | Path to CEL custom rules YAML file      |
| `--save-baseline`    |       | _(empty)_               | Save issues to baseline file            |
| `--compare-baseline` |       | _(empty)_               | Compare issues against saved baseline   |

### `lighthouse` Command Flags

Run Lighthouse audits on one or more URLs.

```
scry lighthouse [flags] <url> [url2 ...]
```

| Flag          | Default   | Description                             |
| ------------- | --------- | --------------------------------------- |
| `--mode`      | `psi`     | Lighthouse mode: `psi` or `browserless` |
| `--psi-key`   | _(empty)_ | PageSpeed Insights API key              |
| `--strategy`  | `mobile`  | PSI strategy: `mobile` or `desktop`     |
| `--urls-file` | _(empty)_ | Path to file with URLs (one per line)   |

### `validate` Command

Validate configuration and show resolved values. Takes no additional flags.

```
scry validate
```

---

## Config File (`scry.yml`)

Scry searches for `scry.yml` in the following order:

1. Path specified via `--config` flag
2. Current working directory
3. User home directory (`$HOME`)

YAML values are only applied when the corresponding environment variable is **not** set.

### Example `scry.yml`

```yaml
crawl:
  max_depth: 3
  max_pages: 200
  concurrency: 5
  respect_robots: true
  rate_limit: 20
  timeout: "15s"
  user_agent: "my-bot/1.0"
  include:
    - "/blog/*"
    - "/docs/*"
  exclude:
    - "/admin/*"
    - "/api/*"

output:
  formats:
    - terminal
    - json
  file: "report.json"
  fail_on: "critical"

filter:
  severity: "critical,warning"
  category: "seo,performance"

lighthouse:
  enabled: false
  mode: "psi"
  strategy: "mobile"

browser:
  enabled: false
  browserless_url: "http://localhost:3000"

rules:
  file: "rules/my-rules.yml"

baseline:
  save: "baseline.json"
  compare: "baseline.json"
```

### YAML Field Reference

| Section      | Field             | Type   | Description                       |
| ------------ | ----------------- | ------ | --------------------------------- |
| `crawl`      | `max_depth`       | int    | Maximum crawl depth               |
| `crawl`      | `max_pages`       | int    | Page cap                          |
| `crawl`      | `concurrency`     | int    | Parallel fetchers                 |
| `crawl`      | `respect_robots`  | bool   | Respect `robots.txt`              |
| `crawl`      | `rate_limit`      | int    | Requests per second               |
| `crawl`      | `timeout`         | string | Per-request timeout (Go duration) |
| `crawl`      | `user_agent`      | string | HTTP User-Agent                   |
| `crawl`      | `include`         | list   | URL inclusion glob patterns       |
| `crawl`      | `exclude`         | list   | URL exclusion glob patterns       |
| `output`     | `formats`         | list   | Output formats                    |
| `output`     | `file`            | string | Output file path                  |
| `output`     | `fail_on`         | string | Exit-code severity threshold      |
| `filter`     | `severity`        | string | Severity filter                   |
| `filter`     | `category`        | string | Category filter                   |
| `lighthouse` | `enabled`         | bool   | Enable Lighthouse                 |
| `lighthouse` | `mode`            | string | Lighthouse mode                   |
| `lighthouse` | `strategy`        | string | PSI strategy                      |
| `browser`    | `enabled`         | bool   | Enable headless browser           |
| `browser`    | `browserless_url` | string | Browserless endpoint URL          |
| `rules`      | `file`            | string | Path to CEL custom rules YAML     |
| `baseline`   | `save`            | string | Save baseline to this file        |
| `baseline`   | `compare`         | string | Compare against this baseline     |
