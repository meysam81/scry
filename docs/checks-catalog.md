# Scry Checks Catalog

Complete reference of all audit checks performed by scry, grouped by category.

Severity levels: **Critical** (must fix), **Warning** (should investigate), **Info** (best-practice suggestion).

---

## SEO

| Check                          | Severity | Description                                                                             |
| ------------------------------ | -------- | --------------------------------------------------------------------------------------- |
| `seo/missing-title`            | Critical | Page is missing a `<title>` tag                                                         |
| `seo/title-length`             | Warning  | Title length is outside the recommended 30-60 character range                           |
| `seo/missing-meta-description` | Warning  | Page is missing a `<meta name="description">` tag                                       |
| `seo/meta-description-length`  | Info     | Meta description length is outside the recommended 70-155 character range               |
| `seo/missing-h1`               | Warning  | Page is missing an `<h1>` tag                                                           |
| `seo/multiple-h1`              | Warning  | Page has more than one `<h1>` tag                                                       |
| `seo/missing-canonical`        | Warning  | Page is missing a `<link rel="canonical">` tag                                          |
| `seo/multiple-canonical`       | Warning  | Page has more than one canonical link                                                   |
| `seo/missing-lang`             | Warning  | Page is missing a `lang` attribute on the `<html>` tag                                  |
| `seo/missing-og-title`         | Info     | Page is missing an `og:title` meta tag                                                  |
| `seo/missing-og-description`   | Info     | Page is missing an `og:description` meta tag                                            |
| `seo/missing-og-image`         | Info     | Page is missing an `og:image` meta tag                                                  |
| `seo/missing-twitter-card`     | Info     | Page is missing a `twitter:card` meta tag                                               |
| `seo/missing-viewport`         | Critical | Page is missing a viewport meta tag                                                     |
| `seo/meta-robots-conflict`     | Warning  | Page has both a `noindex` robots directive and a canonical link                         |
| `seo/url-issues`               | Info     | URL has issues such as excessive length (>100 chars), underscores, or uppercase letters |
| `seo/noindex-in-sitemap`       | Critical | Page has `noindex` but is included in the sitemap (site-wide check)                     |
| `seo/duplicate-titles`         | Warning  | Multiple pages share the same `<title>` text (site-wide check)                          |

## Performance

| Check                                | Severity | Description                                                                             |
| ------------------------------------ | -------- | --------------------------------------------------------------------------------------- |
| `performance/large-html`             | Warning  | HTML response exceeds 100 KB                                                            |
| `performance/no-compression`         | Warning  | HTML response is not compressed (missing gzip or br `Content-Encoding`)                 |
| `performance/render-blocking-script` | Warning  | Script in `<head>` lacks `async` or `defer` attributes                                  |
| `performance/excessive-css`          | Info     | Page has more than 3 stylesheets in `<head>`                                            |
| `performance/render-blocking-css`    | Warning  | Stylesheet in `<head>` without a non-blocking `media` attribute                         |
| `performance/missing-resource-hints` | Info     | Page has no `<link rel="preconnect">` or `<link rel="dns-prefetch">` hints              |
| `performance/font-loading`           | Info     | Inline `@font-face` is missing `font-display` or uses `font-display: block`             |
| `performance/excessive-third-party`  | Warning  | Page loads scripts from more than 5 external origins                                    |
| `performance/excessive-dom-size`     | Warning  | DOM has more than 1500 element nodes                                                    |
| `performance/inline-bloat`           | Info     | Total inline `<script>` and `<style>` content exceeds 50 KB                             |
| `performance/missing-cache-headers`  | Warning  | Response has no `Cache-Control` header                                                  |
| `performance/excessive-webfonts`     | Warning  | Page has more than 4 `@font-face` declarations in inline styles                         |
| `performance/unminified-resources`   | Info     | Inline `<script>` appears to contain unminified code (comments or excessive whitespace) |
| `performance/no-http2`               | Info     | No `Alt-Svc` header indicating HTTP/2 or HTTP/3 support detected                        |

## Health

| Check                                 | Severity | Description                                                                   |
| ------------------------------------- | -------- | ----------------------------------------------------------------------------- |
| `health/4xx`                          | Critical | Page returned an HTTP 4xx client error status code                            |
| `health/5xx`                          | Critical | Page returned an HTTP 5xx server error status code                            |
| `health/redirect-chain`               | Warning  | Redirect chain has more than 2 hops                                           |
| `health/redirect-loop`                | Critical | Page URL appears in its own redirect chain                                    |
| `health/slow-ttfb`                    | Warning  | Time to first byte exceeds 2 seconds                                          |
| `health/mixed-content`                | Warning  | HTTPS page loads HTTP assets                                                  |
| `health/server-version-leak`          | Info     | `Server` header reveals a version number                                      |
| `health/https-redirect-not-permanent` | Warning  | HTTP to HTTPS redirect uses non-permanent status (302/307); prefer 301 or 308 |
| `health/missing-charset`              | Warning  | `Content-Type` header for `text/html` is missing `charset`                    |

## Security

| Check                                        | Severity | Description                                                             |
| -------------------------------------------- | -------- | ----------------------------------------------------------------------- |
| `security/missing-strict-transport-security` | Warning  | HTTPS page is missing the `Strict-Transport-Security` header            |
| `security/weak-hsts`                         | Info     | HSTS `max-age` is less than 31536000 seconds (1 year)                   |
| `security/missing-content-security-policy`   | Warning  | Page is missing the `Content-Security-Policy` header                    |
| `security/csp-unsafe`                        | Warning  | CSP contains `'unsafe-inline'` or `'unsafe-eval'` directives            |
| `security/missing-x-content-type-options`    | Warning  | Page is missing `X-Content-Type-Options: nosniff`                       |
| `security/missing-x-frame-options`           | Info     | Page is missing `X-Frame-Options` (DENY or SAMEORIGIN)                  |
| `security/missing-referrer-policy`           | Info     | Page is missing the `Referrer-Policy` header                            |
| `security/insecure-referrer-policy`          | Warning  | `Referrer-Policy` uses `unsafe-url` or `no-referrer-when-downgrade`     |
| `security/missing-permissions-policy`        | Info     | Page is missing the `Permissions-Policy` header                         |
| `security/insecure-cookies`                  | Warning  | `Set-Cookie` is missing `HttpOnly`, `Secure`, or `SameSite` flags       |
| `security/cors-wildcard`                     | Warning  | `Access-Control-Allow-Origin` is set to wildcard (`*`)                  |
| `security/missing-sri`                       | Info     | External script or stylesheet is missing a `integrity` (SRI) attribute  |
| `security/missing-security-txt`              | Info     | Site does not have a `/.well-known/security.txt` file (site-wide check) |

## Accessibility

| Check                                     | Severity | Description                                                                           |
| ----------------------------------------- | -------- | ------------------------------------------------------------------------------------- |
| `accessibility/missing-form-label`        | Warning  | `<input>` element has no associated label, `aria-label`, or `aria-labelledby`         |
| `accessibility/empty-link`                | Warning  | Anchor element has no accessible text content                                         |
| `accessibility/missing-skip-nav`          | Info     | Page is missing a skip navigation link                                                |
| `accessibility/heading-hierarchy`         | Warning  | Heading levels skip (e.g. `h1` directly to `h3`)                                      |
| `accessibility/missing-button-text`       | Warning  | `<button>` element has no accessible text                                             |
| `accessibility/missing-table-header`      | Info     | `<table>` element has no `<th>` header cells                                          |
| `accessibility/missing-img-alt-in-figure` | Warning  | Image inside `<figure>` has no alt text and no `<figcaption>`                         |
| `accessibility/positive-tabindex`         | Warning  | Element has `tabindex > 0`, disrupting natural tab order                              |
| `accessibility/missing-landmarks`         | Warning  | Page has no ARIA landmark regions (`<main>`, `<nav>`, roles, etc.)                    |
| `accessibility/invalid-aria`              | Warning  | Element has `aria-hidden="true"` with a non-negative `tabindex`                       |
| `accessibility/onclick-without-keyboard`  | Warning  | Non-interactive element has `onclick` but no `tabindex` or `role`                     |
| `accessibility/missing-video-captions`    | Warning  | `<video>` element is missing a `<track>` with `kind="captions"` or `kind="subtitles"` |
| `accessibility/missing-autocomplete`      | Info     | Personal-data input (email, tel, password, etc.) is missing `autocomplete` attribute  |

## Images

| Check                         | Severity | Description                                                             |
| ----------------------------- | -------- | ----------------------------------------------------------------------- |
| `images/missing-alt`          | Warning  | Image is missing the `alt` attribute                                    |
| `images/empty-alt-in-link`    | Warning  | Image inside a link has an empty `alt` attribute                        |
| `images/broken-src`           | Critical | Image `src` returned an HTTP 4xx/5xx error                              |
| `images/large-image`          | Warning  | Image exceeds 500 KB                                                    |
| `images/legacy-format`        | Info     | Image uses a legacy format (jpg, gif, bmp, tiff); consider WebP or AVIF |
| `images/missing-lazy-loading` | Info     | Below-the-fold image is missing `loading="lazy"`                        |
| `images/missing-dimensions`   | Warning  | Image is missing both `width` and `height` attributes                   |
| `images/missing-responsive`   | Info     | Image is missing `srcset` and `sizes` attributes                        |

## Links

| Check                       | Severity | Description                                                           |
| --------------------------- | -------- | --------------------------------------------------------------------- |
| `links/excessive-links`     | Info     | Page has more than 100 links                                          |
| `links/generic-anchor-text` | Info     | Link uses generic anchor text (e.g. "click here", "read more")        |
| `links/broken-internal`     | Critical | Internal link target returned an HTTP 4xx/5xx error (site-wide check) |
| `links/orphan-page`         | Warning  | Page has no internal links pointing to it (site-wide check)           |
| `links/deep-page`           | Info     | Page is at crawl depth greater than 4 (site-wide check)               |

## Structured Data

| Check                                        | Severity | Description                                                                   |
| -------------------------------------------- | -------- | ----------------------------------------------------------------------------- |
| `structured-data/missing-json-ld`            | Info     | Page has no JSON-LD structured data                                           |
| `structured-data/malformed-json-ld`          | Warning  | JSON-LD block contains invalid JSON                                           |
| `structured-data/microdata-detected`         | Info     | HTML contains microdata attributes; consider migrating to JSON-LD             |
| `structured-data/missing-context`            | Info     | JSON-LD block has no @context                                                 |
| `structured-data/wrong-context`              | Warning  | @context doesn't point to schema.org                                          |
| `structured-data/missing-type`               | Warning  | JSON-LD block has no `@type` field                                            |
| `structured-data/unknown-type`               | Info     | JSON-LD `@type` is not a commonly recognised Schema.org type                  |
| `structured-data/missing-required-field`     | Warning  | Schema.org type is missing required fields                                    |
| `structured-data/invalid-date-format`        | Warning  | JSON-LD date field has invalid or non-ISO 8601 value                          |
| `structured-data/invalid-url-field`          | Warning  | JSON-LD URL field has invalid value (not http/https or relative path)         |
| `structured-data/invalid-nested-type`        | Warning  | Nested object's @type doesn't match expected types for the property           |
| `structured-data/invalid-enum-value`         | Warning  | Property value is not in the allowed enum set                                 |
| `structured-data/breadcrumb-positions`       | Warning  | BreadcrumbList items have non-sequential positions                            |
| `structured-data/duplicate-type`             | Info     | Multiple JSON-LD blocks on the same page declare the same @type               |
| `structured-data/google-missing-required`    | Warning  | Google Rich Results required field is missing                                 |
| `structured-data/google-missing-recommended` | Info     | Google Rich Results recommended field is missing                              |
| `structured-data/not-google-eligible`        | Info     | Schema.org type is not eligible for Google Rich Results                       |
| `structured-data/search-action-template`     | Warning  | WebSite SearchAction target is missing {search_term_string} template variable |

## External Links

| Check                     | Severity | Description                                            |
| ------------------------- | -------- | ------------------------------------------------------ |
| `external-links/timeout`  | Info     | External link check timed out                          |
| `external-links/broken`   | Warning  | External link returned an error or HTTP 4xx/5xx status |
| `external-links/redirect` | Info     | External link redirects (HTTP 3xx)                     |

## TLS

| Check                           | Severity | Description                                 |
| ------------------------------- | -------- | ------------------------------------------- |
| `tls/weak-protocol`             | Warning  | Server negotiated a TLS version below 1.2   |
| `tls/certificate-expired`       | Critical | TLS certificate has expired                 |
| `tls/certificate-expiring-soon` | Warning  | TLS certificate expires within 30 days      |
| `tls/self-signed`               | Warning  | TLS certificate is self-signed              |
| `tls/hostname-mismatch`         | Critical | TLS certificate does not match the hostname |

## Hreflang

| Check                             | Severity | Description                                                                 |
| --------------------------------- | -------- | --------------------------------------------------------------------------- |
| `hreflang/invalid-language-code`  | Warning  | Hreflang value is not a valid BCP 47 language tag                           |
| `hreflang/missing-x-default`      | Info     | Page has hreflang annotations but no `x-default`                            |
| `hreflang/missing-return-link`    | Warning  | Hreflang target page does not link back to the source (site-wide check)     |
| `hreflang/self-reference-missing` | Info     | Page has hreflang annotations but does not include itself (site-wide check) |

---

**Total: 110 checks** across 11 categories.
