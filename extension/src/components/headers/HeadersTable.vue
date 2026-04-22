<script setup lang="ts">
/**
 * Renders the response headers captured by chrome.webRequest. Groups by
 * header family (security, cache, tracking, other) to make scanning easy.
 */
import { computed } from "vue";

const props = defineProps<{ headers: Record<string, string[]> }>();

const SECURITY = new Set([
  "content-security-policy",
  "strict-transport-security",
  "x-content-type-options",
  "x-frame-options",
  "referrer-policy",
  "permissions-policy",
  "cross-origin-embedder-policy",
  "cross-origin-opener-policy",
  "cross-origin-resource-policy",
]);

const CACHE = new Set([
  "cache-control",
  "expires",
  "etag",
  "last-modified",
  "age",
  "vary",
]);

type Group = "security" | "cache" | "content" | "other";

const CONTENT = new Set([
  "content-type",
  "content-encoding",
  "content-language",
  "content-length",
  "content-disposition",
]);

function groupOf(name: string): Group {
  const n = name.toLowerCase();
  if (SECURITY.has(n)) return "security";
  if (CACHE.has(n)) return "cache";
  if (CONTENT.has(n)) return "content";
  return "other";
}

const GROUP_ORDER: Group[] = ["security", "content", "cache", "other"];

const GROUP_META: Record<Group, { label: string; hint: string }> = {
  security: { label: "Security", hint: "Headers that protect users" },
  content: { label: "Content", hint: "Type, encoding, language" },
  cache: { label: "Cache", hint: "Caching directives" },
  other: { label: "Other", hint: "Everything else" },
};

const grouped = computed(() => {
  const out: Record<Group, Array<[string, string[]]>> = {
    security: [],
    content: [],
    cache: [],
    other: [],
  };
  for (const [name, values] of Object.entries(props.headers)) {
    out[groupOf(name)].push([name, values]);
  }
  for (const g of GROUP_ORDER) out[g].sort((a, b) => a[0].localeCompare(b[0]));
  return out;
});
</script>

<template>
  <div class="stack">
    <section
      v-for="g in GROUP_ORDER"
      :key="g"
      v-show="grouped[g].length"
      class="group"
    >
      <header class="g-head">
        <h4 class="g-title">{{ GROUP_META[g].label }}</h4>
        <span class="g-hint">{{ GROUP_META[g].hint }}</span>
      </header>
      <table class="tbl">
        <tbody>
          <tr v-for="[name, values] in grouped[g]" :key="name">
            <th>{{ name }}</th>
            <td>
              <div v-for="(v, i) in values" :key="i" class="val">{{ v }}</div>
            </td>
          </tr>
        </tbody>
      </table>
    </section>
    <p v-if="!Object.keys(headers).length" class="empty">
      No response headers captured. Reload the page with the extension installed
      so chrome.webRequest can see the real response.
    </p>
  </div>
</template>

<style scoped>
.stack {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.g-head {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  margin-bottom: var(--space-2);
}

.g-title {
  margin: 0;
  font-size: var(--font-size-xs);
  text-transform: uppercase;
  letter-spacing: var(--letter-wide);
  color: var(--color-text-muted);
  font-weight: var(--font-weight-semibold);
}

.g-hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-faint);
}

.tbl {
  width: 100%;
  border-collapse: collapse;
  font-family: var(--font-mono);
  font-size: var(--font-size-xs);
  background: var(--color-bg-surface-2);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.tbl th,
.tbl td {
  text-align: left;
  padding: var(--space-2) var(--space-3);
  vertical-align: top;
  border-bottom: 1px solid var(--color-border-subtle);
}

.tbl tr:last-child th,
.tbl tr:last-child td {
  border-bottom: none;
}

.tbl th {
  width: 40%;
  color: var(--color-text-muted);
  font-weight: var(--font-weight-medium);
}

.tbl td {
  color: var(--color-text);
  word-break: break-word;
}

.val + .val {
  margin-top: var(--space-1);
}

.empty {
  padding: var(--space-4);
  background: var(--color-bg-surface-2);
  border-radius: var(--radius-md);
  border: 1px dashed var(--color-border-subtle);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  margin: 0;
  line-height: var(--line-height-snug);
}
</style>
