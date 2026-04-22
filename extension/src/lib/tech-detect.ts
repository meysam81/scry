// Lightweight, zero-dependency tech-stack detection. Runs inside the content
// script (where it has access to DOM + response headers relayed from the
// background) and classifies the page by framework, analytics, CMS, CDN, etc.
//
// This is a "best effort" v1 — not as exhaustive as Wappalyzer. Add patterns
// by editing the SIGNATURES array. Every match must come with an icon hint so
// the UI can render it consistently.

export interface TechSignature {
  name: string;
  category: TechCategory;
  /** If any of these DOM selectors match, the signature fires. */
  dom?: string[];
  /** If the HTML source contains any of these strings, the signature fires. */
  html?: RegExp[];
  /** If any header value contains any of these strings, the signature fires. */
  headers?: Array<{ name: string; match: RegExp }>;
  /** If any script src contains this, the signature fires. */
  script?: RegExp[];
  /** If any meta tag with this name/property exists, the signature fires. */
  meta?: Array<{ name?: string; property?: string; match?: RegExp }>;
}

export type TechCategory =
  | "framework"
  | "cms"
  | "analytics"
  | "tag-manager"
  | "cdn"
  | "server"
  | "ecommerce"
  | "ui-library"
  | "advertising"
  | "font"
  | "search";

export interface DetectedTech {
  name: string;
  category: TechCategory;
}

export const SIGNATURES: TechSignature[] = [
  // --- Frameworks ---
  {
    name: "React",
    category: "framework",
    dom: ["[data-reactroot]"],
    html: [/<[^>]+data-reactroot/i, /\b__REACT_DEVTOOLS_GLOBAL_HOOK__\b/],
  },
  {
    name: "Next.js",
    category: "framework",
    dom: ["#__next"],
    script: [/_next\/static\//],
  },
  {
    name: "Vue",
    category: "framework",
    html: [/\b__VUE__\b/, /v-(?:if|for|bind|on|model)=/],
    dom: ["[data-v-app]"],
  },
  {
    name: "Nuxt",
    category: "framework",
    dom: ["#__nuxt"],
    script: [/\/_nuxt\//],
  },
  {
    name: "Svelte",
    category: "framework",
    html: [/\bclass=\"svelte-[a-z0-9]+/i],
  },
  { name: "Angular", category: "framework", dom: ["[ng-version]", "[ng-app]"] },
  {
    name: "Astro",
    category: "framework",
    html: [/\bdata-astro-cid-\b/, /_astro\//],
  },
  {
    name: "Remix",
    category: "framework",
    script: [/\/build\/_assets\//],
    html: [/__remixContext/],
  },
  { name: "SvelteKit", category: "framework", script: [/\/_app\/immutable\//] },

  // --- CMS ---
  {
    name: "WordPress",
    category: "cms",
    html: [/\/wp-content\//, /\/wp-includes\//],
    meta: [{ name: "generator", match: /wordpress/i }],
  },
  { name: "Shopify", category: "cms", html: [/cdn\.shopify\.com/] },
  {
    name: "Webflow",
    category: "cms",
    html: [/webflow\.(?:com|io)/],
    meta: [{ name: "generator", match: /webflow/i }],
  },
  {
    name: "Wix",
    category: "cms",
    html: [/static\.parastorage\.com/, /wix\.com/],
  },
  {
    name: "Squarespace",
    category: "cms",
    html: [/static1\.squarespace\.com/, /squarespace\.com/],
  },
  {
    name: "Ghost",
    category: "cms",
    meta: [{ name: "generator", match: /ghost/i }],
  },
  { name: "Sanity", category: "cms", html: [/cdn\.sanity\.io/] },
  { name: "Contentful", category: "cms", html: [/images\.ctfassets\.net/] },
  {
    name: "Hugo",
    category: "cms",
    meta: [{ name: "generator", match: /hugo/i }],
  },
  {
    name: "Jekyll",
    category: "cms",
    meta: [{ name: "generator", match: /jekyll/i }],
  },

  // --- Analytics / tags ---
  {
    name: "Google Analytics",
    category: "analytics",
    script: [
      /google-analytics\.com\/(analytics|ga)\.js/,
      /googletagmanager\.com\/gtag\/js/,
    ],
  },
  {
    name: "Google Tag Manager",
    category: "tag-manager",
    script: [/googletagmanager\.com\/gtm\.js/],
  },
  { name: "Plausible", category: "analytics", script: [/plausible\.io\/js/] },
  { name: "Fathom", category: "analytics", script: [/cdn\.usefathom\.com/] },
  { name: "Mixpanel", category: "analytics", script: [/cdn\.mxpnl\.com/] },
  { name: "Segment", category: "analytics", script: [/cdn\.segment\.com/] },
  { name: "Hotjar", category: "analytics", script: [/static\.hotjar\.com/] },
  { name: "Amplitude", category: "analytics", script: [/cdn\.amplitude\.com/] },
  { name: "PostHog", category: "analytics", script: [/posthog\.com/] },
  {
    name: "Cloudflare Insights",
    category: "analytics",
    script: [/static\.cloudflareinsights\.com/],
  },

  // --- CDN / infra ---
  {
    name: "Cloudflare",
    category: "cdn",
    headers: [
      { name: "server", match: /cloudflare/i },
      { name: "cf-ray", match: /./ },
    ],
  },
  {
    name: "Vercel",
    category: "cdn",
    headers: [
      { name: "server", match: /vercel/i },
      { name: "x-vercel-id", match: /./ },
    ],
  },
  {
    name: "Netlify",
    category: "cdn",
    headers: [
      { name: "server", match: /netlify/i },
      { name: "x-nf-request-id", match: /./ },
    ],
  },
  {
    name: "AWS CloudFront",
    category: "cdn",
    headers: [{ name: "via", match: /cloudfront/i }],
  },
  {
    name: "Fastly",
    category: "cdn",
    headers: [
      { name: "x-served-by", match: /cache/i },
      { name: "via", match: /varnish/i },
    ],
  },

  // --- Servers ---
  {
    name: "nginx",
    category: "server",
    headers: [{ name: "server", match: /nginx/i }],
  },
  {
    name: "Apache",
    category: "server",
    headers: [{ name: "server", match: /apache/i }],
  },
  {
    name: "Caddy",
    category: "server",
    headers: [{ name: "server", match: /caddy/i }],
  },

  // --- Ecommerce ---
  { name: "Stripe", category: "ecommerce", script: [/js\.stripe\.com/] },
  {
    name: "WooCommerce",
    category: "ecommerce",
    html: [/\/wp-content\/plugins\/woocommerce\//],
  },

  // --- UI libs / fonts ---
  {
    name: "Tailwind CSS",
    category: "ui-library",
    html: [/\btw-|class=\"[^\"]*(?:bg|text|p|m|flex|grid)-\w+/i],
  },
  {
    name: "Bootstrap",
    category: "ui-library",
    html: [/\bclass=\"[^\"]*\b(?:container|row|col-\w+)\b/i],
  },
  { name: "Google Fonts", category: "font", html: [/fonts\.googleapis\.com/] },

  // --- Search ---
  {
    name: "Algolia",
    category: "search",
    script: [/cdn\.jsdelivr\.net\/npm\/algoliasearch/],
  },
];

export interface DetectInput {
  html: string;
  headers: Record<string, string[]>;
  scripts: string[];
  metaTags: Array<{ name?: string; property?: string; content?: string }>;
}

export function detectTech(input: DetectInput): DetectedTech[] {
  const hits = new Map<string, DetectedTech>();

  for (const sig of SIGNATURES) {
    if (hits.has(sig.name)) continue;

    if (sig.html?.some((re) => re.test(input.html))) {
      hits.set(sig.name, { name: sig.name, category: sig.category });
      continue;
    }
    if (sig.script?.some((re) => input.scripts.some((s) => re.test(s)))) {
      hits.set(sig.name, { name: sig.name, category: sig.category });
      continue;
    }
    if (
      sig.meta?.some((m) =>
        input.metaTags.some((t) => {
          if (m.name && t.name?.toLowerCase() !== m.name.toLowerCase())
            return false;
          if (
            m.property &&
            t.property?.toLowerCase() !== m.property.toLowerCase()
          )
            return false;
          if (m.match && !(t.content && m.match.test(t.content))) return false;
          return true;
        }),
      )
    ) {
      hits.set(sig.name, { name: sig.name, category: sig.category });
      continue;
    }
    if (
      sig.headers?.some((h) => {
        const values = input.headers[h.name.toLowerCase()] ?? [];
        return values.some((v) => h.match.test(v));
      })
    ) {
      hits.set(sig.name, { name: sig.name, category: sig.category });
    }
  }

  return Array.from(hits.values()).sort(
    (a, b) =>
      a.category.localeCompare(b.category) || a.name.localeCompare(b.name),
  );
}
