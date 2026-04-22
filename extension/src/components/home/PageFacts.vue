<script setup lang="ts">
/**
 * Editorial-feeling stat panel: the at-a-glance numeric facts about the
 * current page. Mirrors the density of ahrefs' SERP overview while staying
 * on-brand with airy whitespace + display serif for big numbers.
 */
import { computed } from 'vue';
import type { PageSnapshot } from '@/schemas/page';

const props = defineProps<{ snapshot: PageSnapshot | null }>();

interface Fact {
  label: string;
  value: string | number;
  hint?: string;
}

const facts = computed<Fact[]>(() => {
  const m = props.snapshot?.html_meta;
  if (!m) return [];
  return [
    { label: 'Words', value: m.word_count.toLocaleString(), hint: 'Body text length' },
    { label: 'Headings', value: `${m.h1_count} H1 · ${m.h2_count} H2` },
    { label: 'Images', value: m.img_count, hint: `${m.img_without_alt} missing alt` },
    { label: 'Links', value: m.link_count, hint: `${m.external_link_count} external` },
    { label: 'Schemas', value: m.json_ld_count, hint: 'JSON-LD blocks' },
    {
      label: 'Tech',
      value: props.snapshot?.technologies.length ?? 0,
      hint: 'Detected stack items',
    },
  ];
});
</script>

<template>
  <div class="grid">
    <div v-for="f in facts" :key="f.label" class="fact">
      <div class="label">{{ f.label }}</div>
      <div class="value">{{ f.value }}</div>
      <div v-if="f.hint" class="hint">{{ f.hint }}</div>
    </div>
  </div>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-3);
}

.fact {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-3);
  background: var(--color-bg-surface-2);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-md);
}

.label {
  font-size: var(--font-size-xs);
  text-transform: uppercase;
  letter-spacing: var(--letter-wide);
  color: var(--color-text-faint);
}

.value {
  font-family: var(--font-display);
  font-size: var(--font-size-xl);
  color: var(--color-text);
  line-height: var(--line-height-tight);
  letter-spacing: var(--letter-tight);
}

.hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-faint);
}
</style>
