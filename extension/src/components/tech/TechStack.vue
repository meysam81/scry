<script setup lang="ts">
/**
 * Groups detected technologies by category. Categories display in the same
 * order everywhere. Tech strings come through the pipeline as "category:Name"
 * so we can split them without a lookup table.
 */
import { computed } from "vue";

const props = defineProps<{ technologies: string[] }>();

type Grouped = Record<string, string[]>;

const CATEGORY_LABEL: Record<string, string> = {
  framework: "Framework",
  cms: "CMS",
  analytics: "Analytics",
  "tag-manager": "Tag Managers",
  cdn: "CDN",
  server: "Server",
  ecommerce: "Commerce",
  "ui-library": "UI Library",
  advertising: "Advertising",
  font: "Fonts",
  search: "Search",
};

const ORDER = [
  "framework",
  "cms",
  "ui-library",
  "analytics",
  "tag-manager",
  "cdn",
  "server",
  "ecommerce",
  "search",
  "advertising",
  "font",
];

const grouped = computed<Grouped>(() => {
  const g: Grouped = {};
  for (const t of props.technologies) {
    const [cat, ...rest] = t.split(":");
    if (!cat || rest.length === 0) continue;
    (g[cat] ??= []).push(rest.join(":"));
  }
  for (const k of Object.keys(g)) g[k].sort();
  return g;
});

const categories = computed(() =>
  ORDER.filter((c) => grouped.value[c]?.length),
);
</script>

<template>
  <div v-if="categories.length" class="stack">
    <section v-for="cat in categories" :key="cat" class="group">
      <h4 class="group-title">{{ CATEGORY_LABEL[cat] ?? cat }}</h4>
      <div class="pills">
        <span v-for="name in grouped[cat]" :key="name" class="pill">
          {{ name }}
        </span>
      </div>
    </section>
  </div>
  <p v-else class="empty">No recognised technologies detected on this page.</p>
</template>

<style scoped>
.stack {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.group-title {
  margin: 0 0 var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--color-text-faint);
  text-transform: uppercase;
  letter-spacing: var(--letter-wide);
  font-weight: var(--font-weight-semibold);
}

.pills {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.pill {
  display: inline-flex;
  align-items: center;
  padding: 4px var(--space-3);
  border-radius: var(--radius-full);
  background: var(--color-bg-surface-2);
  border: 1px solid var(--color-border-subtle);
  color: var(--color-text);
  font-size: var(--font-size-sm);
  line-height: 1;
}

.empty {
  margin: 0;
  padding: var(--space-4);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
  background: var(--color-bg-surface-2);
  border-radius: var(--radius-md);
  border: 1px dashed var(--color-border-subtle);
}
</style>
