# scry — Chrome extension

Client-side audit engine for the page you're on. The heavy lifting (94 checks
across SEO, performance, security, accessibility, images, structured data,
hreflang, TLS, and more) runs in a Go WebAssembly module compiled from the
same `core/` packages the CLI uses — one source of truth, two frontends.

## Architecture at a glance

```
┌── Chrome service worker (background/index.ts) ─────────────────────┐
│   • Instantiates scry.wasm ONCE                                    │
│   • Listens on chrome.webRequest.onResponseStarted per tab         │
│   • Routes ui:request-audit messages through the WASM engine       │
└───────────────────────────────────────────────────────────────────┘
                         ▲                ▲
            ui:request-audit        content:snapshot
                         │                │
┌── Side panel (sidepanel/App.vue) ──┐  ┌── Content script ──────────┐
│  Vue 3 + Pinia + Zod-validated     │  │  DOM snapshot, tech probe, │
│  state, design tokens, no inline   │  │  OG/twitter/JSON-LD scrape │
│  colours, styles, or spacings.     │  └───────────────────────────┘
└────────────────────────────────────┘
```

## Local dev

```bash
# one-shot production build (WASM + Vite)
make extension

# iterative development
make extension-dev
```

Then load-unpacked `extension/dist/` in `chrome://extensions`.

## Stack

- Vue 3 + Pinia + TypeScript (strict)
- Vite 8 + @crxjs/vite-plugin 2.x
- Zod 4 at every WASM ↔ JS boundary
- Design tokens in `src/styles/tokens.css` — **never** hard-code a colour,
  radius, font, or spacing value outside that file
- Manifest defined in `manifest.config.ts` (typed, derived from `package.json`)
