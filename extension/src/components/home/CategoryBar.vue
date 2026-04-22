<script setup lang="ts">
/**
 * A single row in the per-category health list. Visualises one category's
 * issue severity distribution as a horizontal bar, with the category's
 * accent pulled straight from a CSS var.
 */
import type { CategoryScore } from '@/lib/scoring';

const props = defineProps<{ entry: CategoryScore }>();

function pct(n: number) {
  return props.entry.total === 0 ? 0 : (n / props.entry.total) * 100;
}
</script>

<template>
  <button class="row" :style="{ '--cat': `var(${entry.cssVar})` }">
    <div class="left">
      <span class="dot" />
      <span class="label">{{ entry.label }}</span>
    </div>
    <div class="bar">
      <span class="seg seg-crit" :style="{ width: `${pct(entry.critical)}%` }" />
      <span class="seg seg-warn" :style="{ width: `${pct(entry.warning)}%` }" />
      <span class="seg seg-info" :style="{ width: `${pct(entry.info)}%` }" />
    </div>
    <div class="score">
      <span class="num">{{ entry.score }}</span>
      <span class="slash">/100</span>
    </div>
  </button>
</template>

<style scoped>
.row {
  display: grid;
  grid-template-columns: minmax(0, 130px) 1fr auto;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  text-align: left;
  transition: background var(--duration-fast) var(--ease-out);
  width: 100%;
}

.row:hover {
  background: var(--color-bg-hover);
}

.left {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--cat);
  flex-shrink: 0;
}

.label {
  font-size: var(--font-size-sm);
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.bar {
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--color-bg-surface-3);
  display: flex;
  overflow: hidden;
}

.seg {
  height: 100%;
  transition: width var(--duration-medium) var(--ease-out);
}
.seg-crit { background: var(--color-critical); }
.seg-warn { background: var(--color-warning); }
.seg-info { background: var(--color-info); }

.score {
  display: flex;
  align-items: baseline;
  gap: 2px;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: var(--font-size-sm);
}

.num {
  color: var(--color-text);
  font-weight: var(--font-weight-medium);
}

.slash {
  color: var(--color-text-faint);
  font-size: var(--font-size-xs);
}
</style>
