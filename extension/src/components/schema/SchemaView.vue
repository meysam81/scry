<script setup lang="ts">
/**
 * Renders structured-data findings: OG/Twitter card meta + JSON-LD block
 * count. Pairs with the issues tab for Schema.org deep-validation details.
 */
import { computed } from "vue";
import type { PageSnapshot } from "@/schemas/page";

const props = defineProps<{ snapshot: PageSnapshot | null }>();

const meta = computed(() => props.snapshot?.html_meta);

const ogRows = computed(() => Object.entries(meta.value?.og ?? {}).sort());
const twitterRows = computed(() =>
  Object.entries(meta.value?.twitter ?? {}).sort(),
);
</script>

<template>
  <div v-if="meta" class="stack">
    <section class="block">
      <h4 class="title">OpenGraph</h4>
      <table v-if="ogRows.length" class="tbl">
        <tbody>
          <tr v-for="[k, v] in ogRows" :key="k">
            <th>og:{{ k }}</th>
            <td>{{ v }}</td>
          </tr>
        </tbody>
      </table>
      <p v-else class="empty">No og:* tags on this page.</p>
    </section>

    <section class="block">
      <h4 class="title">Twitter Card</h4>
      <table v-if="twitterRows.length" class="tbl">
        <tbody>
          <tr v-for="[k, v] in twitterRows" :key="k">
            <th>twitter:{{ k }}</th>
            <td>{{ v }}</td>
          </tr>
        </tbody>
      </table>
      <p v-else class="empty">No twitter:* tags on this page.</p>
    </section>

    <section class="block">
      <h4 class="title">JSON-LD</h4>
      <p class="big">
        <span class="num">{{ meta.json_ld_count }}</span>
        <span class="unit">blocks</span>
      </p>
      <p class="hint">
        Deep Schema.org validation runs in the core engine; findings appear
        under the Issues tab as <code>deep-structured-data</code> checks.
      </p>
    </section>
  </div>
  <p v-else class="empty">Run an audit to see structured data.</p>
</template>

<style scoped>
.stack {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.title {
  margin: 0 0 var(--space-2);
  font-size: var(--font-size-xs);
  text-transform: uppercase;
  letter-spacing: var(--letter-wide);
  color: var(--color-text-muted);
  font-weight: var(--font-weight-semibold);
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
  white-space: nowrap;
}

.tbl td {
  color: var(--color-text);
  word-break: break-word;
}

.empty {
  margin: 0;
  padding: var(--space-3);
  background: var(--color-bg-surface-2);
  border-radius: var(--radius-md);
  border: 1px dashed var(--color-border-subtle);
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.big {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  margin: 0 0 var(--space-2);
}

.num {
  font-family: var(--font-display);
  font-size: var(--font-size-3xl);
  line-height: 1;
  color: var(--color-text);
}

.unit {
  font-size: var(--font-size-sm);
  color: var(--color-text-muted);
}

.hint {
  margin: 0;
  font-size: var(--font-size-xs);
  color: var(--color-text-faint);
  line-height: var(--line-height-snug);
}

code {
  background: var(--color-bg-surface-3);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: 0.9em;
}
</style>
