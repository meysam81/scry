import { defineManifest } from "@crxjs/vite-plugin";
import pkg from "./package.json" with { type: "json" };

// Keeping the manifest in TS means every field is typed against the
// @types/chrome manifest definition, so typos surface at build time.
export default defineManifest({
  manifest_version: 3,
  name: "Scry",
  short_name: "Scry",
  description:
    "Audit the page you are on: SEO, performance, security, a11y, schema.",
  version: pkg.version,

  action: {
    default_title: "Scry",
    default_icon: {
      "16": "icons/icon-16.png",
      "32": "icons/icon-32.png",
      "48": "icons/icon-48.png",
      "128": "icons/icon-128.png",
    },
  },

  icons: {
    "16": "icons/icon-16.png",
    "32": "icons/icon-32.png",
    "48": "icons/icon-48.png",
    "128": "icons/icon-128.png",
  },

  side_panel: {
    default_path: "src/entrypoints/sidepanel/index.html",
  },

  background: {
    service_worker: "src/entrypoints/background/index.ts",
    type: "module",
  },

  content_scripts: [
    {
      matches: ["<all_urls>"],
      js: ["src/entrypoints/content/index.ts"],
      run_at: "document_idle",
      all_frames: false,
    },
  ],

  permissions: [
    "activeTab",
    "tabs",
    "storage",
    "sidePanel",
    "scripting",
    "webRequest",
  ],

  host_permissions: ["<all_urls>"],

  web_accessible_resources: [
    {
      resources: ["scry.wasm", "wasm_exec.js"],
      matches: ["<all_urls>"],
    },
  ],

  content_security_policy: {
    // Allow WebAssembly.instantiate in the service worker and pages.
    extension_pages: "script-src 'self' 'wasm-unsafe-eval'; object-src 'self'",
  },
});
